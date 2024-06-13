package chatwebsocket

import (
	"context"
	"errors"
	"server/external/message"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type ChatWebSocket struct {
	sAddr string
	conn  *websocket.Conn
	ctx   context.Context
	eg    *errgroup.Group
	lg    *zap.Logger
}

var ErrorFailedToEstConnection error = errors.New("failed to established connection with server")
var ErrorFailedToParseResp error = errors.New("failed to parse response from server")
var ErrorEmptyFirstServerResp error = errors.New("expected to get user id in first server response")
var ErrorFailedToWriteMsg error = errors.New("error occured during writing message to websocket conn")
var ErrorFailedToParseMsg error = errors.New("error occured during parsing message")
var ErrorFailedToGetMsg error = errors.New("error occured during getting message from server")
var ErrorUnexpectedMsgType error = errors.New("got unexpected message type from server (not equal to websocket.TextMessage)")
var ErrorServerClosedConnection error = errors.New("server closed connection to client")

func NewChat(ctx context.Context, eg *errgroup.Group, sA string, lg *zap.Logger) (ChatWebSocket, error) {
	conn, _, e := websocket.DefaultDialer.Dial(sA, nil) // dialcontext? hmm
	if e != nil {
		lg.Error("Failed to init websocket connection", zap.Error(e), zap.String("server addr", sA))
		return ChatWebSocket{}, ErrorFailedToEstConnection
	}

	return ChatWebSocket{sAddr: sA, conn: conn, ctx: ctx, eg: eg, lg: lg}, nil
}

func (ch *ChatWebSocket) SendMessage(msg message.Message) error {
	ch.lg.Info("Send message to server", zap.Int("message len", len(msg.Text)), zap.String("message author", msg.User))
	buf, e := message.EncodeMsgsToBytes(msg)
	if e != nil {
		ch.lg.Warn("Failed to encode message to bytes", zap.Error(e), zap.Int("message len", len(msg.Text)), zap.String("message author", msg.User))
		return ErrorFailedToParseMsg
	}

	if e = ch.conn.WriteMessage(websocket.TextMessage, buf); e != nil {
		ch.lg.Warn("Failed write message to webscocket conn", zap.Error(e), zap.String("message author", msg.User))
		return ErrorFailedToWriteMsg
	}
	return nil
}

func (ch *ChatWebSocket) RecieveMessages() chan message.Message {
	msgs := make(chan message.Message)
	ch.eg.Go(func() error {
		defer close(msgs)
		for {
			select {
			case <-ch.ctx.Done():
				return nil
			default:
			}

			msgT, buf, e := ch.conn.ReadMessage()
			ch.lg.Info("Received new message from server", zap.Int("message type", msgT))
			if msgT == websocket.CloseMessage || msgT == -1 { // same problem with msgT as on server TODO
				ch.lg.Info("Server closed connection")
				return ErrorServerClosedConnection
			}
			if e != nil {
				ch.lg.Error("Failed to read message from conn", zap.Error(e))
				return ErrorFailedToGetMsg
			}
			if msgT != websocket.TextMessage {
				ch.lg.Warn("Got unexpected message type (not textmessage)", zap.Int("message type", msgT))
				continue
			}
			msg, e := message.DecodeMsgFromBytes(buf)
			if e != nil {
				ch.lg.Error("Failed to decode received message", zap.Error(e))
				return ErrorFailedToParseMsg
			}
			msgs <- msg
		}
	})
	return msgs
}

func (ch *ChatWebSocket) CloseConnection() {
	if e := ch.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Client is shut down")); e != nil {
		ch.lg.Warn("Failed write close message to webscocket conn", zap.Error(e))
		// I don't think parent code need that error
	}
	if e := ch.conn.Close(); e != nil {
		ch.lg.Warn("Failed close websocket conn", zap.Error(e))
		// I don't think parent code need that error
	}
}
