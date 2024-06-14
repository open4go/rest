package rest

import (
	"context"
	"github.com/open4go/log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// Status Define custom type for response status
type Status string

// Define constants for response statuses
const (
	Success Status = "success"
	Failed  Status = "failed"
	// Add more statuses as needed
)

// MakeResponse returns the response.
func MakeResponse(c *gin.Context, err error, msg string, result interface{}) {
	t := time.Now().Unix()
	// 从请求上下文中获取 Trace ID
	RequestID, exists := c.Get("RequestID")
	if !exists {
		// 如果不存在，使用一个默认值
		RequestID = "UNKNOWN"
	}
	// 读取环境变量
	version := os.Getenv("SERVER_VERSION")
	if version == "" {
		version = "-"
	}

	if err != nil {
		// Operation failed
		c.JSON(http.StatusInternalServerError, gin.H{
			"message":    msg,
			"error":      err.Error(),
			"status":     Failed,
			"timestamp":  t,
			"request_id": RequestID,
			"version":    version,
		})
		log.Log(c.Request.Context()).WithField("content", msg).
			Error(err)
		return
	}

	switch v := result.(type) {
	case string:
		// Creation
		c.JSON(http.StatusCreated, gin.H{
			"message": msg,
			"id":      v,
			"status":  Success,
		})
		c.Writer.Header().Set("TargetId", v)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "TargetId", v))
	default:
		_id := c.Param("_id")

		if result != nil {
			// 如果是请求数据详情则直接返回传入的数据结构本身
			// Return details
			c.IndentedJSON(http.StatusOK, v)
		} else {
			// Deletion/Update/Operation success
			c.JSON(http.StatusAccepted, gin.H{
				"message": msg,
				"id":      _id,
				"status":  Success,
			})
		}
		c.Writer.Header().Set("TargetId", _id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "TargetId", _id))
	}
}
