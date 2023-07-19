package rest

import (
	b64 "encoding/base64"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultExpireLoginSessionTime 默认超时时间
	DefaultExpireLoginSessionTime = 24 // 默认 登陆有效时长24h
)

var (
	// ExpireLoginSessionTime 登陆时间
	ExpireLoginSessionTime = DefaultExpireLoginSessionTime
	// CustomLoginExpireTime 自定义超时时间
	CustomLoginExpireTime = os.Getenv("CUSTOM_LOGIN_EXPIRE_TIME")
)

func init() {
	if CustomLoginExpireTime != "" {
		ExpireLoginSessionTime, _ = strconv.Atoi(CustomLoginExpireTime)
	}
}

// RenderLogin 返回登陆信息
func RenderLogin(c *gin.Context, loginPayload string, passwordFromDB []byte, passwordFromReq string,
	jwtKey []byte, host string, roles []string, toolbar int) {

	logCtx := log.WithField("accountID", loginPayload)
	// 检查密码hash是否相同
	if err := bcrypt.CompareHashAndPassword(passwordFromDB, []byte(passwordFromReq)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "the password or account isn't correct",
			"payload": loginPayload,
			"status":  "error",
			"title":   "An error occurred.",
		})
		logCtx.Error(err)
		return
	}

	// 声明密码签名
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    loginPayload,
		ExpiresAt: time.Now().Add(time.Hour * time.Duration(ExpireLoginSessionTime)).Unix(), //1 day
	})

	sEnc := b64.StdEncoding.EncodeToString(jwtKey)
	sDec, err := b64.StdEncoding.DecodeString(sEnc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "the sign isn't correct",
			"payload": loginPayload,
			"status":  "error",
			"title":   "An error occurred.",
		})
		logCtx.Error(err)
		return
	}
	// 签名密钥
	token, err := claims.SignedString(sDec)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "the encode to base64 isn't correct",
			"payload": loginPayload,
			"status":  "error",
			"title":   "An error occurred.",
			"error":   err.Error(),
		})
		logCtx.Error(err)
		return
	}

	// 设置缓存过期时间
	c.SetCookie("jwt", token, 3600*ExpireLoginSessionTime, "/", host, false, false)

	c.JSON(http.StatusOK, gin.H{
		"message":  "sign in success",
		"payload":  loginPayload,
		"roles":    roles,
		"tool_bar": toolbar,
		"status":   "success",
		"title":    "Sign In.",
	})
}
