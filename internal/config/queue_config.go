// internal/config/queue_config.go
package config

const (
	RABBIT_URL_ENV      = "RABBITMQ_URL"
	WORKER_COUNT_ENV    = "WORKER_COUNT"
	RECONNECT_DELAY_ENV = "RECONNECT_DELAY" // in seconds
	PREFETCH_COUNT_ENV  = "PREFETCH_COUNT"
)

const (
	MAIN_PREFIX  = "MAIN_"
	RETRY_PREFIX = "RETRY_"
	DLQ_PREFIX   = "DLQ_"
)

const (
	QUEUE_NAME_SUFFIX         = "QUEUE_NAME"
	QUEUE_EXCHANGE_SUFFIX     = "QUEUE_EXCHANGE"
	QUEUE_ROUTING_KEY_SUFFIX  = "QUEUE_ROUTING_KEY"
	QUEUE_MAX_RETRIES_SUFFIX  = "QUEUE_MAX_RETRIES"
	QUEUE_DELAY_FACTOR_SUFFIX = "QUEUE_DELAY_FACTOR"
)

type Config struct {
	RabbitMQURL    string
	WorkerCount    int
	ReconnectDelay uint8
	PrefetchCount  int
	MainQueue      MainQueueSettings
	RetryQueue     RetryQueueSettings
	DLQ            BaseQueueSettings
	SMTP           SMTPConfig
}

type BaseQueueSettings struct {
	Name       string
	Exchange   string
	RoutingKey string
}

type MainQueueSettings struct {
	BaseQueueSettings
	MaxRetries int
}

type RetryQueueSettings struct {
	Name        string
	Exchange    string
	DelayFactor float32
}
