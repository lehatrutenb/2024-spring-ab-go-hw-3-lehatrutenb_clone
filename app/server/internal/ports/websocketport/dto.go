package websocketport

import (
	"errors"
	"server/external/message"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var ErrorFailedToWriteMsg error = errors.New("failed to write message")
var ErrorClosedConnection error = errors.New("websocket connection was closed")
var ErrorRepoFailedToReadMsg error = errors.New("repo failed to read message")
var ErrorServerFailedToReadMsg error = errors.New("server failed to read message from conn")
var ErrorFailedToParseMsg error = errors.New("server failed to parse message from conn")
var ErrorFailedToWriteMsgToRepo error = errors.New("server failed to write message to repo")
var ErrorFailedToEncodeMsg error = errors.New("server failed to encode message from repo to buffer")

func (s server) writeLastMessagesToNewbie(conn *websocket.Conn, amt int) {
	s.eg.Go(func() error {
		msgs, e := s.repo.GetLastKMessages(amt)
		if e != nil {
			s.lg.Error("Failed to get last k messages for newbie", zap.Error(e), zap.Int("user id", s.clients[conn]))
			return ErrorRepoFailedToReadMsg
		}

		bufs, _, e := s.prepareMsgsToSend(msgs)
		if e != nil {
			s.lg.Error("Failed to prepare messages for newbie", zap.Error(e), zap.Int("user id", s.clients[conn]))
			return ErrorFailedToEncodeMsg
		}
		for _, buf := range bufs {
			select {
			case <-s.ctx.Done():
				return nil
			default:
			}

			if e := conn.WriteMessage(websocket.TextMessage, buf); e != nil {
				s.lg.Error("Failed to write message to websocket connection", zap.Error(e), zap.Int("user id", s.clients[conn]))
				return ErrorFailedToWriteMsg
			}
		}
		return nil
	})
}

func (s server) prepareMsgsToSend(msgs []message.Message) ([][]byte, []int, error) {
	toReturnBufs := make([][]byte, len(msgs))
	toReturnIds := make([]int, len(msgs))

	for i, msg := range msgs {
		var e error
		toReturnBufs[i], e = message.EncodeMsgsToBytes(msg)
		if e != nil {
			s.lg.Error("Failed to encode message from repo", zap.Error(e))
			return [][]byte{}, []int{}, ErrorFailedToEncodeMsg
		}
		toReturnIds[i] = msg.GetUserId()
	}

	return toReturnBufs, toReturnIds, nil
}

func (s server) waitForMessages() {
	s.eg.Go(func() error {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return nil
			case <-ticker.C:
				msgs, e := s.repo.GetNewerMessages()
				if len(msgs) == 0 {
					s.lg.Debug("No new messages received")
					continue
				}
				if e != nil {
					s.lg.Error("Failed to get message from repo", zap.Error(e))
					return ErrorRepoFailedToReadMsg
				}

				bufs, uIds, e := s.prepareMsgsToSend(msgs)
				if e != nil {
					return e
				}

				for i, buf := range bufs {
					if e = s.writeMessage(buf, uIds[i]); e != nil {
						return e
					}
				}
			}
		}
	})
}

func (s server) writeMessage(buf []byte, maId int) (errReturn error) {
	s.lg.Info("Send message to clients", zap.Int("author user id", maId), zap.Int("message buf len", len(buf)))
	for conn, uId := range s.clients {
		if uId == maId {
			continue
		}

		if e := conn.WriteMessage(websocket.TextMessage, buf); e != nil {
			s.lg.Warn("Failed to write message to websocket connection", zap.Error(e), zap.Int("user id", uId))
			errReturn = ErrorFailedToWriteMsg
		}
	}
	return
}

func (s server) closeConns() { // I know that I will close conns twice, but it is for more secure
	s.lg.Info("Close connections with clients")
	for conn, uId := range s.clients {
		if e := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Server is shut down")); e != nil {
			s.lg.Warn("Failed to write close message to websocket connection", zap.Error(e), zap.Int("user id", uId))
		}
		if e := conn.Close(); e != nil {
			s.lg.Warn("Failed to close websocket connection", zap.Error(e), zap.Int("user id", uId))
		}
	}
}

func (s *server) recieveMessages(conn *websocket.Conn) error {
	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}
		mt, buf, e := conn.ReadMessage()
		s.lg.Info("Got message from client", zap.Int("user id", s.clients[conn]), zap.Int("message buf len", len(buf)), zap.Int("message type", mt), zap.Error(e))

		if e != nil && websocket.IsUnexpectedCloseError(e, websocket.CloseNormalClosure) { // CloseGoingAway?
			s.lg.Warn("Failed to read message from conn", zap.Error(e))
			return ErrorServerFailedToReadMsg
		}
		if mt == websocket.CloseMessage || mt == -1 { // now I can't really explain why server got -1 not 8 - TODO check it
			s.lg.Info("Client closed connection with a server as he wished", zap.Int("user id", s.clients[conn]))
			return ErrorClosedConnection
		}
		if mt != websocket.TextMessage {
			s.lg.Warn("Server got unexpected message type", zap.Int("message type", mt), zap.Int("user id", s.clients[conn]))
			continue
		}

		msg, e := message.DecodeMsgFromBytes(buf)
		if e != nil {
			s.lg.Warn("Unable to decode received message", zap.Error(e), zap.Int("user id", s.clients[conn]))
			return ErrorFailedToParseMsg
		}
		s.lg.Warn("!!!!!!", zap.Any("USER ID ", s.clients[conn]))
		msg.SetUserId(s.clients[conn])
		msg.SetChatId(0) // now only one chat for developing
		if e = s.repo.AddMessage(msg); e != nil {
			s.lg.Warn("Failed to send message to client", zap.Error(e), zap.Int("user id", s.clients[conn]))
			return ErrorFailedToWriteMsgToRepo
		}
	}
}
