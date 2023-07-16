package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/open4go/middle"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func RenderForCRUD(c *gin.Context, err error, msg string, result interface{}) {
	l := middle.LoadFromHeader(c)
	id := c.Param("_id")
	if err != nil {
		// 操作失败
		c.JSON(http.StatusInternalServerError, gin.H{
			"message":    msg,
			"account_id": l.AccountId,
			"id":         c.Param("_id"),
			"error":      err.Error(),
			"status":     "failed",
		})
		log.WithField("id", id).WithField("message", msg).
			WithField("account_id", l.AccountId).Error(err)
		return
	} else {
		switch v := result.(type) {
		case string:
			// 创建
			c.JSON(http.StatusCreated, gin.H{
				"message":    msg,
				"account_id": l.AccountId,
				"id":         result,
				"status":     "success",
			})
			return
		default:
			if result != nil {
				// 返回详情
				c.IndentedJSON(http.StatusOK, v)
			} else {
				// 删除/更新/操作成功
				c.JSON(http.StatusAccepted, gin.H{
					"message":    msg,
					"account_id": l.AccountId,
					"id":         c.Param("_id"),
					"status":     "success",
				})
			}
			return
		}
	}
}
