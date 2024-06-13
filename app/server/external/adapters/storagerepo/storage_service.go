package storagerepo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"server/external/message"
	"strconv"

	storage_response "storage/external/api_response"
	"storage/external/producer"

	"go.uber.org/zap"
)

type StorageRepo struct {
	sAddr    string
	producer *producer.Producer
	lg       *zap.Logger
	lMsgTmSt int64
}

var ErrorFailedMsgRequest error = errors.New("got non ok status code from server")

func NewRepo(ctx context.Context, rAddr map[string]string, lg *zap.Logger) *StorageRepo {
	producer, err := producer.NewProducer(ctx, rAddr["kafkaAddr"], lg, "chat.messages.add")
	if err != nil {
		lg.Fatal("Failed to connect to storage")
	}
	return &StorageRepo{sAddr: rAddr["storageAddr"], producer: producer, lg: lg, lMsgTmSt: -1}
}

func (sr *StorageRepo) GetNewerMessages() ([]message.Message, error) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/get", sr.sAddr), http.NoBody)
	if err != nil {
		return nil, err
	}

	queries := req.URL.Query()
	queries.Set("last_message_time_stamp", fmt.Sprint(sr.GetLastMessageTimeStamp()))
	queries.Set("conference_id", "0")
	req.URL.RawQuery = queries.Encode()

	resp, err := client.Do(req)
	if err != nil {
		sr.lg.Error("Storage response failed", zap.Error(err))
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		sr.lg.Error("Storage response failed", zap.Int("Status code", resp.StatusCode))
		return nil, ErrorFailedMsgRequest
	}

	buf := make([]byte, 1e5) // bad const change !!!
	n, err := resp.Body.Read(buf)
	if err != nil && err != io.EOF {
		sr.lg.Error("New messages failed to read response buffer", zap.Error(err))
		return nil, err
	}

	respData, err := storage_response.DecodeResponseFromBytes(buf[:n])
	if err != nil && err != io.EOF {
		sr.lg.Error("New messages failed to decode response", zap.Error(err))
		return nil, err
	}

	sr.lg.Debug("Successfully get new messages", zap.Int("Msgs amount", len(respData.GetMsgs())), zap.Int64("last message time stamp", respData.GetLastMessageTimeStamp()))
	if lMsgTst := respData.GetLastMessageTimeStamp(); lMsgTst > sr.GetLastMessageTimeStamp() {
		sr.SetLastMessageTimeStamp(lMsgTst)
	}

	return respData.GetMsgs(), nil
}

const GetLastMessagesQuery = `SELECT userid, username, text, timestamp FROM messages ORDER BY timestamp DESC LIMIT $1;`

func (sr *StorageRepo) GetLastKMessages(k int) ([]message.Message, error) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/get_newbie", sr.sAddr), http.NoBody)
	if err != nil {
		sr.lg.Error("Failed to create request to get newbie messages", zap.Error(err))
		return nil, err
	}

	queries := req.URL.Query()
	queries.Set("conference_id", "0")
	queries.Set("amount", strconv.Itoa(k))
	req.URL.RawQuery = queries.Encode()

	resp, err := client.Do(req)
	if err != nil {
		sr.lg.Error("Newbie messages request failed", zap.Error(err))
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorFailedMsgRequest
	}

	buf := make([]byte, 1e5) // bad const change !!!
	n, err := resp.Body.Read(buf)
	if err != nil && err != io.EOF {
		sr.lg.Error("Newbie messages failed to read body buffer", zap.Error(err))
		return nil, err
	}

	if n == 0 {
		return []message.Message{}, nil
	}

	respData, err := storage_response.DecodeResponseFromBytes(buf[:n])
	if err != nil {
		sr.lg.Error("Newbie messages failed to decode response", zap.Error(err))
		return nil, err
	}

	sr.lg.Debug("Successfully get newbie messages", zap.Int("Msgs amount", len(respData.GetMsgs())), zap.Int64("last message time stamp", respData.GetLastMessageTimeStamp()))
	if lMsgTst := respData.GetLastMessageTimeStamp(); lMsgTst > sr.GetLastMessageTimeStamp() {
		sr.SetLastMessageTimeStamp(lMsgTst)
	}

	return respData.GetMsgs(), nil
}
func (sr *StorageRepo) AddMessage(m message.Message) error {
	sr.lg.Debug("Add new message", zap.Int("user id", m.GetUserId()), zap.Int("chat id", m.GetChatId()))
	buf, err := message.EncodeMsgsToBytes(m)
	if err != nil {
		sr.lg.Error("Failed to encode message to bytes", zap.Error(err))
		return err
	}
	go sr.producer.WriteMessage(buf)
	return nil
}

func (sr *StorageRepo) SetLastMessageTimeStamp(lMsgTimeStamp int64) {
	sr.lMsgTmSt = lMsgTimeStamp
}

func (sr *StorageRepo) GetLastMessageTimeStamp() int64 {
	return sr.lMsgTmSt
}

func (pr *StorageRepo) CloseRepo() error {
	return nil
}
