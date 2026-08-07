package mail

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	min "github.com/tdewolff/minify/v2"
	mincss "github.com/tdewolff/minify/v2/css"
	minhtml "github.com/tdewolff/minify/v2/html"
	gomail "github.com/wneessen/go-mail"

	"errors"

	"github.com/ferrarodomenico/gimli/internal/config"
)

type Mail struct {
	From       string
	To         string
	Subject    string
	Body       string
	Vars       map[string]string
	Minified   bool
	Parsed     bool
	RetryCount int
}

type Service struct {
	m    *min.M
	pool chan *gomail.Client
	cfg  config.SMTPConfig
}

var (
	instance *Service
	once     sync.Once
)

func NewService(cfg config.SMTPConfig) *Service {
	once.Do(func() {
		m := min.New()
		m.Add("text/html", &minhtml.Minifier{
			KeepConditionalComments: true,
			KeepDefaultAttrVals:     true,
			KeepEndTags:             true,
			KeepQuotes:              true,
		})
		m.Add("text/css", &mincss.Minifier{})

		opts := []gomail.Option{
			gomail.WithPort(cfg.Port),
			gomail.WithTimeout(time.Duration(cfg.Timeout) * time.Second),
		}
		if !cfg.AllowTLS {
			opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
		}
		if cfg.Username != "" && cfg.Password != "" {
			opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthPlain))
			opts = append(opts, gomail.WithUsername(cfg.Username))
			opts = append(opts, gomail.WithPassword(cfg.Password))
		}

		pool := make(chan *gomail.Client, cfg.PoolSize)
		for i := 0; i < cfg.PoolSize; i++ {
			client, err := gomail.NewClient(cfg.Host, opts...)
			if err != nil {
				panic(fmt.Sprintf("create smtp client %d: %v", i, err))
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := client.DialWithContext(ctx); err != nil {
				cancel()
				panic(fmt.Sprintf("smtp connection %d test: %v", i, err))
			}
			cancel()

			pool <- client
		}

		instance = &Service{m: m, pool: pool, cfg: cfg}
	})
	return instance
}

func (s *Service) Close() {
	close(s.pool)
	for client := range s.pool {
		client.Close()
	}
}

func (s *Service) Minify(mail *Mail) {
	minified, err := s.m.String("text/html", mail.Body)
	if err != nil {
		return
	}
	mail.Body = minified
	mail.Minified = true
}

func (s *Service) Parse(mail *Mail) {
	html := mail.Body
	var buf strings.Builder
	buf.Grow(len(html))

	i := 0

	for {
		start := strings.Index(html[i:], "{{")
		if start == -1 {
			buf.WriteString(html[i:])
			break
		}
		buf.WriteString(html[i : i+start])
		i += start + 2

		end := strings.Index(html[i:], "}}")

		if end == -1 {
			buf.WriteString("{{")
			buf.WriteString(html[i:])

			break
		}

		key := html[i : i+end]

		if val, ok := mail.Vars[key]; ok {
			buf.WriteString(val)
		} else {
			buf.WriteString("{{")
			buf.WriteString(key)
			buf.WriteString("}}")
		}

		i += end + 2

	}

	mail.Body = buf.String()
	mail.Parsed = true
}

func (s *Service) Send(mail *Mail) error {
	msg := gomail.NewMsg()
	if err := msg.From(mail.From); err != nil {
		return fmt.Errorf("set from: %w", err)
	}
	if err := msg.To(mail.To); err != nil {
		return fmt.Errorf("set to: %w", err)
	}
	msg.Subject(mail.Subject)
	msg.SetBodyString(gomail.TypeTextHTML, mail.Body)

	client := <-s.pool
	err := client.Send(msg)

	if err != nil && !s.IsSendErrorTemp(err) {
		if redialErr := s.redial(client); redialErr == nil {
			err = client.Send(msg)
		}
	}

	s.pool <- client
	return err
}

func (s *Service) redial(client *gomail.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = client.Close() // no-op if already closed
	return client.DialWithContext(ctx)
}

func (s *Service) IsSendErrorTemp(err error) bool {
	if err == nil {
		return false
	}
	var sendErr *gomail.SendError
	if errors.As(err, &sendErr) {
		return sendErr.IsTemp()
	}
	return false
}
