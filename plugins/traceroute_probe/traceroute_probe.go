package traceroute_probe

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	cds "mda-traceroute-go/dataStruct"
	"mda-traceroute-go/plugins/traceroute_probe/mda"
	"mda-traceroute-go/plugins/traceroute_probe/utils"
	"mda-traceroute-go/plugins/traceroute_probe/ws"
	"mda-traceroute-go/util"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const PluginName = "traceroute_probe"

type TracerouteProbe struct {
	WsConn      *websocket.Conn
	CommandChan chan []byte
	ResultChan  chan []byte

	SrcAddr net.IP

	CurrentProbeNum uint16 // 当前执行的任务数
	MaxProbeNum     uint16 // 探测节点最多同时执行几个任务
	taskMap         map[string]*mda.ICMPApp

	taskEndCh chan string // 任务消亡或结束时主动注销

	MaxTTL     uint8
	Protocol   string
	PacketRate float64 // PPS 单位：个/秒

	NetSrcAddr net.IP
	NetDstAddr net.IP

	StopSign int32 // 停止标志

	ReportSign *int32

	Lock sync.RWMutex
}

func NewTracerouteProbe(maxProbeNum uint16, maxTTL uint8, protocol string, packetRate float64) *TracerouteProbe {
	return &TracerouteProbe{
		CurrentProbeNum: 0,
		MaxProbeNum:     maxProbeNum,
		MaxTTL:          maxTTL,
		Protocol:        protocol,
		PacketRate:      packetRate,
		StopSign:        0,
	}
}

func (tp *TracerouteProbe) Start() {
	var err error

	// 连接到控制节点
	tp.initWsConn()

	reloadDur := time.After(time.Second *
		time.Duration(utils.ConfigData.ReloadConfDuration))
	for {
		select {
		case c := <-tp.CommandChan:
			msg := &cds.Message{}
			err = json.Unmarshal(c, msg)
			if err != nil {
				logrus.Errorf("%v", err)
			}
			// 解析服务端传来的命令
			if msg.MsgType == "heartbeat" {
				continue
			} else if msg.MsgType == "taskNum" {
				tp.Lock.RLock()
				num := tp.CurrentProbeNum
				tp.Lock.RUnlock()
				send := &cds.Message{
					MsgType:  "taskNum",
					Msg:      strconv.Itoa(int(num)),
					Datetime: util.FormatNow(),
				}
				sendBytes, err := json.Marshal(send)
				if err != nil {
					logrus.Errorf("%v", err)
				}
				tp.ResultChan <- sendBytes

			} else if msg.MsgType == "dst" {
				logrus.Infof("Start the tracert mission. Message[%v]", msg)
				if tp.CurrentProbeNum >= tp.MaxProbeNum {
					logrus.Warningf("task come up to MaxProbeNum. CurrentProbeNum:%d, MaxProbeNum:%d.\n",
						tp.CurrentProbeNum, tp.MaxProbeNum)
					send := &cds.Message{
						MsgType:  "overflow",
						Msg:      "",
						Datetime: util.FormatNow(),
					}
					sendBytes, err := json.Marshal(send)
					if err != nil {
						logrus.Errorf("%v", err)
					}
					tp.ResultChan <- sendBytes

				} else {
					dstAddr, err := tp.verifyConf(msg.Msg, tp.MaxTTL)
					if err != nil {
						logrus.Errorf("%v", err)
					}
					datetime, err := time.Parse("2006-01-02 15:04:05", msg.Datetime)
					if err != nil {
						logrus.Errorf("%v", err)
					}
					hash := utils.GetHash(tp.SrcAddr.To4(), dstAddr.To4(), 65535, 65535, 1)
					app := mda.NewICMPApp(hash, dstAddr, tp.SrcAddr, tp.MaxTTL, true, datetime.UnixMicro())
					go app.Start()
					go tp.Report(hash, time.Duration(utils.ConfigData.ReportFreq))
					tp.Lock.Lock()
					tp.CurrentProbeNum++
					tp.taskMap[hash] = app
					tp.Lock.Unlock()

					send := &cds.Message{
						MsgType:  "success",
						Msg:      msg.Msg,
						Datetime: util.FormatNow(),
					}
					sendBytes, err := json.Marshal(send)
					if err != nil {
						logrus.Errorf("%v", err)
					}
					tp.ResultChan <- sendBytes
				}
			} else {
				continue
			}
		case <-reloadDur:
			// 重载配置文件
			utils.ReloadConfig()
			reloadDur = time.After(time.Second *
				time.Duration(utils.ConfigData.ReloadConfDuration))
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (tp *TracerouteProbe) Stop() {
	atomic.StoreInt32(&tp.StopSign, 1)
}

func (tp *TracerouteProbe) initWsConn() {
	// 根据配置文件合成ws请求地址
	tp.WsConn = ws.ConnWsServer(utils.ConfigData.WebSocketConf.Server, utils.ConfigData.WebSocketConf.Port,
		tp.CommandChan, tp.ResultChan)

	if tp.WsConn == nil {
		logrus.Errorf("conn ws server failed!")
	}
}

// 验证配置信息
func (tp *TracerouteProbe) verifyConf(dst string, maxTTL uint8) (net.IP, error) {
	var dstAddr net.IP
	var err error
	if util.MatchDst(dst) == 0 {
		// 取得目的域名的 IP
		addr, err := net.LookupIP(dst)
		logrus.Infof("Dst domain IPs: %v", addr)
		if err != nil {
			logrus.Fatal("verifyConf() Dst domain lookup error:", err)
		}
		dstAddr = addr[0]
	} else {
		// 字符串的IP转为 net.IP类型 这样写是错误的
		//t.netDstAddr = net.IP(t.Dst)
		dst, err := net.ResolveIPAddr("ip4", dst)
		if err != nil {
			logrus.Errorf("dst ip resolve error. %v", err)
		}
		dstAddr = dst.IP
	}

	// 验证源IP是否有问题
	err = tp.verifySrcAddr(false)
	if err != nil {
		logrus.Fatal(err)
	}

	if maxTTL > 64 {
		logrus.Warn("Large TTL may cause low performance")
	}
	return dstAddr, err
}

// force参数表示是否强制更新本地源地址
func (tp *TracerouteProbe) verifySrcAddr(force bool) error {
	if !force && string(tp.SrcAddr) != "" {
		return nil
	}

	// 设置本地IP为源地址
	conn, err := net.Dial("udp", "114.114.114.114:53")
	if err != nil {
		logrus.Fatal(err)
	}
	local := conn.LocalAddr().(*net.UDPAddr)
	conn.Close()
	tp.SrcAddr = local.IP
	return nil
}

func (tp *TracerouteProbe) Report(hash string, freq time.Duration) {
	app := tp.taskMap[hash]
	for {
		if atomic.LoadUint32(&app.Exit) == 1 {
			app.GracefulClose(1)
			for ttl := 1; ttl <= int(tp.MaxTTL); ttl++ {
				m := app.ResMap[ttl]
				for _, v := range m {
					sr, err := json.Marshal(v)
					if err != nil {
						logrus.Fatal(err)
					}
					tp.ResultChan <- sr
					logrus.Infof("Report data: %v", string(sr))
				}
			}
			end := cds.Message{
				MsgType:  "end",
				Msg:      string(tp.taskMap[hash].DstAddr),
				Datetime: time.Now().Format("2006-01-02 15:04:05"),
			}
			endFlag, err := json.Marshal(end)
			if err != nil {
				logrus.Errorf("json end flag error: %v", err)
			}
			tp.ResultChan <- endFlag
			tp.Lock.Lock()
			delete(tp.taskMap, hash)
			tp.CurrentProbeNum--
			tp.Lock.Unlock()
			break
		}
		time.Sleep(freq * time.Second)
	}
}
