package v1

import (
	"github.com/gin-gonic/gin"
	"time"
)

type HttpResponse struct {
	Code    int
	Message interface{}
	Data    interface{}
}

func (res HttpResponse) Success(data interface{}) gin.H {
	return gin.H{
		"code":     200,
		"message":  "success",
		"data":     data,
		"req_time": time.Now().Unix(),
	}
}

func (res HttpResponse) Fail(msg ...interface{}) gin.H {
	return gin.H{
		"code":     500,
		"message":  msg,
		"data":     nil,
		"req_time": time.Now().Unix(),
	}
}
