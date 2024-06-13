package client

import (
	"client/internal/chat/chatwebsocket"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"server/external/message"
	"syscall"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var ErrorFailedToReadKeyboard = errors.New("got error during reading client keyboard")
var ErrorSigQuit = errors.New("client got signal to quit")

func RunClient(sAddr string, lg *zap.Logger) error { // TODO too enormous func
	eg, ctx := errgroup.WithContext(context.Background())

	sigQuit := make(chan os.Signal, 2)
	signal.Notify(sigQuit, syscall.SIGINT, syscall.SIGTERM)

	chat, e := chatwebsocket.NewChat(ctx, eg, sAddr, lg)
	if e != nil {
		return e
	}

	eg.Go(func() error {
		select {
		case s := <-sigQuit:
			lg.Warn("Shutting client down sigQuit", zap.Any("signal", s))
			return ErrorSigQuit
		case <-ctx.Done():
			return nil
		}
	})

	var uName string = "Undefined"
	msgsToSend := waitForMessages(ctx, eg, &uName, lg)

	msgsToRecieve := chat.RecieveMessages()
	eg.Go(func() error {
		for {
			select {
			case m := <-msgsToSend:
				if e = chat.SendMessage(message.Message{User: uName, Text: m}); e != nil {
					return e
				}
			case m, ok := <-msgsToRecieve:
				if ok {
					fmt.Println(m.BeautifulPrint())
				}
			case <-ctx.Done():
				color.Red("Type enter to close client")
				chat.CloseConnection()
				if e = os.Stdin.Close(); e != nil {
					lg.Error("Failed to close stdin to shut down", zap.Error(e))
				}
				tckr := time.NewTicker(100 * time.Millisecond)
				defer tckr.Stop()
				for ; ; <-tckr.C { // clear channels not to have any chances to thread lock
					_, ok1 := <-msgsToSend
					_, ok2 := <-msgsToRecieve
					if !ok1 && !ok2 {
						break
					}
				}
				return nil
			}
		}
	})
	if e = eg.Wait(); e != nil {
		if e == ErrorSigQuit {
			lg.Info("Client shut down", zap.Error(e))
		} else {
			lg.Error("Client shut down", zap.Error(e))
		}
		return e
	}
	return nil
}

func waitForMessages(ctx context.Context, eg *errgroup.Group, uName *string, lg *zap.Logger) chan string {
	color.Cyan("Enter your name")
	msgs := make(chan string, 1)
	var msg string
	var first bool = true
	eg.Go(func() error {
		defer close(msgs)
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			if _, e := fmt.Scanln(&msg); e != nil {
				if _, ok := <-ctx.Done(); !ok {
					return nil
				}
				lg.Error("Failed to scan message from keyboard", zap.Error(e))
				return ErrorFailedToReadKeyboard
			}

			lg.Info("Got new message to send to server")
			if first {
				*uName = msg
				first = false
			} else {
				msgs <- msg
			}
		}
	})
	return msgs
}
