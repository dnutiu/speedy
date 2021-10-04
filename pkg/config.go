package pkg

import (
	"errors"
	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-json"
	"github.com/spf13/viper"
	"math"
)

// Configuration holds all the application's configurable settings.
type Configuration struct {
	// LoggingLevel is the logging level.
	LoggingLevel string `json:"logging_level"`
	// SentryDSN is the DSN used by Sentry, for reporting errors.
	SentryDSN string `json:"sentry_dsn"`
	// KafkaPollingGoroutines is the number of goroutines that will poll Kafka for new messages.
	KafkaPollingGoroutines int `json:"kafka_polling_goroutines"`
	// KafkaPollingTimeoutMs is the timeout in milliseconds for the message poll().
	KafkaPollingTimeoutMs int `json:"kafka_polling_timeout_ms"`
	// KafkaBoostrapServers is a string of comma separated boostrap servers.
	KafkaBoostrapServers string `json:"kafka_boostrap_servers"`
	// KafkaGroupId is the Kafka consumer group id.
	KafkaGroupId string `json:"kafka_group_id"`
	// SubscribeTopics is the list of topics or patterns to subscribe to.
	SubscribeTopics []string `json:"subscribe_topics"`
	// LokiPushUrl is the full URL of the Loki push API endpoint.
	LokiPushUrl string `json:"loki_push_url"`
	// LokiPushMode is the mode used to push data to Loki, http or proto.
	LokiPushMode string `json:"loki_push_mode"`
	// BufferMaxBatchSize is the batch size that will be sent to Loki.
	BufferMaxBatchSize int `json:"buffer_max_batch_size"`
	// BufferMaxBytesSize is max buffer size in bytes uncompressed and unserialized that will be sent to Loki.
	BufferMaxBytesSize int `json:"buffer_max_bytes_size"`
	// KafkaOffsetReset is analogous to https://kafka.apache.org/documentation/#consumerconfigs_auto.offset.reset
	KafkaOffsetReset string `json:"kafka_offset_reset"`
}

// ToPrettyJson transforms the current configuration to a pretty json.
func (c Configuration) ToPrettyJson() string {
	prettyJson, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		SugaredLogger.Error("failed to convert config to pretty json")
		return ""
	}
	return string(prettyJson)
}

// ViperConfigurator is a convenient wrapper over Viper.
type ViperConfigurator struct {
	viper         *viper.Viper
	configuration Configuration
}

func NewViperConfigurator() (*ViperConfigurator, error) {
	viperInstance := viper.New()
	viperInstance.SetConfigName("config")
	viperInstance.SetConfigType("json")
	viperInstance.AddConfigPath("$HOME/.speedy")
	viperInstance.AddConfigPath(".")
	viperInstance.SetEnvPrefix("SG")
	viperInstance.AutomaticEnv()
	if err := viperInstance.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			SugaredLogger.Warn("Config file not found")
		} else {
			SugaredLogger.Error("Error loading config file.")
			sentry.CaptureException(err)
			return nil, err
		}
	}

	viperConfigurator := ViperConfigurator{viperInstance, Configuration{}}
	err := viperConfigurator.loadConfig()
	if err != nil {
		SugaredLogger.Error(err)
		sentry.CaptureException(err)
		return nil, err
	}

	return &viperConfigurator, nil
}

// GetConfig returns a copy of the Configuration data structure.
func (v *ViperConfigurator) GetConfig() Configuration {
	return v.configuration
}

// loadConfig loads configuration from viper onto the internal data structures.
func (v *ViperConfigurator) loadConfig() error {

	v.configuration.KafkaBoostrapServers = v.viper.GetString("kafka_bootstrap_servers")
	if v.configuration.KafkaBoostrapServers == "" {
		return errors.New("kafka_bootstrap_servers is empty")
	}

	v.configuration.KafkaGroupId = v.viper.GetString("kafka_group_id")
	if v.configuration.KafkaGroupId == "" {
		return errors.New("kafka_group_id is empty")
	}

	v.configuration.LokiPushUrl = v.viper.GetString("loki_push_url")
	if v.configuration.LokiPushUrl == "" {
		return errors.New("loki_push_url is empty")
	}

	v.configuration.SubscribeTopics = v.viper.GetStringSlice("subscribe_topics")
	if len(v.configuration.SubscribeTopics) == 0 {
		return errors.New("subscribe_topics is empty")
	}

	v.viper.SetDefault("buffer_max_batch_size", 10_000)
	v.configuration.BufferMaxBatchSize = v.viper.GetInt("buffer_max_batch_size")

	v.viper.SetDefault("buffer_max_bytes_size", math.MaxInt32)
	v.configuration.BufferMaxBytesSize = v.viper.GetInt("buffer_max_bytes_size")

	v.viper.SetDefault("loki_push_mode", "http")
	v.configuration.LokiPushMode = v.viper.GetString("loki_push_mode")

	v.viper.SetDefault("kafka_offset_reset", "earliest")
	v.configuration.KafkaOffsetReset = v.viper.GetString("kafka_offset_reset")

	v.viper.SetDefault("logging_level", "info")
	v.configuration.LoggingLevel = v.viper.GetString("logging_level")

	v.viper.SetDefault("kafka_polling_goroutines", 5)
	v.configuration.KafkaPollingGoroutines = v.viper.GetInt("kafka_polling_goroutines")

	v.viper.SetDefault("kafka_polling_timeout_ms", 30_000)
	v.configuration.KafkaPollingTimeoutMs = v.viper.GetInt("kafka_polling_timeout_ms")

	v.viper.SetDefault("sentry_dsn", "")
	v.configuration.SentryDSN = v.viper.GetString("sentry_dsn")

	return nil
}
