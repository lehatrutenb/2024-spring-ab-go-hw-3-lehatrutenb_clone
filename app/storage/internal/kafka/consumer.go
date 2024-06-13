package kafka

import (
	"log"

	"github.com/IBM/sarama"
)

type MessageHandlerFunc func(message *sarama.ConsumerMessage) error

type Consumer struct {
	handler MessageHandlerFunc
}

func NewConsumer(handler MessageHandlerFunc) *Consumer {
	return &Consumer{
		handler: handler,
	}
}

func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				log.Println("message channel closed")
				return nil
			}
			err := consumer.handler(message)
			if err != nil {
				log.Fatal(err)
			}
			session.MarkMessage(message, "")
		case <-session.Context().Done():
			session.Commit()
			return nil
		}
	}
}
