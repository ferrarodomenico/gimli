// internal/config/load.go
package config

import (
	"fmt"
	"os"
	"strconv"
)

// LoadConfig reads all configuration from environment variables and returns a populated Config.
// Missing required variables cause a fatal exit.
func LoadConfig() Config {

	return Config{
		RabbitMQURL:    loadEnv[string](RABBIT_URL_ENV),
		WorkerCount:    loadEnv[int](WORKER_COUNT_ENV),
		ReconnectDelay: loadEnv[uint8](RECONNECT_DELAY_ENV),
		PrefetchCount:  loadEnv[int](PREFETCH_COUNT_ENV),
		MaxRetries:     loadEnv[int](MAX_RETRIES_ENV),
		MainQueue:      loadBaseQueue(MAIN_PREFIX),
		RetryQueue:     loadRetryQueue(),
		DLQ:            loadBaseQueue(DLQ_PREFIX),
		SMTP:           loadSMTP(),
	}
}

func loadBaseQueue(prefix string) BaseQueueSettings {
	return BaseQueueSettings{
		Name:       loadEnv[string](prefix + QUEUE_NAME_SUFFIX),
		Exchange:   loadEnv[string](prefix + QUEUE_EXCHANGE_SUFFIX),
		RoutingKey: loadEnv[string](prefix + QUEUE_ROUTING_KEY_SUFFIX),
	}
}

func loadRetryQueue() RetryQueueSettings {
	return RetryQueueSettings{
		Name:        loadEnv[string](RETRY_PREFIX + QUEUE_NAME_SUFFIX),
		Exchange:    loadEnv[string](RETRY_PREFIX + QUEUE_EXCHANGE_SUFFIX),
		DelayFactor: loadEnv[float32](RETRY_PREFIX + QUEUE_DELAY_FACTOR_SUFFIX),
	}
}

func loadSMTP() SMTPConfig {
	return SMTPConfig{
		Host:     loadEnv[string](SMTP_HOST_ENV),
		Port:     loadEnv[int](SMTP_PORT_ENV),
		Username: loadEnv[string](SMTP_USERNAME_ENV),
		Password: loadEnv[string](SMTP_PASSWORD_ENV),
		AllowTLS: loadEnv[bool](SMTP_ALLOW_TLS_ENV),
		PoolSize: loadEnv[int](SMTP_POOL_SIZE_ENV),
		Timeout:  loadEnv[uint8](SMTP_TIMEOUT_ENV),
	}
}

func loadEnv[T any](key string) T {
	raw, ok := os.LookupEnv(key)
	if !ok {
		exitf("missing required environment variable: %s", key)
	}

	v, err := parse[T](raw)
	if err != nil {
		exitf("invalid value for %s: %v", key, err)
	}

	return v
}

func parse[T any](raw string) (T, error) {
	var zero T
	switch any(zero).(type) {
	case string:
		return any(raw).(T), nil
	case int:
		n, err := strconv.Atoi(raw)
		return any(n).(T), err
	case uint8:
		n, err := strconv.ParseUint(raw, 10, 8)
		return any(uint8(n)).(T), err
	case float32:
		f, err := strconv.ParseFloat(raw, 32)
		return any(float32(f)).(T), err
	case bool:
		b, err := strconv.ParseBool(raw)
		return any(b).(T), err
	default:
		return zero, fmt.Errorf("unsupported type %T", zero)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "config error: "+format+"\n", args...)
	os.Exit(1)
}
