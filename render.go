package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/open4go/middle"
	"github.com/sirupsen/logrus"
	"net/http"
)

// MakeResponse 返回数据
func MakeResponse(c *gin.Context, err error, msg string, result interface{}) {
	// Retrieve the Logrus logger from the context
	logger, _ := c.Get("log")
	log := logger.(*logrus.Entry)
	logWithContext := log.WithField("result", result)
	l := middle.LoadFromHeader(c)
	id := c.Param("_id")
	if err != nil {
		logWithContext.WithField("id", id).
			WithField("message", msg).
			WithField("account_id", l.AccountId).Error(err)
		// 操作失败
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": msg,
			"error":   err.Error(),
			"status":  "failed",
		})
		return
	} else {
		switch v := result.(type) {
		case string:
			// 创建
			c.JSON(http.StatusCreated, gin.H{
				"message": msg,
				"id":      v,
				"status":  "success",
			})
			c.Header("id", v)
			return
		default:
			_id := c.Param("_id")

			// 查询详情
			if result != nil {
				// 返回详情
				c.IndentedJSON(http.StatusOK, v)
			} else {
				// 删除/更新/操作成功
				c.JSON(http.StatusAccepted, gin.H{
					"message": msg,
					"id":      _id,
					"status":  "success",
				})
			}
			c.Header("id", _id)
			return
		}
	}
}
