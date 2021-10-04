# Speedy

Speedy is a Go streaming program that consumes data from Kafka topics based on a pattern and writes it to Loki.

### Configuration

Configuration is done via configuration file `config.json`, additionally you may override values using ENVIRONMENT variables.

Environment variables are prefixed with `SG_` and are directly mapped to the configuration file, for example:

`SG_KAFKA_GROUP_ID=override` will override `kafka_group_id` from `config.json`.

#### Example configuration:
```json
{
  "logging_level": "info",
  "kafka_bootstrap_servers": "kafkaboostrap:9092",
  "sentry_dsn": "",
  "kafka_group_id": "speedy",
  "kafka_offset_reset": "latest",
  "subscribe_topics": ["^topic\\.pattern.+"],
  "buffer_max_batch_size": 1000,
  "kafka_polling_goroutines": 10,
  "kafka_polling_timeout_ms": 30000,
  "loki_push_url": "http://loki-distributor.loki:3100/loki/api/v1/push",
  "loki_push_mode": "proto"
}
```

## Custom librdkafka build

To add support for regex negative lookahead expression a custom libdrdkafka build was necessary. 
Please see the `Dockerfile.librdkafka` file.

### Example deployment on Kubernetes

**deploy.yaml**

```yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: speedy-consumer-all
    labels:
      app: speedy
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: speedy
    template:
      metadata:
        labels:
          app: speedy
      spec:
        containers:
          - name: speedy
            imagePullPolicy: Always
            image: xxx.dkr.ecr.eu-central-1.amazonaws.com/speedy:latest
            resources:
              requests:
                memory: "512Mi"
                cpu: "1000m"
              limits:
                memory: "2096Mi"
                cpu: "1000m"
            volumeMounts:
              - name: speedy-config-all
                mountPath: /root/.speedy
        volumes:
          - name: speedy-config-all
            configMap:
              # All configs.
              name: speedy-config-all
```

**configmap.yaml**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: speedy-config-all
data:
  config.json: |-
    {
      "kafka_bootstrap_servers": "xxx.kafka.eu-central-1.amazonaws.com:9092",
      "sentry_dsn": "",
      "kafka_group_id": "speedy",
      "subscribe_topics": ["^.+"],
      "loki_push_url": "http://loki-distributor.loki:3100/loki/api/v1/push",
      "buffer_max_batch_size": 10000,
      "loki_push_mode": "proto"
    }
```