package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer  *kafkaGo.Writer
	enabled bool
}

// NewProducer initializes a Kafka writer with connection pooling, asynchronous batching, and optional TLS/SASL auth.
func NewProducer(brokers []string, auth ...AuthConfig) *Producer {
	if len(brokers) == 0 {
		log.Println("[Kafka] No brokers configured. Kafka producer operating in standby mode.")
		return &Producer{enabled: false}
	}

	var authCfg AuthConfig
	if len(auth) > 0 {
		authCfg = auth[0]
	}

	mechanism, tlsConfig, err := BuildTLSAndSASL(authCfg)
	if err != nil {
		log.Printf("[Kafka] Warning: Failed to configure TLS/SASL authentication: %v\n", err)
	}

	var transport *kafkaGo.Transport
	if mechanism != nil || tlsConfig != nil {
		transport = &kafkaGo.Transport{
			TLS:  tlsConfig,
			SASL: mechanism,
		}
	}

	writer := &kafkaGo.Writer{
		Addr:                   kafkaGo.TCP(brokers...),
		Transport:              transport,
		Balancer:               &kafkaGo.LeastBytes{},
		BatchTimeout:           10 * time.Millisecond,
		BatchSize:              100,
		MaxAttempts:            3,
		WriteTimeout:           5 * time.Second,
		RequiredAcks:           kafkaGo.RequireOne,
		AllowAutoTopicCreation: true,
	}

	if mechanism != nil {
		log.Printf("[Kafka] Producer initialized with SASL_SSL authentication connected to brokers: %v\n", brokers)
	} else {
		log.Printf("[Kafka] Producer initialized connected to brokers: %v\n", brokers)
	}

	return &Producer{
		writer:  writer,
		enabled: true,
	}
}

// Publish serializes payload to JSON and publishes it to the specified topic.
func (p *Producer) Publish(ctx context.Context, topic string, key string, payload any) error {
	if p == nil || !p.enabled {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal kafka payload: %w", err)
	}

	msg := kafkaGo.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	err = p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("[Kafka] Failed to publish message to topic %s: %v\n", topic, err)
		return err
	}

	log.Printf("[Kafka] Published event to topic %s (key: %s)\n", topic, key)
	return nil
}

// Close gracefully flushes pending batch messages and closes the writer.
func (p *Producer) Close() error {
	if p == nil || !p.enabled || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// EnsureTopics verifies or programmatically creates required Kafka topics on the broker.
func EnsureTopics(ctx context.Context, brokers []string, auth AuthConfig, topics ...string) {
	if len(brokers) == 0 || len(topics) == 0 {
		return
	}

	mechanism, tlsConfig, err := BuildTLSAndSASL(auth)
	if err != nil {
		log.Printf("[Kafka] Topic admin auth warning: %v\n", err)
	}

	var transport *kafkaGo.Transport
	if mechanism != nil || tlsConfig != nil {
		transport = &kafkaGo.Transport{
			TLS:  tlsConfig,
			SASL: mechanism,
		}
	}

	client := &kafkaGo.Client{
		Addr:      kafkaGo.TCP(brokers...),
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	var topicConfigs []kafkaGo.TopicConfig
	for _, t := range topics {
		topicConfigs = append(topicConfigs, kafkaGo.TopicConfig{
			Topic:             t,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	req := &kafkaGo.CreateTopicsRequest{
		Addr:   kafkaGo.TCP(brokers[0]),
		Topics: topicConfigs,
	}

	resp, err := client.CreateTopics(ctx, req)
	if err != nil {
		log.Printf("[Kafka] Topic auto-creation request returned: %v (topics can also be created in Aiven console)\n", err)
		return
	}

	for t, tErr := range resp.Errors {
		if tErr != nil && tErr != kafkaGo.TopicAlreadyExists {
			log.Printf("[Kafka] Topic status '%s': %v\n", t, tErr)
		} else {
			log.Printf("[Kafka] Verified/created topic: %s\n", t)
		}
	}
}

