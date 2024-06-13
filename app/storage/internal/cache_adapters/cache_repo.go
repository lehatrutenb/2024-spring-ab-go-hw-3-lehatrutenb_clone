package cache_adapters

import "server/external/message"

type CacheRepository interface {
	AddMessage(message.Message, int64)
	CheckLastMsgTimeStamp(string, int64) (bool, error)
	CloseRepo() error
}
