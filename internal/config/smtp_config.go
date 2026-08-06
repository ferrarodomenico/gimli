// internal/config/smtp_config.go
package config

const (
	SMTP_HOST_ENV      = "SMTP_HOST"
	SMTP_PORT_ENV      = "SMTP_PORT"
	SMTP_USERNAME_ENV  = "SMTP_USERNAME"
	SMTP_PASSWORD_ENV  = "SMTP_PASSWORD"
	SMTP_ALLOW_TLS_ENV = "SMTP_ALLOW_TLS"
	SMTP_POOL_SIZE_ENV = "SMTP_POOL_SIZE"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	AllowTLS bool
	PoolSize int
}
