package dataStruct

import (
	"mda-traceroute-go/plugins/traceroute_probe/linkInfo"
	"sync"
)

type SendPacket struct {
	Key       string
	ID        uint32
	TTL       uint8
	TimeStamp int64
}

type RecvPacket struct {
	Key       string
	ID        uint32
	DstIP     string
	ResAddr   string
	TimeStamp int64
}

type ProbeResponse struct {
	Key        string `json:"key"`
	TaskGeneTs int64  `json:"task-gene-ts"`
	//Header    *ipv4.Header
	Domain   string `json:"domain"`
	TTL      uint8  `json:"ttl"`
	DstIP    string `json:"dst-ip"`
	ResAddr  string `json:"res-addr"`
	FlowID   uint32
	FlowDiff bool  // FlowID 是否有变动
	CreateTs int64 `json:"create-ts""`
	Latency  *linkInfo.LatencyStat
	Lock     sync.RWMutex
}

func NewProbeResponse(key string, taskGeneTs int64, ttl uint8, dstIP string, resAddr string, flowId uint32, createTs int64) *ProbeResponse {
	return &ProbeResponse{
		Key:        key,
		TaskGeneTs: taskGeneTs,
		TTL:        ttl,
		DstIP:      dstIP,
		ResAddr:    resAddr,
		FlowID:     flowId,
		FlowDiff:   false,
		CreateTs:   createTs,
	}
}
