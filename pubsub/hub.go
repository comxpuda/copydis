package pubsub

import (
	"copydis/datastruct/dict"
	"copydis/datastruct/lock"
)

// Hub stores all subscribe relation
type Hub struct {
	// channnel -> list(*Client)
	subs dict.Dict
	// lock channel
	subsLocker *lock.Locks
}

func MakeHub() *Hub {
	return &Hub{
		subs:       dict.MakeConcurrent(4),
		subsLocker: lock.Make(16),
	}
}
