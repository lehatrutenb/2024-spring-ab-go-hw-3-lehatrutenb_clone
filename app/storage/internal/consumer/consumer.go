package consumer

import (
	"context"
	"strings"

	"server/external/adapters"
	"server/external/message"
	"storage/internal/cache_adapters"
	"storage/internal/kafka"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func initConsumerConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.DefaultVersion
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	return config
}

type MessageHandler struct {
	Ctx context.Context
	Db  adapters.Repository
	Cdb cache_adapters.CacheRepository
	Eg  *errgroup.Group
	Lg  *zap.Logger
}

func NewMessageHandler(ctx context.Context, db adapters.Repository, cdb cache_adapters.CacheRepository, eg *errgroup.Group, lg *zap.Logger) *MessageHandler {
	return &MessageHandler{ctx, db, cdb, eg, lg}
}

func (mh *MessageHandler) handleMessage(mb *sarama.ConsumerMessage) error {
	msg, err := message.DecodeMsgFromBytes(mb.Value)
	if err != nil {
		mh.Lg.Warn("Failed to decode message", zap.Time("msg time stamp", mb.Timestamp))
		return err
	}
	mh.Cdb.AddMessage(msg, mb.Timestamp.UnixMilli())
	if err := mh.Db.AddMessage(msg); err != nil {
		mh.Lg.Error("Failed to add msg to db")
		return err
	}
	mh.Lg.Debug("Successfully added msg to db", zap.Int("user id", msg.GetUserId()), zap.Int("chat id", msg.GetChatId()))

	return nil
}

func RunConsumer(ctx context.Context, msgHandler *MessageHandler, brokers string, group string, topics string) (sarama.ConsumerGroup, error) {
	cons := kafka.NewConsumer(msgHandler.handleMessage)
	consGroup, err := sarama.NewConsumerGroup(strings.Split(brokers, ","), group, initConsumerConfig())
	if err != nil {
		msgHandler.Lg.Error("Failed to init consumer group", zap.String("brokers", brokers), zap.String("group", group))
		return nil, err
	}

	msgHandler.Eg.Go(
		func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				default:
					if err = consGroup.Consume(ctx, strings.Split(topics, ","), cons); err != nil {
						msgHandler.Lg.Error("Consumer cluster connection lost", zap.String("topics", topics))
						return err
					}
				}
			}
		},
	)
	return consGroup, nil
}
