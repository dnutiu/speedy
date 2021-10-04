package main

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-json"
	"os"
	"os/signal"
	"speedy/pkg"
	"sync"
)

func main() {
	configurator, err := pkg.NewViperConfigurator()
	if err != nil {
		panic(err)
	}
	// Grab hostName
	hostName, err := os.Hostname()
	if err != nil {
		pkg.SugaredLogger.Warn("Failed to get hostname! Using generic name.")
		hostName = "Speedy"
	}
	pkg.SugaredLogger.Infof("Using hostname %s", hostName)

	// Init logging
	config := configurator.GetConfig()
	pkg.InitLoggingWithParams(config.LoggingLevel, "console", "")

	// Init sentry
	err = sentry.Init(sentry.ClientOptions{
		// Either set your DSN here or set the SENTRY_DSN environment variable.
		Dsn: config.SentryDSN,
		// Either set environment and release here or set the SENTRY_ENVIRONMENT
		// and SENTRY_RELEASE environment variables.
		Environment: hostName,
		Release:     "speedy-gonzales",
	})
	if err != nil {
		pkg.SugaredLogger.Error("failed to init Sentry.")
	}

	// Init kafka
	pkg.SugaredLogger.Infof("Using config:\n %s", config.ToPrettyJson())
	kafkaConsumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaBoostrapServers,
		"group.id":          config.KafkaGroupId,
		"auto.offset.reset": config.KafkaOffsetReset,
		"socket.timeout.ms": "300000",
	})

	if err != nil {
		panic(err)
	}

	err = kafkaConsumer.SubscribeTopics(config.SubscribeTopics, nil)
	if err != nil {
		pkg.SugaredLogger.Errorf("failed to subscribe: %s", err)
		sentry.CaptureException(err)
		panic(err)
	}
	defer func(c *kafka.Consumer) {
		err := c.Close()
		if err != nil {
			panic(err)
		}
	}(kafkaConsumer)
	pkg.SugaredLogger.Info("Initializing")

	// Init Sink & Pusher
	var lokiClient = pkg.LokiClientFactoryCreate(config.LokiPushMode, config.LokiPushUrl)
	var speedyPusher = pkg.NewPusher(lokiClient, config.BufferMaxBatchSize, config.BufferMaxBytesSize)
	go speedyPusher.RunForever()
	isRunning := true
	var waitGroup sync.WaitGroup
	for i := 0; i < config.KafkaPollingGoroutines; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for isRunning {
				ev := kafkaConsumer.Poll(config.KafkaPollingTimeoutMs)

				switch event := ev.(type) {
				case kafka.AssignedPartitions:
					err := kafkaConsumer.Assign(event.Partitions)
					if err != nil {
						pkg.SugaredLogger.Error(err)
						sentry.CaptureException(err)
						return
					}
				case kafka.RevokedPartitions:
					err := kafkaConsumer.Unassign()
					if err != nil {
						pkg.SugaredLogger.Error(err)
						sentry.CaptureException(err)
						return
					}
				case *kafka.Message:
					var messageMap = make(map[string]interface{})
					err := json.Unmarshal(event.Value, &messageMap)
					if err != nil {
						pkg.SugaredLogger.Error(err)
						continue
					}
					flattenMap := pkg.FlattenMap(messageMap)
					flattenMapString, err := json.Marshal(flattenMap)
					if err != nil {
						pkg.SugaredLogger.Error(err)
						continue
					}

					labelsMap := map[string]string{
						"key": *event.TopicPartition.Topic,
					}

					clientId := (*flattenMap)["clientID"]
					clientIdStr, ok := clientId.(string)
					if ok {
						labelsMap["clientId"] = clientIdStr
					}

					// Size of clientId key and val
					labelsSize := 8 + len(clientIdStr)

					speedyPusher.DataChannel <- pkg.LokiStream{
						Labels: labelsMap,
						Values: [][]string{{"", string(flattenMapString)}},
						Size:   len(event.Value) + labelsSize,
					}
				case kafka.PartitionEOF:
					pkg.SugaredLogger.Info()
				case kafka.Error:
					if event.Code() == kafka.ErrTimedOut {
						pkg.SugaredLogger.Debugf("Consumer error: %v\n", err)
					} else {
						// The client will automatically try to recover from all errors.
						pkg.SugaredLogger.Warnf("Consumer error: %v\n", err)
						sentry.CaptureException(err)
					}
				default:
				}
			}
		}()
	}
	go func() {
		// Handle SIGINT
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		// Block until a signal is received.
		<-c
		pkg.SugaredLogger.Info("Received SIGINT, shutting down.")
		isRunning = false
	}()
	waitGroup.Wait()
	pkg.SugaredLogger.Info("Exiting.")
}
