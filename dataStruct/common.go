package dataStruct

type RouteInfo struct {
	Domain      string `json:"domain"`
	TTL         uint8  `json:"ttl"`
	DstIP       string `json:"dst-ip"`
	ResAddr     string `json:"res-addr"`
	Name        string `json:"name"`
	Session     string `json:"session"`
	LatencyStat LatencyStat
	RecvCnt     uint64 `json:"recv-cnt"`
	TimeStamp   int64  `json:"ts"`
}
