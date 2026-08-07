// internal/config/queue_config.go
package config

import (
	"fmt"
	"net/url"
)

const (
	RABBIT_URL_ENV      = "RABBITMQ_URL"
	WORKER_COUNT_ENV    = "WORKER_COUNT"
	RECONNECT_DELAY_ENV = "RECONNECT_DELAY" // in seconds
	PREFETCH_COUNT_ENV  = "PREFETCH_COUNT"
	MAX_RETRIES_ENV     = "MAX_RETRIES"
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
	QUEUE_DELAY_FACTOR_SUFFIX = "QUEUE_DELAY_FACTOR"
)

// Config holds all runtime configuration sourced from environment variables.
type Config struct {
	RabbitMQURL    string
	WorkerCount    int
	ReconnectDelay uint8
	PrefetchCount  int
	MaxRetries     int
	MainQueue      BaseQueueSettings
	RetryQueue     RetryQueueSettings
	DLQ            BaseQueueSettings
	SMTP           SMTPConfig
}

// BaseQueueSettings defines the minimum parameters for a RabbitMQ queue.
type BaseQueueSettings struct {
	Name       string
	Exchange   string
	RoutingKey string
}

// RetryQueueSettings configures the retry queue with exponential backoff delay factor.
type RetryQueueSettings struct {
	Name        string
	Exchange    string
	DelayFactor float32
}

func maskURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(invalid)"
	}
	if u.User != nil {
		p, _ := u.User.Password()
		if p != "" {
			u.User = url.UserPassword(u.User.Username(), "***")
		}
	}
	return u.String()
}

// String returns a log-safe representation of the config with masked credentials.
func (c Config) String() string {
	return fmt.Sprintf("Config{RabbitMQURL:%q WorkerCount:%d ReconnectDelay:%d PrefetchCount:%d MaxRetries:%d MainQueue:%+v RetryQueue:%+v DLQ:%+v SMTP:%s}",
		maskURL(c.RabbitMQURL), c.WorkerCount, c.ReconnectDelay, c.PrefetchCount, c.MaxRetries, c.MainQueue, c.RetryQueue, c.DLQ, c.SMTP)
}
