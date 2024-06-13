package main

import (
	"context"
	"envconfig"
	"log"
	"os"
	"os/signal"
	"server/external/adapters/postgresrepo"
	"storage/internal/cache_adapters/redisrepo"
	"storage/internal/consumer"
	"storage/internal/ports/httpnetserver"
	"syscall"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func runServer(msgHandler *consumer.MessageHandler, serverAddr string) {
	httpnetserver.RunServer(serverAddr, msgHandler)
}

func connectToDbs(ctx context.Context, DbAddr string, cDbAddr string, lg *zap.Logger) *consumer.MessageHandler {
	eg, newCtx := errgroup.WithContext(ctx)
	db := postgresrepo.NewRepo(DbAddr, newCtx, lg.With(zap.String("db", "postgres")))
	cdb := redisrepo.NewRepo(cDbAddr, newCtx, lg.With(zap.String("cdb", "redis")))
	return consumer.NewMessageHandler(newCtx, db, cdb, eg, lg)
}

var group = "2"

func main() {
	es := envconfig.NewEnvStorage()
	brokers := es.EnvGetAddr("kafkaAddr")
	topics := es.EnvGetAddr("messageTopics")

	lg, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to init logger")
	}
	lg.Info("Starting storage")
	lg.Warn("check", zap.String("brokers", brokers), zap.String("topics", topics))
	sarama.Logger = zap.NewStdLog(lg.With(zap.String("storage", "sarama")))

	ctx, cncl := context.WithCancel(context.Background())
	msgHandler := connectToDbs(ctx, es.EnvGetAddr("postgresAddr"), es.EnvGetAddr("redisAddr"), lg)
	consumer, err := consumer.RunConsumer(ctx, msgHandler, brokers, group, topics)
	if err != nil {
		log.Fatal(err)
	}

	lg.Info("Sarama is running")
	runServer(msgHandler, es.EnvGetAddr("storageServerAddr"))

	sigterm := make(chan os.Signal, 2)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	keepRunning := true
	for keepRunning {
		select {
		case <-ctx.Done():
			msgHandler.Lg.Info("Ctx cancelled")
			keepRunning = false
		case <-sigterm:
			msgHandler.Lg.Info("Get sigterm")
			keepRunning = false
			cncl()
		}
	}

	if err = consumer.Close(); err != nil {
		msgHandler.Lg.Error("Failed to shut down consumer", zap.Error(err))
		return
	}

	if err := msgHandler.Eg.Wait(); err != nil {
		msgHandler.Lg.Error("Storage shut down with error", zap.Error(err))
		return
	}

	if err = msgHandler.Cdb.CloseRepo(); err != nil {
		msgHandler.Lg.Error("Failed to shut down cache db", zap.Error(err))
		return
	}

	if err = msgHandler.Db.CloseRepo(); err != nil {
		msgHandler.Lg.Error("Failed to shut down db", zap.Error(err))
		return
	}

	msgHandler.Lg.Info("Storage shut down successfully")
}
