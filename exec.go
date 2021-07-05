package copydis

import (
	"copydis/interface/redis"
	"copydis/lib/logger"
	"copydis/redis/reply"
	"fmt"
	"runtime/debug"
	"strings"
)

// Exec executes command
// parameter `cmdLine` contains command and its arguments, for example: "set key value"
func (db *DB) Exec(c redis.Connection, cmdLine [][]byte) (result redis.Reply) {
	defer func() {
		if err := recover(); err != nil {
			logger.Warn(fmt.Sprintf("error occurs: %v\n%s", err, string(debug.Stack())))
			result = &reply.UnknownErrReply{}
		}
	}()
	cmdName := strings.ToLower(string(cmdLine[0]))
	if cmdName == "auth" {
		return Auth(db, c, cmdLine[1:])
	}
	if !isAuthenticated(c) {
		return reply.MakeErrReply("NOAUTH Authentication required")
	}

	// special commands
	done := false
	result, done = execSpecialCmd(c, cmdLine, cmdName, db)
	if done {
		return result
	}
	if c != nil && c.InMultiState() {
		return EnqueueCmd(db, c, cmdLine)
	}
	// normal command
	return execNormalCommand(db, cmdLine)
}

func execSpecialCmd(c redis.Connection, cmdLine [][]byte, cmdName string, db *DB) (redis.Reply, bool) {
	if cmdName == "subscribe" {
		if len(cmdLine) < 2 {
			return reply.MakeArgNumErrReply("subscribe"), true
		}
		// return pubsub.Subscribe(db.hub, c, cmdLine[1:]), true
	} else if cmdName == "publish" {
		// return pubsub.Publish(db.hub, cmdLine[1:]), true
	} else if cmdName == "unsubscribe" {
		// return pubsub.UnSubscribe(db.hub, c, cmdLine[1:]), true
	} else if cmdName == "bgrewriteaof" {
		// aof.go imports router.go, router.go cannot import BGRewriteAOF from aof.go
		return BGRewriteAOF(db, cmdLine[1:]), true
	} else if cmdName == "multi" {
		if len(cmdLine) != 1 {
			return reply.MakeArgNumErrReply(cmdName), true
		}
		return StartMulti(db, c), true
	} else if cmdName == "discard" {
		if len(cmdLine) != 1 {
			return reply.MakeArgNumErrReply(cmdName), true
		}
		return DiscardMulti(db, c), true
	} else if cmdName == "exec" {
		if len(cmdLine) != 1 {
			return reply.MakeArgNumErrReply(cmdName), true
		}
		return execMulti(db, c), true
	} else if cmdName == "watch" {
		if !validateArity(-2, cmdLine) {
			return reply.MakeArgNumErrReply(cmdName), true
		}
		return Watch(db, c, cmdLine[1:]), true
	}
	return nil, false
}
