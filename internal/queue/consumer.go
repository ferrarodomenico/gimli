// internal/queue/consumer.go
package queue

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"sync/atomic"

	"github.com/ferrarodomenico/gimli/internal/config"
	"github.com/ferrarodomenico/gimli/internal/mail"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
)

type Consumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	cfg         config.Config
	mailService *mail.Service
	logger      zerolog.Logger
	logWriter   diode.Writer
	isReady     atomic.Bool
}

func NewConsumer(cfg config.Config) (*Consumer, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.Qos(
		cfg.PrefetchCount,
		0,
		false,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set qos with prefetch %d: %w", cfg.PrefetchCount, err)
	}

	mailSvc := mail.NewService(cfg.SMTP)

	logWriter := diode.NewWriter(os.Stdout, 1000, 10*time.Millisecond, func(missed int) {
		fmt.Fprintf(os.Stderr, "logger dropped %d messages\n", missed)
	})
	logger := zerolog.New(logWriter).With().Timestamp().Logger()

	return &Consumer{conn: conn, channel: ch, cfg: cfg, mailService: mailSvc, logger: logger, logWriter: logWriter}, nil
}

func (c *Consumer) Close() {
	c.channel.Close()
	c.conn.Close()
	c.mailService.Close()
	c.logWriter.Close()
}

func (c *Consumer) IsReady() bool {
	return c.isReady.Load()
}

func (c *Consumer) Setup() error {
	if err := c.declareQueue(c.cfg.MainQueue.BaseQueueSettings); err != nil {
		return fmt.Errorf("setup main queue: %w", err)
	}

	if err := c.declareRetryQueue(); err != nil {
		return fmt.Errorf("setup retry queue: %w", err)
	}

	if err := c.declareQueue(c.cfg.DLQ); err != nil {
		return fmt.Errorf("setup dlq: %w", err)
	}

	return nil
}

func (c *Consumer) declareQueue(q config.BaseQueueSettings) error {
	return c.declare(q, nil)
}

func (c *Consumer) declareRetryQueue() error {
	q := config.BaseQueueSettings{
		Name:       c.cfg.RetryQueue.Name,
		Exchange:   c.cfg.RetryQueue.Exchange,
		RoutingKey: c.cfg.MainQueue.RoutingKey,
	}
	return c.declare(q, &c.cfg.MainQueue.BaseQueueSettings)
}

func (c *Consumer) declare(q config.BaseQueueSettings, dlx *config.BaseQueueSettings) error {
	if err := c.channel.ExchangeDeclare(
		q.Exchange, "direct", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare exchange %s: %w", q.Exchange, err)
	}

	var args amqp.Table
	if dlx != nil {
		args = amqp.Table{
			"x-dead-letter-exchange":    dlx.Exchange,
			"x-dead-letter-routing-key": dlx.RoutingKey,
		}
	}

	if _, err := c.channel.QueueDeclare(
		q.Name, true, false, false, false, args,
	); err != nil {
		return fmt.Errorf("declare queue %s: %w", q.Name, err)
	}

	if err := c.channel.QueueBind(
		q.Name, q.RoutingKey, q.Exchange, false, nil,
	); err != nil {
		return fmt.Errorf("bind queue %s: %w", q.Name, err)
	}

	return nil
}

func (c *Consumer) Start(workerCount int) error {
	deliveries, err := c.channel.Consume(
		c.cfg.MainQueue.Name, // queue
		"gimli-worker",       // consumer tag
		false,                // auto-ack = false
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	c.isReady.Store(true)

	closeChan := make(chan *amqp.Error, 1)
	c.conn.NotifyClose(closeChan)
	go c.monitorConnection(closeChan, workerCount)

	taskChan := make(chan amqp.Delivery, workerCount)

	for i := 1; i <= workerCount; i++ {
		go c.worker(i, taskChan)
	}

	go func() {
		for d := range deliveries {
			taskChan <- d
		}
		close(taskChan)
	}()

	return nil
}

func (c *Consumer) worker(id int, tasks <-chan amqp.Delivery) {
	for msg := range tasks {
		var m mail.Mail
		if err := json.Unmarshal(msg.Body, &m); err != nil {
			c.logger.Error().Err(err).Int("worker", id).Msg("Unmarshal failed")
			_ = msg.Nack(false, false)
			continue
		}

		logger := c.logger.With().Int("worker", id).Str("to", m.To).Str("subject", m.Subject).Logger()

		if !m.Parsed {
			c.mailService.Parse(&m)
		}
		if !m.Minified {
			c.mailService.Minify(&m)
		}

		if err := c.processMail(&m); err != nil {
			logger.Error().Err(err).Msg("Send failed")
			c.handleFailure(&m, msg, logger)
			continue
		}

		_ = msg.Ack(false)
	}
}

func (c *Consumer) monitorConnection(closeChan <-chan *amqp.Error, workerCount int) {
	err := <-closeChan
	if err == nil {
		c.isReady.Store(false)
		return
	}

	c.isReady.Store(false)
	c.logger.Error().Err(err).Msg("RabbitMQ connection closed unexpectedly, attempting reconnection")

	for {
		time.Sleep(time.Duration(c.cfg.ReconnectDelay) * time.Second)

		conn, dialErr := amqp.Dial(c.cfg.RabbitMQURL)
		if dialErr != nil {
			continue
		}

		ch, chErr := conn.Channel()
		if chErr != nil {
			conn.Close()
			continue
		}

		if qosErr := ch.Qos(c.cfg.WorkerCount, 0, false); qosErr != nil {
			ch.Close()
			conn.Close()
			continue
		}

		c.conn = conn
		c.channel = ch

		if setupErr := c.Setup(); setupErr != nil {
			c.logger.Error().Err(setupErr).Msg("Reconnection setup failed")
			continue
		}

		c.Start(workerCount)
		c.logger.Info().Msg("RabbitMQ reconnection successful")
		break
	}
}

func (c *Consumer) processMail(m *mail.Mail) error {
	return c.mailService.Send(m)
}

func (c *Consumer) handleFailure(m *mail.Mail, msg amqp.Delivery, logger zerolog.Logger) {
	m.RetryCount++

	if m.RetryCount > c.cfg.MainQueue.MaxRetries {
		c.publishMsg(c.cfg.DLQ.Exchange, c.cfg.DLQ.RoutingKey, m, "")
		logger.Warn().Int("retry", m.RetryCount).Msg("Dead-lettered")
		_ = msg.Ack(false)
		return
	}

	delay := int(float32(c.cfg.RetryQueue.DelayFactor) * float32(math.Pow(2, float64(m.RetryCount))) * 1000)
	c.publishMsg(
		c.cfg.RetryQueue.Exchange,
		c.cfg.MainQueue.RoutingKey,
		m,
		strconv.Itoa(delay),
	)
	logger.Warn().Int("retry", m.RetryCount).Int("delay_ms", delay).Msg("Requeued for retry")
	_ = msg.Ack(false)
}

func (c *Consumer) publishMsg(exchange, routingKey string, m *mail.Mail, expiration string) {
	body, _ := json.Marshal(m)
	_ = c.channel.Publish(
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Expiration:  expiration,
		},
	)
}
