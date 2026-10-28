package rest

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/open4go/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// Status Define custom type for response status
type Status string

// Define constants for response statuses
const (
	Success Status = "success"
	Failed  Status = "failed"
	// Add more statuses as needed
)

// classifyError maps common library errors to an HTTP status and a client-safe
// error string. Raw database / validator text stays in the server log only.
func classifyError(err error) (code int, clientErr string) {
	if err == nil {
		return http.StatusOK, ""
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return http.StatusNotFound, "not found"
	}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		return http.StatusBadRequest, "validation failed"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no documents in result"),
		strings.Contains(msg, "not found"):
		return http.StatusNotFound, "not found"
	case strings.Contains(msg, "field validation"),
		strings.Contains(msg, "cannot unmarshal"),
		strings.Contains(msg, "invalid character"),
		strings.Contains(msg, "unexpected end of json"),
		strings.Contains(msg, "eof"):
		return http.StatusBadRequest, "bad request"
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

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
		code, clientErr := classifyError(err)
		c.JSON(code, gin.H{
			"message":    msg,
			"error":      clientErr,
			"status":     Failed,
			"timestamp":  t,
			"request_id": RequestID,
			"version":    version,
		})
		log.Log(c.Request.Context()).WithField("content", msg).
			WithField("http_status", code).
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
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "TargetId", v)) //nolint:staticcheck // SA1029: plain-string key kept for existing readers
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
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), "TargetId", _id)) //nolint:staticcheck // SA1029: plain-string key kept for existing readers
	}
}
