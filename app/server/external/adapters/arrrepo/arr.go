package maprepo

import (
	"server/external/message"
	"sync"
)

type MapRepo struct {
	data []message.Message
	mu   *sync.RWMutex
}

func NewRepo() *MapRepo {
	return &MapRepo{make([]message.Message, 0), &sync.RWMutex{}}
}

func (mr *MapRepo) WriteMessage(msg message.Message) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.data = append(mr.data, msg)
	return nil
}

func (mr MapRepo) ReadMessage(lastMsgId *int) (message.Message, bool, error) {
	if *lastMsgId < 0 {
		*lastMsgId = 0
	}

	if *lastMsgId >= len(mr.data) {
		return message.Message{}, false, nil
	}
	defer func() { *lastMsgId += 1 }()
	return mr.data[*lastMsgId], true, nil
}

func (mr MapRepo) GetKthNewestMessageId(k int) int {
	return max(0, len(mr.data)-10)
}
