package kafka

import (
	"fmt"
	"log"
	"strings"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"
)

// NewReader initializes a Kafka consumer group reader with optional TLS/SASL auth.
// When groupID is provided, consumer offset commits are tracked by the Kafka cluster.
func NewReader(brokers []string, groupID string, topic string, auth AuthConfig, startOffset ...int64) *kafkaGo.Reader {
	if len(brokers) == 0 {
		return nil
	}

	mechanism, tlsConfig, _ := BuildTLSAndSASL(auth)

	var dialer *kafkaGo.Dialer
	if mechanism != nil || tlsConfig != nil {
		dialer = &kafkaGo.Dialer{
			Timeout:       10 * time.Second,
			DualStack:     false,
			TLS:           tlsConfig,
			SASLMechanism: mechanism,
		}
	}

	offset := kafkaGo.LastOffset
	if len(startOffset) > 0 {
		offset = startOffset[0]
	}

	rCfg := kafkaGo.ReaderConfig{
		Brokers:          brokers,
		Topic:            topic,
		Dialer:           dialer,
		MinBytes:         1,    // 1 byte: deliver immediately on arrival
		MaxBytes:         10e6, // 10MB
		CommitInterval:   time.Second,
		MaxWait:          1 * time.Second,
		ReadBatchTimeout: 10 * time.Second,
		StartOffset:      offset,
		ErrorLogger: kafkaGo.LoggerFunc(func(msg string, args ...any) {
			formatted := fmt.Sprintf(msg, args...)
			// Filter out expected idle long-polling socket timeouts and normal EOFs
			if !strings.Contains(formatted, "i/o timeout") && !strings.Contains(formatted, "EOF") {
				log.Printf("[Kafka Reader Error] %s\n", formatted)
			}
		}),
	}
	if groupID != "" {
		rCfg.GroupID = groupID
	} else {
		rCfg.Partition = 0
	}

	return kafkaGo.NewReader(rCfg)
}
