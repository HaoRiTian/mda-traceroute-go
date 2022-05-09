package traceroute_agg

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"mda-traceroute-go/dataStruct"
	"mda-traceroute-go/db/dao"
	"mda-traceroute-go/plugins/traceroute_agg/geoip"
	"mda-traceroute-go/plugins/traceroute_agg/ws"
	"strconv"
	"sync"
	"time"
)

const PluginName = "Traceroute_agg"

type TracerouteAgg struct {
	Dst         string
	Group       string
	NodeNum     int32
	TracertTime time.Time

	WsManager *ws.Manager
	Result    map[uint8][]*dao.Topo
	Lock      sync.Mutex

	// 完成的标志
	Complete chan bool
}

func NewTracerouteAgg(dst string, group string, nodeNum int32, tracertTime time.Time, wsManager *ws.Manager) (*TracerouteAgg, error) {

	ta := &TracerouteAgg{
		Dst:         dst,
		Group:       group,
		NodeNum:     nodeNum,
		WsManager:   wsManager,
		TracertTime: tracertTime,
		Result:      make(map[uint8][]*dao.Topo, 1024),
		// 不设置空间，写端不写入，读端就阻塞
		Complete: make(chan bool),
	}

	var err error = nil
	cnt, ok := ta.WsManager.LenClientInGroup(group)
	if !ok || cnt <= 0 {
		err = fmt.Errorf("error! This group [%s] don't have node", ta.Group)
	}
	return ta, err
}

func (ta *TracerouteAgg) Start() {
	// 将探测记录插入 TracertRecord 表
	ta.insertTracertRecord()

	if ta.Group == "All" {
		if ta.NodeNum >= 0 {
			ta.WsManager.SendAll([]byte(ta.Dst))
		} else {
			// 每组挑选 |NodeNum| 个节点发送

		}
	} else {
		notice := &dataStruct.Message{
			MsgType:  "dst",
			Msg:      ta.Dst,
			Datetime: ta.TracertTime.Format("2006-01-02 15:04:05"),
		}
		msg, err := json.Marshal(notice)
		if err != nil {
			logrus.Errorf("%v", err)
		}
		ta.WsManager.SendGroup(ta.Group, msg)
	}

	// 接收子节点传来的数据
	go ta.recvData(ta.Group)
}

func (ta *TracerouteAgg) recvData(group string) {
	if group == "All" {
		for i := 0; i < int(ta.WsManager.LenGroup()); i++ {

		}
	} else {
		var wg sync.WaitGroup
		for _, v := range ta.WsManager.Group[group] {
			if cnt, ok := ta.WsManager.LenClientInGroup(group); ok {
				wg.Add(int(cnt))
				logrus.Infof("waitgroup add %d", cnt)
			}
			go func(c *ws.Client) {
				defer func() {
					err := recover()
					if err != nil {
						logrus.Errorf("%v", err)
					}
					wg.Done()
				}()

			loop:
				for {
					if msg, ok := <-c.ToBeReadMessage; ok {
						var msgIntf interface{}
						err := json.Unmarshal(msg, &msgIntf)
						if err != nil {
							logrus.Errorf("json unmarshal ToBeReadMessage error: %v", err)
							continue
						}
						switch t := msgIntf.(type) {
						case dataStruct.RouteInfo:
							ta.Lock.Lock()
							loc, err := geoip.GlobalGeoIP.Lookup(t.ResAddr)
							var topo *dao.Topo
							mean, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", t.LatencyStat.Mean), 64)

							if err != nil {
								logrus.Errorf("%v", err)

								topo = &dao.Topo{
									Domain:      t.Domain,
									TTL:         t.TTL,
									DstIP:       t.DstIP,
									ResAddr:     t.ResAddr,
									Name:        t.Name,
									Session:     t.Session,
									MeanLatency: mean,
									RecvCnt:     t.RecvCnt,
									Country:     "-",
									Region:      "-",
									City:        "-",
									ISP:         "-",
									TracertTime: time.UnixMicro(t.TimeStamp),
								}
							} else {
								topo = &dao.Topo{
									Domain:      t.Domain,
									TTL:         t.TTL,
									DstIP:       t.DstIP,
									ResAddr:     t.ResAddr,
									Name:        t.Name,
									Session:     t.Session,
									MeanLatency: mean,
									RecvCnt:     t.RecvCnt,
									Country:     loc.Country,
									Region:      loc.Region,
									City:        loc.City,
									ISP:         loc.SPName,
									TracertTime: time.UnixMicro(t.TimeStamp),
								}
							}
							ta.Result[topo.TTL] = append(ta.Result[topo.TTL], topo)
							ta.Lock.Unlock()
						case dataStruct.Message:
							if t.MsgType == "end" {
								logrus.Infof("client [%v] all data is received. ", c.Id)
								break loop
							}
						}
					} else {
						logrus.Errorf("client [%v] 's chan ToBeReadMessage isn't ok.", c.Id)
						break
					}
				}
			}(v)
		}
		wg.Wait()
		ta.Complete <- true
	}
}

func (ta *TracerouteAgg) insertTopo(domain string, ttl uint8, dstIp string, resAddr string,
	name string, session string, meanLatency float64, recvCnt uint64, tracertTime time.Time) {

	topo := &dao.Topo{
		Domain:      domain,
		TTL:         ttl,
		DstIP:       dstIp,
		ResAddr:     resAddr,
		Name:        name,
		Session:     session,
		MeanLatency: meanLatency,
		RecvCnt:     recvCnt,
		TracertTime: tracertTime,
	}
	_, err := dao.GlobalTopoData.InsertTopo(topo)
	if err != nil {
		logrus.Errorf("%v", err)
		return
	}
}

func (ta *TracerouteAgg) insertTracertRecord() {
	tr := &dao.TracertRecord{
		Dst:         ta.Dst,
		Group:       ta.Group,
		NodeNum:     ta.NodeNum,
		TracertTime: ta.TracertTime,
	}
	dao.GlobalTracertRecordData.InsertTracertRecord(tr)
}
