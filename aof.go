package copydis

import (
	"copydis/config"
	"copydis/datastruct/dict"
	"copydis/datastruct/lock"
	"copydis/lib/logger"
	"copydis/lib/utils"
	"copydis/redis/parser"
	"copydis/redis/reply"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

var pExpireAtBytes = []byte("PEXPIREAT")

func makeExpireCmd(key string, expireAt time.Time) *reply.MultiBulkReply {
	args := make([][]byte, 3)
	args[0] = pExpireAtBytes
	args[1] = []byte(key)
	args[2] = []byte(strconv.FormatInt(expireAt.UnixNano()/1e6, 10))
	return reply.MakeMultiBulkReply(args)
}

func makeAofCmd(cmd string, args [][]byte) *reply.MultiBulkReply {
	params := make([][]byte, len(args)+1)
	copy(params[1:], args)
	params[0] = []byte(cmd)
	return reply.MakeMultiBulkReply(params)
}

// AddAof send command to aof goroutine through channel
func (db *DB) AddAof(args *reply.MultiBulkReply) {
	// aofChan == nil when loadAof
	if config.Properties.AppendOnly && db.aofChan != nil {
		db.aofChan <- args
	}
}

// handleAof listen aof channel and write into file
func (db *DB) handleAof() {
	for cmd := range db.aofChan {
		db.pausingAof.RLock() // prevent other goroutines from pausing aof
		if db.aofRewriteBuffer != nil {
			// replica during rewrite
			db.aofRewriteBuffer <- cmd
		}
		_, err := db.aofFile.Write(cmd.ToBytes())
		if err != nil {
			logger.Warn(err)
		}
		db.pausingAof.RUnlock()
	}
	db.aofFinished <- struct{}{}
}

// loadAof read aof file
func (db *DB) loadAof(maxBytes int) {
	// delete aofChan to prevent write again
	aofChan := db.aofChan
	db.aofChan = nil
	defer func(aofChan chan *reply.MultiBulkReply) {
		db.aofChan = aofChan
	}(aofChan)
	file, err := os.Open(db.aofFilename)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return
		}
		logger.Warn(err)
		return
	}
	defer file.Close()

	reader := utils.NewLimitedReader(file, maxBytes)
	ch := parser.ParseStream(reader)
	for p := range ch {
		if p.Err != nil {
			if p.Err == io.EOF {
				break
			}
			logger.Error("parse error: " + p.Err.Error())
			continue
		}
		if p.Data == nil {
			logger.Error("empty payload")
			continue
		}
		r, ok := p.Data.(*reply.MultiBulkReply)
		if !ok {
			logger.Error("require multi bulk reply")
			continue
		}
		cmd := strings.ToLower(string(r.Args[0]))
		command, ok := cmdTable[cmd]
		if ok {
			handler := command.executor
			handler(db, r.Args[1:])
		}
	}
}

/*-- aof rewrite --*/
func (db *DB) aofRewrite() {
	file, fileSize, err := db.startRewrite()
	if err != nil {
		logger.Warn(err)
		return
	}

	// load aof file
	tmpDB := &DB{
		data:   dict.MakeSimple(),
		ttlMap: dict.MakeSimple(),
		locker: lock.Make(lockerSize),

		aofFilename: db.aofFilename,
	}
	tmpDB.loadAof(int(fileSize))
	// rewrite aof file
	tmpDB.data.ForEach(func(key string, raw interface{}) bool {
		entity, _ := raw.(*DataEntity)
		cmd := EntityToCmd(key, entity)
		if cmd != nil {
			_, _ = file.Write(cmd.ToBytes())
		}
		return true
	})
	tmpDB.ttlMap.ForEach(func(key string, raw interface{}) bool {
		expireTime, _ := raw.(time.Time)
		cmd := makeExpireCmd(key, expireTime)
		if cmd != nil {
			_, _ = file.Write(cmd.ToBytes())
		}
		return true
	})

	db.finishRewrite(file)
}

func (db *DB) startRewrite() (*os.File, int64, error) {
	db.pausingAof.Lock() // pausing aof
	defer db.pausingAof.Unlock()

	err := db.aofFile.Sync()
	if err != nil {
		logger.Warn("fsync failed")
		return nil, 0, err
	}
	// create rewrite channel
	db.aofRewriteBuffer = make(chan *reply.MultiBulkReply, aofQueueSize)

	// get current aof file size
	fileInfo, _ := os.Stat(db.aofFilename)
	filesize := fileInfo.Size()

	// create tmp file
	file, err := ioutil.TempFile("", "aof")
	if err != nil {
		logger.Warn("tmp file create failed")
		return nil, 0, err
	}
	return file, filesize, nil
}

func (db *DB) finishRewrite(tmpFile *os.File) {
	db.pausingAof.Lock() // pausing aof
	defer db.pausingAof.Unlock()

	// write commands created during rewriting to tmp file
loop:
	for {
		// aof is pausing, there won't be any new commands in aofRewriteBuffer
		select {
		case cmd := <-db.aofRewriteBuffer:
			_, err := tmpFile.Write(cmd.ToBytes())
			if err != nil {
				logger.Warn(err)
			}
		default:
			// channel is empty, break loop
			break loop
		}
	}
	close(db.aofRewriteBuffer)
	db.aofRewriteBuffer = nil

	// replace current aof file by tmp file
	_ = db.aofFile.Close()
	_ = os.Rename(tmpFile.Name(), db.aofFilename)

	// reopen aof file for further write
	aofFile, err := os.OpenFile(db.aofFilename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}
	db.aofFile = aofFile
}
