package adapters

import "server/external/message"

type Repository interface {
	AddMessage(message.Message) error
	GetNewerMessages() ([]message.Message, error)
	GetLastKMessages(int) ([]message.Message, error)
	SetLastMessageTimeStamp(int64)
	GetLastMessageTimeStamp() int64
	CloseRepo() error
}
