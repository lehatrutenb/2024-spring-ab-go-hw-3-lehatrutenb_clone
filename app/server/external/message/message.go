package message

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/fatih/color"
)

type Message struct {
	User string
	Text string
	UId  int
	CId  int
}

func (m Message) GetUserId() int {
	return m.UId
}

func (m *Message) SetUserId(uId int) {
	m.UId = uId
}

func (m Message) GetChatId() int {
	return m.CId
}

func (m *Message) SetChatId(cId int) {
	m.CId = cId
}

func DecodeMsgFromBytes(b []byte) (Message, error) {
	var buf *bytes.Buffer = bytes.NewBuffer(b)
	enc := gob.NewDecoder(buf)

	var msg Message
	if err := enc.Decode(&msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}

func DecodeArrMsgFromBytes(b []byte) ([]Message, error) {
	var buf *bytes.Buffer = bytes.NewBuffer(b)
	enc := gob.NewDecoder(buf)

	var msg []Message
	if err := enc.Decode(&msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func EncodeMsgsToBytes(msgs any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(msgs); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (msg Message) BeautifulPrint() string {
	return fmt.Sprintf("%s:\n%s\n\n", color.CyanString("%s", msg.User), msg.Text)
}
