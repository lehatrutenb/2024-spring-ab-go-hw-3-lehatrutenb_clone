package httpnetserver

import (
	"context"
	"errors"
	"net/http"
	strorage_response "storage/external/api_response"
	"storage/internal/consumer"
	"strconv"

	"go.uber.org/zap"
)

var ErrorFailedToParseMsgAmt error = errors.New("Failed to parse message newbie amount")

type server struct {
	ctx *context.Context
	mh  *consumer.MessageHandler
	lg  *zap.Logger
}

func newServer(mh *consumer.MessageHandler) *server {
	return &server{&mh.Ctx, mh, mh.Lg.With(zap.String("port", "httpserver"))}
}

func (s *server) getNewMessagesHandler(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	cId := qs.Get("conference_id")
	lMsgTimeStamp, err := strconv.ParseInt(qs.Get("last_message_time_stamp"), 10, 64)
	if err != nil {
		s.lg.Warn("Failed to parse message time stamp", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	s.lg.Debug("Server asked for new messages", zap.String("conference_id", cId))

	ok, err := s.mh.Cdb.CheckLastMsgTimeStamp(cId, lMsgTimeStamp)
	if err != nil {
		s.lg.Error("Failed to check new messages in cache db", zap.Error(err))
	}

	if !ok {
		s.lg.Debug("No new messages found", zap.String("conference_id", cId))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
		return
	}

	s.mh.Db.SetLastMessageTimeStamp(lMsgTimeStamp)
	msgs, err := s.mh.Db.GetNewerMessages()
	if err != nil {
		s.lg.Warn("Failed to get new messages", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError) // change
		return
	}

	buf, err := strorage_response.EncodeResponseToBytes(strorage_response.NewResponse(msgs, s.mh.Db.GetLastMessageTimeStamp()))
	if err != nil {
		s.lg.Warn("Failed to conv messages to bytes", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	s.lg.Debug("Successfully send messages to server", zap.Int("msgs amount", len(msgs)))
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
}

func (s *server) getNewbieMessagesHandler(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	amt, err := strconv.Atoi(qs.Get("amount"))
	if amt < 0 || amt > 100 || err != nil { // bad const 100 change !!!
		s.lg.Warn("Failed to parse message amount", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	s.lg.Debug("Server asked for newbie messages", zap.Int("amount msgs", amt))
	msgs, err := s.mh.Db.GetLastKMessages(amt)
	if err != nil {
		s.lg.Warn("Failed to get new messages", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError) // change
		return
	}

	buf, err := strorage_response.EncodeResponseToBytes(strorage_response.NewResponse(msgs, s.mh.Db.GetLastMessageTimeStamp()))
	if err != nil {
		s.lg.Warn("Failed to conv response to bytes", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.lg.Debug("Successfully send newbie messages to server", zap.Int("msgs amount", len(msgs)))
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
}
