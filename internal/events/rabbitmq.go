package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "saas.events"
	QueueName    = "projects.events"
)

type RabbitMQPublisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel

	confirms chan amqp.Confirmation
	returns  chan amqp.Return

	mu     sync.Mutex
	closed bool
}

type ProjectCreatedEvent struct {
	Event     string    `json:"event"`
	TenantID  string    `json:"tenant_id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	cleanup := func() {
		_ = ch.Close()
		_ = conn.Close()
	}

	err = ch.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("declare RabbitMQ exchange: %w", err)
	}

	queue, err := ch.QueueDeclare(
		QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("declare projects queue: %w", err)
	}

	err = ch.QueueBind(
		queue.Name,
		"project.created",
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("bind projects queue: %w", err)
	}

	// Enable publisher confirmation mode.
	if err := ch.Confirm(false); err != nil {
		cleanup()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	publisher := &RabbitMQPublisher{
		connection: conn,
		channel:    ch,
		confirms:   ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
		returns:    ch.NotifyReturn(make(chan amqp.Return, 1)),
	}

	return publisher, nil
}

func (p *RabbitMQPublisher) PublishProjectCreated(
	ctx context.Context,
	event ProjectCreatedEvent,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("RabbitMQ publisher is closed")
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("serialize project event: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		ExchangeName,
		"project.created",
		true,  // mandatory: detect unroutable messages
		false, // immediate: unused
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish project.created event: %w", err)
	}

	// Wait for RabbitMQ's publisher confirmation.
	select {
	case confirmation, ok := <-p.confirms:
		if !ok {
			p.closeLocked()
			return fmt.Errorf("RabbitMQ confirmation channel closed")
		}

		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ negatively acknowledged the message")
		}

	case <-ctx.Done():
		// The outcome is uncertain. Close this channel so a late
		// confirmation cannot be mistaken for a future publication.
		p.closeLocked()
		return fmt.Errorf("waiting for RabbitMQ confirmation: %w", ctx.Err())
	}

	// An unroutable mandatory message is returned before its ACK.
	// Check for that return after receiving the confirmation.
	select {
	case returned, ok := <-p.returns:
		if ok {
			return fmt.Errorf(
				"RabbitMQ returned unroutable message: %s",
				returned.ReplyText,
			)
		}
	default:
	}

	return nil
}

func (p *RabbitMQPublisher) closeLocked() {
	if p.closed {
		return
	}

	p.closed = true

	if p.channel != nil {
		_ = p.channel.Close()
	}

	if p.connection != nil {
		_ = p.connection.Close()
	}
}

func (p *RabbitMQPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.closeLocked()
}
