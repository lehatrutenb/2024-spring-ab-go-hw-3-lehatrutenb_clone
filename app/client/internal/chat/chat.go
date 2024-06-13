package chat

import (
	"server/external/message"
)

type Chat interface {
	SendMessage(message.Message) error
	RecieveMessages() chan message.Message
}
