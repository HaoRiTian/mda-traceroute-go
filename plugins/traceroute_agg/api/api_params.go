package api

type TraceParams struct {
	Dst string `json:"dst" form:"dst"`
	// group为all，意为全地域
	Group string `json:"group" form:"group"`
	// group为某地域时，node-num表示选择该地域的多少个节点
	// group为all时，node-num >= 0 无意义，node-num < 0，意为每个地域选择 |node-num| 个节点
	NodeNum int32 `json:"node-num" form:"node-num"`
}

type NodeWsParams struct {
	Group string `json:"group"`
}
