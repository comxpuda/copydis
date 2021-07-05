package copydis

import (
	"copydis/config"
	"copydis/lib/utils"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"
)

func TestAof(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "copydis")
	if err != nil {
		t.Error(err)
		return
	}
	aofFilename := path.Join(tmpDir, "a.aof")
	defer func() {
		_ = os.Remove(aofFilename)
	}()
	config.Properties = &config.ServerProperties{
		AppendOnly:     true,
		AppendFilename: aofFilename,
	}
	aofWriteDB := MakeDB()
	size := 10
	keys := make([]string, 0)
	cursor := 0
	for i := 0; i < size; i++ {
		key := strconv.Itoa(cursor)
		cursor++
		execSet(aofWriteDB, utils.ToCmdLine(key, utils.RandString(8), "EX", "10000"))
		keys = append(keys, key)
	}
}
