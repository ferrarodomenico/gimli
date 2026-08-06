package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ferrarodomenico/gimli/internal/config"
	"github.com/ferrarodomenico/gimli/internal/mail"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

var sampleBody = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<!--[if mso]><noscript><xml><o:OfficeDocumentSettings><o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript><![endif]-->
<style>
  body { margin: 0; padding: 0; background-color: #f4f6f9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; }
  .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; }
  .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 40px 30px; text-align: center; }
  .header h1 { color: #ffffff; font-size: 28px; margin: 0; font-weight: 700; letter-spacing: -0.5px; }
  .header p { color: rgba(255,255,255,0.85); font-size: 16px; margin: 8px 0 0 0; }
  .body { padding: 32px 30px; }
  .body p { color: #333333; font-size: 16px; line-height: 1.6; margin: 0 0 16px 0; }
  .greeting { font-size: 18px; font-weight: 600; color: #1a1a2e; margin-bottom: 12px; }
  .order-table { width: 100%; border-collapse: collapse; margin: 24px 0; }
  .order-table th { background-color: #f8f9fc; color: #555; font-size: 12px; text-transform: uppercase; letter-spacing: 0.8px; padding: 12px 16px; text-align: left; border-bottom: 2px solid #e2e6f0; }
  .order-table td { padding: 14px 16px; color: #333; font-size: 15px; border-bottom: 1px solid #eef0f5; }
  .order-table .total { font-weight: 700; font-size: 18px; color: #667eea; }
  .cta-button { display: inline-block; background-color: #667eea; color: #ffffff !important; text-decoration: none; font-size: 16px; font-weight: 600; padding: 14px 36px; border-radius: 6px; margin: 16px 0 24px 0; }
  .info-box { background-color: #f0f4ff; border-left: 4px solid #667eea; padding: 16px 20px; margin: 24px 0; border-radius: 0 6px 6px 0; }
  .info-box p { font-size: 14px; color: #555; margin: 4px 0; }
  .divider { border: 0; border-top: 1px solid #eef0f5; margin: 24px 0; }
  .footer { padding: 20px 30px; background-color: #f8f9fc; text-align: center; }
  .footer p { color: #999; font-size: 12px; line-height: 1.6; margin: 0 0 8px 0; }
  .footer a { color: #667eea; text-decoration: none; }
  @media only screen and (max-width: 600px) {
    .header { padding: 28px 20px; }
    .header h1 { font-size: 22px; }
    .body { padding: 24px 20px; }
    .order-table th, .order-table td { padding: 10px 12px; font-size: 13px; }
  }
</style>
</head>
<body>
<!--[if mso]><table role="presentation" width="600" align="center"><tr><td><![endif]-->
<div class="container">
  <div class="header">
    <h1>Order Confirmed</h1>
    <p>{{companyName}} - Your order is on its way</p>
  </div>
  <div class="body">
    <p class="greeting">Hi {{name}},</p>
    <p>Great news — your order has been confirmed and is now being processed. Here is a summary of your purchase:</p>

    <table class="order-table">
      <thead>
        <tr>
          <th>Item</th>
          <th>Qty</th>
          <th>Price</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>{{itemName}}</td>
          <td>{{itemQty}}</td>
          <td>{{orderTotal}}</td>
        </tr>
      </tbody>
    </table>

    <div class="info-box">
      <p><strong>Order Number:</strong> {{orderNumber}}</p>
      <p><strong>Order Date:</strong> {{orderDate}}</p>
    </div>

    <a href="{{trackingLink}}" class="cta-button">Track Your Order</a>

    <hr class="divider">

    <p>You can manage your orders and preferences from your <a href="{{dashboardLink}}">account dashboard</a> at any time.</p>
    <p>If you have any questions, reach out to us at <a href="mailto:{{supportEmail}}">{{supportEmail}}</a> — we are happy to help.</p>

    <p>Thanks for choosing {{companyName}},<br>The {{companyName}} Team</p>
  </div>
  <div class="footer">
    <p>&copy; {{year}} {{companyName}}. All rights reserved.</p>
    <p><a href="{{unsubscribeLink}}">Unsubscribe</a> from these emails.</p>
  </div>
</div>
<!--[if mso]></td></tr></table><![endif]-->
</body>
</html>`

func main() {
	count := flag.Int("count", 10000, "number of messages to publish")
	workers := flag.Int("workers", 1000, "concurrent publisher goroutines")
	flag.Parse()

	_ = godotenv.Load()

	cfg := config.LoadConfig()

	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to rabbitmq: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open channel: %v\n", err)
		os.Exit(1)
	}
	defer ch.Close()

	template := mail.Mail{
		From:    "loadtest@gimli.local",
		To:      "recipient@example.com",
		Subject: "Performance Test",
		Body:    sampleBody,
		Vars: map[string]string{
			"name":            "Tester",
			"orderNumber":     "ORD-2024-0042",
			"orderDate":       "Aug 6, 2026",
			"orderTotal":      "$149.97",
			"itemName":        "Gimli Pro License",
			"itemQty":         "3",
			"supportEmail":    "support@gimli.local",
			"companyName":     "Gimli",
			"year":            "2026",
			"dashboardLink":   "https://gimli.local/dashboard",
			"trackingLink":    "https://gimli.local/orders/ORD-2024-0042",
			"unsubscribeLink": "https://gimli.local/unsubscribe",
		},
	}

	body, err := json.Marshal(template)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal mail: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Publishing %d messages with %d workers to exchange=%q key=%q...\n",
		*count, *workers, cfg.MainQueue.Exchange, cfg.MainQueue.RoutingKey)

	var sent int64
	var wg sync.WaitGroup

	start := time.Now()
	fmt.Printf("Start time: %s\n", start.Format(time.RFC3339))

	msgCh := make(chan struct{}, *workers*4)
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range msgCh {
				if err := ch.Publish(
					cfg.MainQueue.Exchange,
					cfg.MainQueue.RoutingKey,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        body,
					},
				); err != nil {
					fmt.Fprintf(os.Stderr, "publish error: %v\n", err)
					return
				}
				atomic.AddInt64(&sent, 1)
			}
		}()
	}

	for i := 0; i < *count; i++ {
		msgCh <- struct{}{}
	}
	close(msgCh)

	wg.Wait()
	elapsed := time.Since(start)

	total := atomic.LoadInt64(&sent)
	fmt.Printf("Done. Sent: %d | Duration: %s | Throughput: %.0f msg/s\n",
		total, elapsed.Round(time.Millisecond), float64(total)/elapsed.Seconds())
}
