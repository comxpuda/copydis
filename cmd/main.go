package main

import (
	"copydis/config"
	"copydis/lib/logger"
	RedisServer "copydis/redis/server"
	"copydis/tcp"
	"fmt"
	"os"
)

var banner = `
___   ___   _ __   _   _   __| |(_) ___
/ __| / _ \ | '_ \ | | | | / _ || |/ __|
| (__ | (_) || |_) || |_| || (_| || |\__ \
\___| \___/ | .__/  \__, | \__,_||_||___/
		   |_|     |___/
`

var defaultProperties = &config.ServerProperties{
	Bind:           "0.0.0.0",
	Port:           6399,
	AppendOnly:     false,
	AppendFilename: "",
	MaxClients:     1000,
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

func main() {
	print(banner)
	logger.Setup(&logger.Settings{
		Path:       "logs",
		Name:       "copydis",
		Ext:        ".log",
		TimeFormat: "2006-01-02",
	})
	configFilename := os.Getenv("CONFIG")
	if configFilename == "" {
		if fileExists("redis.conf") {
			config.SetupConfig("redis.conf")
		} else {
			config.Properties = defaultProperties
		}
	} else {
		config.SetupConfig(configFilename)
	}
	err := tcp.ListenAndServeWithSignal(&tcp.Config{
		Address: fmt.Sprintf("%s:%d", config.Properties.Bind, config.Properties.Port),
	}, RedisServer.MakeHandler())
	if err != nil {
		logger.Error(err)
	}
}
