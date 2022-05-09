package utils

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"mda-traceroute-go/util"
	"os"
	"sync"
)

type Config struct {
	RunArgs
	WebSocketConf
	MatchCacheConf
}

type RunArgs struct {
	FirstSendCnt       uint    `toml:"firstSendCnt"`
	PacketRate         float64 `toml:"packetRate"`
	ReloadConfDuration uint    `toml:"reloadConfDuration"`
	ReportFreq         uint8   `toml:"reportFreq"`
}

type WebSocketConf struct {
	Server string `toml:"server"`
	Port   uint16 `toml:"port"`
	Group  string `toml:"group"`
}

type MatchCacheConf struct {
	Timeout   uint8 `toml:"timeout"`
	CheckFreq uint8 `toml:"checkFreq"`
}

var (
	ConfigFile string
	ConfigData *Config
	ConfigLock sync.RWMutex

	lastModifyTime int64 = 0
	currModifyTime int64
)

// ParseConfig 解析配置文件
func ParseConfig(cfg string) error {
	var c Config
	err := util.ParseConfigToml(cfg, &c)
	if err != nil {
		logrus.Fatal("read config file ", cfg, " error: ", err)
		return err
	}

	ConfigLock.Lock()
	ConfigFile = cfg
	ConfigData = &c
	ConfigLock.Unlock()
	logrus.Infof("parse config success.")

	return nil
}

func ReloadConfig() {
	if !isModify() {
		return
	}
	var c Config
	err := util.ParseConfigToml(ConfigFile, &c)
	if err != nil {
		logrus.Errorf("reload config " + ConfigFile + " failed!")
		return
	}

	ConfigLock.Lock()
	ConfigData = &c
	ConfigLock.Unlock()
	logrus.Errorf("reload config success.")
}

func isModify() bool {
	file, err := os.Open(ConfigFile)
	if err != nil {
		return false
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		fmt.Printf("File stat error:%s\n", err)
		return false
	}
	currModifyTime = fileStat.ModTime().Unix()
	if currModifyTime > lastModifyTime {
		lastModifyTime = currModifyTime
		return true
	}
	return false
}
