package redisrepo

import (
	"context"
	"fmt"
	"math/rand"
	"server/external/message" // можно было, конечно полностью отделить логику от message - но там и так красивый интерфейс, так что не думаю, что это было бы хорошо, тк код выглядел бы тогда менее красиво
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisRepo struct {
	client *redis.Client
	ctx    context.Context
	lg     *zap.Logger
	mu     sync.Mutex
}

func NewRepo(dbAddr string, ctx context.Context, lg *zap.Logger) *RedisRepo {
	client := redis.NewClient(&redis.Options{
		Addr:     dbAddr,
		Password: "",
		DB:       0,
	})
	return &RedisRepo{client: client, ctx: ctx, lg: lg.With(zap.String("adapters", "redis cache")), mu: sync.Mutex{}}
}

type txContext struct {
	key         string
	val         int64
	valIsActual bool
	ctx         context.Context
}

func (tc *txContext) addMessageTransaction(tx *redis.Tx) error {
	val, err := tx.Get(tc.ctx, tc.key).Int64()
	if err != nil && err != redis.Nil {
		return err
	}
	if val >= tc.val {
		tc.valIsActual = false
		return nil
	}

	_, err = tx.TxPipelined(tc.ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(tc.ctx, tc.key, fmt.Sprint(tc.val), 0)
		return nil
	})
	return err
}

func (rr *RedisRepo) AddMessage(msg message.Message, lMsgTmSt int64) {
	go func() {
		rr.mu.Lock()
		defer rr.mu.Unlock()

		cId := strconv.Itoa(msg.GetChatId())
		txc := txContext{cId, lMsgTmSt, true, rr.ctx}

		timeTck := time.NewTicker(time.Duration(rand.Intn(100)) * time.Millisecond)
		defer timeTck.Stop()
		for txc.valIsActual {
			select {
			case <-(*timeTck).C:
				err := rr.client.Watch(rr.ctx, txc.addMessageTransaction, cId)
				if err == nil {
					return
				}
				if err != redis.TxFailedErr {
					rr.lg.Warn("Failed to add message, got unexpected err", zap.Error(err), zap.String("conf id", cId), zap.Int("user id", msg.GetUserId()))
				}
			case <-rr.ctx.Done():
				return
			}
		}
		rr.lg.Debug("Successfully add last message time stamp")
	}()
}

// return true if there are unread messages, false otherwise
func (rr *RedisRepo) CheckLastMsgTimeStamp(cId string, msgTimeStamp int64) (bool, error) {
	lSaved, err := rr.client.Get(rr.ctx, cId).Int64()
	if err != nil && err != redis.Nil {
		rr.lg.Warn("Failed to check last message time stamp", zap.Error(err), zap.String("conf id", cId))
		return false, err
	}
	return lSaved > msgTimeStamp, nil
}

func (rr *RedisRepo) CloseRepo() error {
	if err := rr.client.Close(); err != nil {
		rr.lg.Error("Failed to close cache repo", zap.Error(err))
		return err
	}
	return nil
}
