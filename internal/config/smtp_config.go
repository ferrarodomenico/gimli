// internal/config/smtp_config.go
package config

import "fmt"

const (
	SMTP_HOST_ENV      = "SMTP_HOST"
	SMTP_PORT_ENV      = "SMTP_PORT"
	SMTP_USERNAME_ENV  = "SMTP_USERNAME"
	SMTP_PASSWORD_ENV  = "SMTP_PASSWORD"
	SMTP_ALLOW_TLS_ENV = "SMTP_ALLOW_TLS"
	SMTP_POOL_SIZE_ENV = "SMTP_POOL_SIZE"
	SMTP_TIMEOUT_ENV   = "SMTP_TIMEOUT"
)

// SMTPConfig holds connection parameters for the SMTP mail server.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	AllowTLS bool
	PoolSize int
	Timeout  uint8 // Timeout in seconds
}

// String returns a log-safe representation of the SMTP config with the password masked.
func (c SMTPConfig) String() string {
	masked := "***"
	if c.Password == "" {
		masked = "(empty)"
	}
	return fmt.Sprintf("SMTPConfig{Host:%q Port:%d Username:%q Password:%s AllowTLS:%v PoolSize:%d Timeout:%d}",
		c.Host, c.Port, c.Username, masked, c.AllowTLS, c.PoolSize, c.Timeout)
}
