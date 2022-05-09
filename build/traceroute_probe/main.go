package main

import (
	tp "mda-traceroute-go/plugins/traceroute_probe"
	tpu "mda-traceroute-go/plugins/traceroute_probe/utils"
	"mda-traceroute-go/util"
)

var (
	DefaultMaxProbeNum = uint16(1)
	DefaultMaxTTL      = uint8(64)
	DefaultProtocol    = "icmp"
	DefaultPacketRate  = 1.0
)

func main() {
	// 加载配置
	err := tpu.ParseConfig("probe.toml")
	if err != nil {
		return
	}
	// 初始化logrus
	util.InitLog(tp.PluginName)

	probe := tp.NewTracerouteProbe(DefaultMaxProbeNum, DefaultMaxTTL, DefaultProtocol, DefaultPacketRate)
	probe.Start()
}
