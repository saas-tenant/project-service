package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "saas.events"
)

type RabbitMQPublisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
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

	err = ch.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare RabbitMQ exchange: %w", err)
	}

	// Declare a durable queue.
	queue, err := ch.QueueDeclare(
		"projects.events",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare projects queue: %w", err)
	}

	// Bind the queue to receive project.created events.
	err = ch.QueueBind(
		queue.Name,
		"project.created",
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("bind projects queue: %w", err)
	}

	return &RabbitMQPublisher{
		connection: conn,
		channel:    ch,
	}, nil
}

func (p *RabbitMQPublisher) PublishProjectCreated(
	ctx context.Context,
	event ProjectCreatedEvent,
) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("serialize project event: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		ExchangeName,
		"project.created",
		false, // mandatory
		false, // immediate (unused by RabbitMQ)
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

	return nil
}

func (p *RabbitMQPublisher) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.connection != nil {
		_ = p.connection.Close()
	}
}
