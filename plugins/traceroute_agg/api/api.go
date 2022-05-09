package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"mda-traceroute-go/db/dao"
	"mda-traceroute-go/plugins/traceroute_agg"
	v1 "mda-traceroute-go/plugins/traceroute_agg/api/v1"
	"mda-traceroute-go/plugins/traceroute_agg/ws"
	"mda-traceroute-go/util"
	"strings"
	"time"
)

func Start() {
	// 初始化服务端 websocket 连接池
	initWsManager()

	router := gin.Default()

	apiGroup(router)
	staticGroup(router)
	wsGroup(router)

	router.Run("0.0.0.0:" + "20118")
}

func initWsManager() {
	go ws.WebsocketManager.Start()
	go ws.WebsocketManager.SendService()
	go ws.WebsocketManager.SendGroupService()
	go ws.WebsocketManager.SendAllService()
}

func apiGroup(router *gin.Engine) {
	apiGroup := router.Group("/api")
	apiGroup.POST("/tracert", recvDst)
	apiGroup.GET("/nodes", getNodes)
}

func staticGroup(router *gin.Engine) {
	//router.LoadHTMLFiles("./static/view/tracert.html")
	router.Static("/static", "./static")
	router.Static("/js", "./static/js")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")
	router.StaticFile("/", "./static/view/tracert.html")

	//router.GET("/", func(context *gin.Context) {
	//	//context.HTML(http.StatusOK, "tracert.html", nil)
	//	context.HTML(http.StatusOK, "tracert.html", nil)
	//})
	//router.GET("/tracert", func(context *gin.Context) {
	//	context.HTML(http.StatusOK, "tracert.html", nil)
	//})
}

func wsGroup(router *gin.Engine) {
	wsGroup := router.Group("/ws")
	// 子节点与服务端建立websocket连接，子节点建立连接需要注明自己的 group
	wsGroup.GET("/:group", ws.WebsocketManager.WsClient)
}

func recvDst(c *gin.Context) {
	var res v1.HttpResponse

	var params TraceParams

	var err error
	err = c.BindJSON(&params)
	if err != nil {
		c.JSON(500, res.Fail("参数错误，json化出错:", err.Error()))
		logrus.Errorf("参数错误，json化出错: %v. ", err.Error())
		return
	}

	// 正则匹配检查格式是否正确
	if util.MatchDst(params.Dst) == -1 {
		c.JSON(500, res.Fail("探测目的地址不符合规则:", params.Dst))
		logrus.Errorf("探测目的地址不符合规则: %s", params.Dst)
		return
	}

	err = verifyParams(&params)
	if err != nil {
		c.JSON(500, res.Fail(err))
		logrus.Errorf("%v", err)
		return
	}

	// 新建一个TracertAgg，并运行
	agg, err := traceroute_agg.NewTracerouteAgg(params.Dst, params.Group, params.NodeNum, time.Now(), &ws.WebsocketManager)
	if err != nil {
		c.JSON(500, res.Fail(err))
		logrus.Errorf("%v", err)
		return
	}
	logrus.Infof("New TracertAgg sucess, next start tracert.")
	agg.Start()
	<-agg.Complete

	//testFillResult(agg, 40)

	// 不需要对Result进行json编码
	c.JSON(200, res.Success(agg.Result))
	return
}

func getNodes(c *gin.Context) {
	var res v1.HttpResponse
	var nodes []string
	ws.WebsocketManager.NodeInfo(&nodes)
	logrus.Infof("GetNodes() Nodes: %v", nodes)
	c.JSON(200, res.Success(nodes))
	return
}

// 用于验证请求的参数
func verifyParams(params *TraceParams) error {
	if strings.Contains(params.Group, "INVALID") {
		return fmt.Errorf("error! group is invalid")
	}
	return nil
}

func testFillResult(agg *traceroute_agg.TracerouteAgg, cnt int) {
	for i := 1; i <= cnt; i++ {
		topo := &dao.Topo{
			Domain:      "-",
			TTL:         uint8(i),
			DstIP:       "127.0.0.1",
			ResAddr:     "127.0.0.1",
			Name:        "ri.Name",
			Session:     "ri.Session",
			MeanLatency: 23,
			RecvCnt:     1,
			Country:     "China",
			Region:      "Hubei",
			City:        "Wuhan",
			ISP:         "电信",
			TracertTime: time.Now(),
		}
		agg.Lock.Lock()
		agg.Result[uint8(i)] = append(agg.Result[topo.TTL], topo)
		agg.Lock.Unlock()
	}
}
