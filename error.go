package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/open4go/log"
)

// ErrResp 错误数据结构返回
type ErrResp struct {
	Title     string `json:"title"`      // 错误标题
	Message   string `json:"message"`    // 错误详情
	Status    string `json:"status"`     // 错误等级
	RequestID string `json:"request_id"` // 请求ID
	Redirect  string `json:"redirect"`   // 指导前端进行跳转的页面
}

// MakeError 创建错误响应
func MakeError(c *gin.Context, title string, message string, code int, redirect ...string) {
	// 确定 redirect 的默认值
	var redirectURL string
	if len(redirect) > 0 {
		redirectURL = redirect[0]
	} else {
		redirectURL = ""
	}

	// 从请求上下文中获取 Trace ID
	requestID, exists := c.Get("RequestID")
	if !exists {
		// 如果不存在，使用一个默认值
		requestID = "Unknown"
	}

	// 根据前端显示不同的级别
	status := "none"
	if code >= 500 {
		status = "error"
	} else if code < 500 && code >= 400 {
		status = "fail"
	} else if code < 400 && code >= 300 {
		status = "loading"
	} else {
		status = "success"
	}

	// 返回 JSON 错误响应
	c.JSON(code, ErrResp{
		Title:     title,
		Message:   message,
		Status:    status,
		RequestID: requestID.(string),
		Redirect:  redirectURL,
	})

	// 写日志
	log.Log(c.Request.Context()).WithField("title", title).
		WithField("message", message).WithField("status", status).
		Error("call MakeError")
}
