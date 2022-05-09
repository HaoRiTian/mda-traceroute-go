package mda

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/ipv4"
	ds "mda-traceroute-go/plugins/traceroute_probe/dataStruct"
	"mda-traceroute-go/plugins/traceroute_probe/utils"
	"mda-traceroute-go/util"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type ICMPType uint8

// ICMP的类型字段种类
var (
	ICMPEchoReply                     ICMPType
	ICMPDestUnreachable               ICMPType = 3
	ICMPSourceQuench                  ICMPType = 4
	ICMPRedirect                      ICMPType = 5
	ICMPAlternateHostAddr             ICMPType = 6
	ICMPEchoRequest                   ICMPType = 8
	ICMPRouterAdv                     ICMPType = 9
	ICMPRouterSol                     ICMPType = 10
	ICMPTimeExceeded                  ICMPType = 11
	ICMPParamProblem                  ICMPType = 12
	ICMPTimestampReq                  ICMPType = 13
	ICMPTimestampReply                ICMPType = 14
	ICMPAddrMaskReq                   ICMPType = 17
	ICMPAddrMaskReply                 ICMPType = 18
	ICMPTraceroute                    ICMPType = 30
	ICMPConversionErr                 ICMPType = 31
	ICMPMobileHostRedirect            ICMPType = 32
	ICMPIPv6WhereAreYou               ICMPType = 33
	ICMPIPv6IAmHere                   ICMPType = 34
	ICMPMobileRegistrationReq         ICMPType = 35
	ICMPMobileRegistrationReply       ICMPType = 36
	ICMPDomainNameReq                 ICMPType = 37
	ICMPDomainNameReply               ICMPType = 38
	ICMPSkipAlgoDiscoveryProtocol     ICMPType = 39
	ICMPPhoturis                      ICMPType = 40
	ICMPExperimentalMobilityProtocols ICMPType = 41
)

// ICMPHeader ICMP头部字段
type ICMPHeader struct {
	IType    ICMPType
	ICode    ICMPCode
	Checksum uint16
	ID       uint16
	Seq      uint16
}

// ICMPCode ICMP的code字段
type ICMPCode uint8

// 终点不可达即 icmpType=3时，code字段有5种情况
/*
   0 = net unreachable;
   1 = host unreachable;
   2 = protocol unreachable;
   3 = port unreachable;
   4 = fragmentation needed and DF set.
*/
var (
	NetUnreachable              ICMPCode = 0
	HostUnreachable             ICMPCode = 1
	ProtocolUnreachable         ICMPCode = 2
	PortUnreachable             ICMPCode = 3
	FragmentationNeededAndDFSet ICMPCode = 4
)

type ICMPApp struct {
	key     string
	srcAddr net.IP
	DstAddr net.IP
	maxTTL  uint8

	matchCache    *MatchCache
	RecvConn      *net.IPConn
	ResMap        []map[string]*ds.ProbeResponse
	ResTTL        []uint8           // 排序插入
	ResFlowIDMap  map[string]uint32 //记录探测到某端口用的流标签
	ResFlowIDLock sync.RWMutex

	SendChan chan *ds.SendPacket
	RecvChan chan *ds.RecvPacket

	simple bool

	TaskGeneTs int64
	TaskEndTs  int64

	Exit      uint32 // 0 means not exit
	ExcepFlag uint32
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewICMPApp(hash string, dstAddr net.IP, srcAddr net.IP, maxTTL uint8, simple bool, taskGeneTs int64) *ICMPApp {

	cacheConf := utils.ConfigData.MatchCacheConf
	matchCache := NewMatchCache(hash, cacheConf.Timeout, cacheConf.CheckFreq)

	ctx, cancel := context.WithCancel(context.Background())
	go matchCache.Cache.RunCheck()
	return &ICMPApp{
		key:          hash,
		srcAddr:      srcAddr,
		DstAddr:      dstAddr,
		maxTTL:       maxTTL,
		matchCache:   matchCache,
		ResMap:       make([]map[string]*ds.ProbeResponse, 256),
		ResTTL:       make([]uint8, 256),
		ResFlowIDMap: make(map[string]uint32, 256),
		SendChan:     make(chan *ds.SendPacket, 10),
		RecvChan:     make(chan *ds.RecvPacket, 10),
		simple:       simple,
		TaskGeneTs:   taskGeneTs,
		Exit:         0,
		ExcepFlag:    0,
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (app *ICMPApp) Start() {
	go app.ListenFor()
	go app.SendPacket()
	go app.match()

	if app.simple {
		// 简单地探测，未进行多路径探测
	} else { // mda
		time.Sleep(20 * time.Second)
		app.mda()
	}

	// 检查是否在运行
	for {
		after := time.After(10 * time.Second)
		select {
		case <-after:
			if app.matchCache.Cache.Len() == 0 {
				app.GracefulClose(1)
				return
			}
		}
	}
}

// SendPacket 构建ICMP报文并发送
func (app *ICMPApp) SendPacket() {
	logrus.Infof("Start send ICMP Datagram. netSrcAddr: %v", app.srcAddr)

	conn, err := net.ListenPacket("ip4:icmp", app.srcAddr.String())
	if err != nil {
		logrus.Fatal(err)
	}
	defer conn.Close()

	rSocket, err := ipv4.NewRawConn(conn)
	if err != nil {
		logrus.Fatal("Can't create raw socket, ", err)
	}
	id := uint16(1)
	mod := uint16(1 << 15)

	cnt := uint(0)
	for {
		for ttl := 1; ttl <= int(app.maxTTL); ttl++ {
			hdr, payload := app.buildICMP(uint8(ttl), id, id, 0)
			_ = rSocket.WriteTo(hdr, payload, nil)
			report := &ds.SendPacket{
				Key:       app.key,
				ID:        uint32(hdr.ID),
				TTL:       uint8(ttl),
				TimeStamp: time.Now().UnixMicro(),
			}
			app.SendChan <- report
			id = (id + 1) % mod
		}
		cnt++

		//if cnt >= 6 {
		if cnt >= utils.ConfigData.FirstSendCnt {
			break
		}
		time.Sleep(time.Microsecond * time.Duration(1000000/utils.ConfigData.PacketRate))
	}
}

func (app *ICMPApp) ListenFor() {
	logrus.Info("Start Listen ICMP Datagram and receive.")
	localAddr := &net.IPAddr{IP: app.srcAddr}
	var err error
	app.RecvConn, err = net.ListenIP("ip4:icmp", localAddr)
	if err != nil {
		logrus.Fatal("ListenIP failure: ", err)
	}
	for {
		if atomic.LoadUint32(&app.Exit) == 1 {
			return
		}

		buf := make([]byte, 1500)
		n, recvAddr, err := app.RecvConn.ReadFrom(buf)
		if err != nil {
			logrus.Fatal(err)
			break
		}

		icmpType := ICMPType(buf[0])
		code := ICMPCode(buf[1])

		if (icmpType == ICMPTimeExceeded ||
			(icmpType == ICMPDestUnreachable && code == PortUnreachable)) && (n >= 36) {
			id := binary.BigEndian.Uint16(buf[32:34])

			dstIP := net.IP(buf[24:28])
			srcIP := net.IP(buf[20:24])

			if dstIP.Equal(app.DstAddr) {
				key := utils.GetHash(srcIP, dstIP, 65535, 65535, 1)
				m := &ds.RecvPacket{
					Key:       key,
					ID:        uint32(id),
					DstIP:     string(app.DstAddr),
					ResAddr:   recvAddr.String(),
					TimeStamp: time.Now().UnixMicro(),
				}
				app.RecvChan <- m
			}
		} else {
			logrus.Warningf("receive packet icmpType: %d, icmpCode: %d. \n", icmpType, code)
		}
	}
}

// 处理发送和接收的包
func (app *ICMPApp) match() {
	for {
		select {
		case <-app.ctx.Done():
			logrus.Errorf("exit match goroutine.")
			return
		case v := <-app.SendChan:
			app.matchCache.Cache.Store(v.ID, v, time.UnixMicro(v.TimeStamp))

		case v := <-app.RecvChan:
			logrus.Infof("Recv traceroute data: %+v", v)
			s, ok := app.matchCache.Cache.Load(v.ID)
			if !ok {
				logrus.Warningf("cache hasn't ID: %d packet.", v.ID)
				continue
			}
			sent := s.(*ds.SendPacket)
			util.SortInsertUint8(&app.ResTTL, sent.TTL)
			pr, ok := app.ResMap[sent.TTL][v.ResAddr]
			if !ok {
				pr = ds.NewProbeResponse(app.key, app.TaskGeneTs, sent.TTL, v.DstIP, v.ResAddr, v.ID, v.TimeStamp)
				app.ResMap[sent.TTL][v.ResAddr] = pr
			}
			pr.Lock.Lock()
			pr.Latency.Cnt++
			latency := float64((v.TimeStamp - sent.TimeStamp) / 1000) // 单位 ms
			pr.Latency.Append(latency, 4)
			pr.Lock.Unlock()

			app.ResFlowIDLock.Lock()
			app.ResFlowIDMap[v.ResAddr] = v.ID
			app.ResFlowIDLock.Unlock()

			// 为 isPerFlow 服务
			hash := string(sent.TTL) + ":" + strconv.Itoa(int(v.ID))
			app.matchCache.FlowIDCache.Store(hash, v.ResAddr, time.Now())
		}

	}
}

func (app *ICMPApp) buildICMP(ttl uint8, id, seq uint16, tos int) (*ipv4.Header, []byte) {
	hdr := &ipv4.Header{
		Version:  ipv4.Version,
		TOS:      tos,
		Len:      ipv4.HeaderLen,
		TotalLen: 40,
		ID:       int(id),
		Flags:    0,
		FragOff:  0,
		TTL:      int(ttl),
		Protocol: 1,
		Checksum: 0,
		Src:      app.srcAddr,
		Dst:      app.DstAddr,
	}
	h, err := hdr.Marshal()
	if err != nil {
		logrus.Fatal(err)
	}

	hdr.Checksum = int(utils.CheckSum(h))

	icmp := ICMPHeader{
		IType:    ICMPEchoRequest,
		ICode:    0,
		Checksum: util.If(app.simple, 0, id).(uint16),
		ID:       id,
		Seq:      seq,
	}

	payload := make([]byte, 32)
	for i := 0; i < 32; i++ {
		payload[i] = uint8(i + 64)
	}

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, &icmp)
	binary.Write(&buf, binary.BigEndian, &payload)
	return hdr, buf.Bytes()
}

//func (app *ICMPApp) forgeCheckSum(id uint16) uint16 {
//    checkSum := []byte("\x00\x00")
//
//    checkSum[0] = byte((id >> 8) & 0xff)
//    checkSum[1] = byte(id & 0xff)
//
//    val := binary.BigEndian.Uint16(checkSum)
//    return val
//}

func (app *ICMPApp) GracefulClose(d time.Duration) {
	atomic.StoreUint32(&app.Exit, 1)
	app.matchCache.Close()
	app.cancel()
}

func (app *ICMPApp) mda() {
	ttl := uint8(1)
	mapLen := len(app.ResTTL)
	for ; ttl < app.ResTTL[mapLen-1]; ttl++ {
		if len(app.ResMap[ttl]) <= 1 {
			continue
		} else { // 从ttl-1开始探测
			for k, _ := range app.ResMap[ttl-1] {
				app.nextHops(k, ttl)
				if app.isPerFlow(ttl) { // 说明k接口对应的网络有基于流的负载均衡
					app.nextHops(k, ttl)
				} else { // 碰到非基于流的负载均衡，就退出
					return
				}
			}
		}
	}
}

// 先判断addr接口对应的网络是否有基于流的负载均衡，有的话采用和adder一样的流标签，获取addr对应网络的所有下一条端口
func (app *ICMPApp) nextHops(addr string, ttl uint8) {
	app.ResFlowIDLock.RLock()
	id := app.ResFlowIDMap[addr]
	app.ResFlowIDLock.RUnlock()

	app.sendICMPWithTTLAndId(ttl, uint16(id))
}

// 判断 addr 所在路由器的负载均衡是否是基于流的
func (app *ICMPApp) isPerFlow(ttl uint8) bool {
	app.sendICMPWithTTLAndId(ttl, 333)

	time.Sleep(time.Duration(float64(utils.ConfigData.Timeout) + 6*utils.ConfigData.PacketRate))

	return !app.matchCache.FlowIDCache.Repeat
}

// 指定ttl和id发送ICMP报文
func (app *ICMPApp) sendICMPWithTTLAndId(ttl uint8, id uint16) {
	conn, err := net.ListenPacket("ip4:icmp", app.srcAddr.String())
	if err != nil {
		logrus.Fatal(err)
	}
	defer conn.Close()
	rSocket, err := ipv4.NewRawConn(conn)
	if err != nil {
		logrus.Fatal("Can't create raw socket, ", err)
	}

	mod := uint16(1 << 15)
	id = id % mod
	for {
		hdr, payload := app.buildICMP(ttl, id, id, 0)
		_ = rSocket.WriteTo(hdr, payload, nil)
		report := &ds.SendPacket{
			Key:       app.key,
			ID:        uint32(hdr.ID),
			TTL:       ttl,
			TimeStamp: time.Now().UnixMicro(),
		}
		app.SendChan <- report

		time.Sleep(time.Microsecond * time.Duration(1000000/utils.ConfigData.PacketRate))
	}
}
