/*
des   : 全局类型定义
create:
author: chh
*/
package typedefs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	ServerIP   string
	ServerPort int
	LogFile    string
}

func (this *Config) LoadFromJson(jsonfile string) {
	r, err := os.Open(jsonfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer r.Close()

	decoder := json.NewDecoder(r)
	err = decoder.Decode(this)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	ServerConfig Config
	Logger       *log.Logger
)

/*
读配置; 获取日志接口
*/
func init() {
	// make default config
	ServerConfig.ServerIP = "127.0.0.1"
	ServerConfig.ServerPort = 19000
	ServerConfig.LogFile = "./Default.log"

	// load config from file
	fmt.Println("Default Config:", ServerConfig)
	ServerConfig.LoadFromJson("config.json")
	fmt.Println("Config After Load:", ServerConfig)

	// init logger
	logfile, err := os.OpenFile(ServerConfig.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0)
	if err != nil {
		fmt.Printf("%s\r\n", err.Error())
		os.Exit(-1)
	}
	Logger = log.New(logfile, "\r\n", log.Ldate|log.Ltime)
	if Logger == nil {
		fmt.Println("make Logger failed")
		os.Exit(-1)
	}
}
