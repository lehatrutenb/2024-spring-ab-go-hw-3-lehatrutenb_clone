package storage_response

import (
	"bytes"
	"encoding/gob"
	"server/external/message"
)

type StorageResponse struct {
	Msgs                    []message.Message
	Last_message_time_stamp int64
	ErrExplanation          string
}

func NewResponse(msgs []message.Message, lMsgTimeSt int64) StorageResponse {
	return StorageResponse{Msgs: msgs, Last_message_time_stamp: lMsgTimeSt}
}

func (r *StorageResponse) SetErrExplanation(expl string) {
	r.ErrExplanation = expl
}

func (r StorageResponse) GetErrExplanation() string {
	return r.ErrExplanation
}

func (r StorageResponse) GetMsgs() []message.Message {
	return r.Msgs
}

func (r StorageResponse) GetLastMessageTimeStamp() int64 {
	return r.Last_message_time_stamp
}

func DecodeResponseFromBytes(b []byte) (StorageResponse, error) {
	var buf *bytes.Buffer = bytes.NewBuffer(b)
	enc := gob.NewDecoder(buf)

	var resp StorageResponse
	if err := enc.Decode(&resp); err != nil {
		return StorageResponse{}, err
	}
	return resp, nil
}

func EncodeResponseToBytes(resp StorageResponse) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(resp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
