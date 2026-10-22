package rest

import (
	b64 "encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/open4go/middle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
		if v, err := strconv.Atoi(strings.TrimSpace(CustomLoginExpireTime)); err == nil && v > 0 {
			ExpireLoginSessionTime = v
		} else {
			log.WithField("CUSTOM_LOGIN_EXPIRE_TIME", CustomLoginExpireTime).
				Warn("invalid CUSTOM_LOGIN_EXPIRE_TIME, using default")
		}
	}
}

// EnvJWTCookieSecure 环境变量，设为 false 时登录 cookie 不带 Secure（仅限本地 http 开发）
const EnvJWTCookieSecure = "JWT_COOKIE_SECURE"

// jwtCookieSecure 登录 cookie 是否带 Secure 标记，默认 true。
// 可通过 viper jwt.cookieSecure 或环境变量 JWT_COOKIE_SECURE 关闭（仅限本地 http 开发）。
func jwtCookieSecure() bool {
	if v := strings.TrimSpace(os.Getenv(EnvJWTCookieSecure)); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	if viper.IsSet("jwt.cookieSecure") {
		return viper.GetBool("jwt.cookieSecure")
	}
	return true
}

// SetJWTCookie 写入登录 cookie：HttpOnly、SameSite=Lax，默认 Secure。
func SetJWTCookie(c *gin.Context, token string, maxAge int, host string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("jwt", token, maxAge, "/", host, jwtCookieSecure(), true)
}

// RenderLogin 返回登陆信息
func RenderLogin(c *gin.Context, lp middle.LoginInfo, passwordFromDB []byte, passwordFromReq string,
	jwtKey []byte, host string, roles []string, toolbar int) {

	logCtx := log.WithField("accountID", lp.AccountID).
		WithField("UserId", lp.UserID).WithField("UserName", lp.UserName)
	// 检查密码hash是否相同
	if err := bcrypt.CompareHashAndPassword(passwordFromDB, []byte(passwordFromReq)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "the password or account isn't correct",
			"payload": lp.UserName,
			"status":  "error",
			"title":   "An error occurred.",
		})
		logCtx.Error(err)
		return
	}

	// 声明密码签名
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    middle.DumpLoginInfo(lp),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(ExpireLoginSessionTime))), //1 day
	})

	sEnc := b64.StdEncoding.EncodeToString(jwtKey)
	sDec, err := b64.StdEncoding.DecodeString(sEnc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "the sign isn't correct",
			"user_id": lp.UserID,
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
			"user_id": lp.UserID,
			"status":  "error",
			"title":   "An error occurred.",
			"error":   err.Error(),
		})
		logCtx.Error(err)
		return
	}

	// 设置缓存过期时间
	SetJWTCookie(c, token, 3600*ExpireLoginSessionTime, host)

	c.JSON(http.StatusOK, gin.H{
		"message":     "sign in success",
		"user_id":     lp.UserID,
		"merchant_id": lp.MerchantID,
		"account_id":  lp.AccountID,
		"phone":       lp.Phone,
		"roles":       roles,
		"tool_bar":    toolbar,
		"status":      "success",
		"title":       "Sign In.",
	})
}
