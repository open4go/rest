package rest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/open4go/middle"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func newQueryCtx(t *testing.T, tenant string, params map[string]string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/items?"+v.Encode(), nil)
	if tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	c.Request = req
	return c
}

func filterOf(t *testing.T, f interface{}) bson.D {
	t.Helper()
	d, ok := f.(bson.D)
	if !ok {
		t.Fatalf("AsMongoFilter must return bson.D, got %T", f)
	}
	return d
}

func merchantEntries(d bson.D) []bson.E {
	out := []bson.E{}
	for _, e := range d {
		if e.Key == MerchantFilterKey {
			out = append(out, e)
		}
	}
	return out
}

func assertTenant(t *testing.T, d bson.D, tenant string) {
	t.Helper()
	m := merchantEntries(d)
	if len(m) != 1 {
		t.Fatalf("want exactly one merchant filter, got %d in %v", len(m), d)
	}
	in, ok := m[0].Value.(bson.M)["$in"].([]string)
	if !ok || len(in) != 2 || in[0] != tenant || in[1] != "share" {
		t.Fatalf("merchant filter = %v, want $in [%s share]", m[0].Value, tenant)
	}
}

func TestAsMongoFilterArrayKeepsTenantAndOtherFilters(t *testing.T) {
	ids := []string{primitive.NewObjectID().Hex(), primitive.NewObjectID().Hex()}
	c := newQueryCtx(t, "tenant-a", map[string]string{
		"filter": `{"id":["` + ids[0] + `","` + ids[1] + `"],"status":"on"}`,
	})
	q := LoadQuery(c)
	f, _ := q.AsMongoFilter([]string{"id=>_id", "status"}, q.Filter)
	d := filterOf(t, f)
	assertTenant(t, d, "tenant-a")

	var hasIn, hasStatus bool
	for _, e := range d {
		if e.Key == "_id" {
			hasIn = true
		}
		if e.Key == "status" && e.Value == "on" {
			hasStatus = true
		}
	}
	if !hasIn || !hasStatus {
		t.Fatalf("array and scalar filters must both be kept: %v", d)
	}

	// string arrays too
	c = newQueryCtx(t, "tenant-a", map[string]string{"filter": `{"name":["a","b"]}`})
	q = LoadQuery(c)
	f, _ = q.AsMongoFilter([]string{"name"}, q.Filter)
	assertTenant(t, filterOf(t, f), "tenant-a")
}

func TestAsMongoFilterTenantNotOverridable(t *testing.T) {
	c := newQueryCtx(t, "tenant-a", map[string]string{
		"filter": `{"_.meta.merchant_id":"tenant-b"}`,
	})
	q := LoadQuery(c)
	f, _ := q.AsMongoFilter([]string{"_.meta.merchant_id"}, q.Filter)
	assertTenant(t, filterOf(t, f), "tenant-a")

	c = newQueryCtx(t, "tenant-a", map[string]string{
		"filter": `{"_.meta.merchant_id":["tenant-b","tenant-c"]}`,
	})
	q = LoadQuery(c)
	f, _ = q.AsMongoFilter([]string{"_.meta.merchant_id"}, q.Filter)
	assertTenant(t, filterOf(t, f), "tenant-a")
}

func TestAsMongoFilterNoTenantNoMerchantFilter(t *testing.T) {
	c := newQueryCtx(t, "", map[string]string{"filter": `{"status":"on"}`})
	q := LoadQuery(c)
	f, _ := q.AsMongoFilter([]string{"status"}, q.Filter)
	if len(merchantEntries(filterOf(t, f))) != 0 {
		t.Fatal("no tenant header => no merchant filter (unchanged behaviour)")
	}
}

func sortOf(t *testing.T, params map[string]string) interface{} {
	t.Helper()
	q := LoadQuery(newQueryCtx(t, "", params))
	_, opts := q.AsMongoFilter(nil, q.Filter)
	d := opts.Sort.(bson.D)
	return d[0].Value
}

func TestSortDirection(t *testing.T) {
	if v := sortOf(t, map[string]string{"sort": `["created_at","ASC"]`}); v != 1 {
		t.Fatalf("ASC => %v, want 1", v)
	}
	if v := sortOf(t, map[string]string{"sort": `["created_at","asc"]`}); v != 1 {
		t.Fatalf("asc => %v, want 1", v)
	}
	if v := sortOf(t, map[string]string{"sort": `["created_at","DESC"]`}); v != -1 {
		t.Fatalf("DESC => %v, want -1", v)
	}
	// 未传 sort：保持修复前的实际默认（id 倒序）
	if v := sortOf(t, nil); v != -1 {
		t.Fatalf("default => %v, want -1", v)
	}
}

func resetKey(t *testing.T) {
	t.Helper()
	dataKeyMu.Lock()
	prev := dataEncryptKey
	dataEncryptKey = nil
	dataKeyMu.Unlock()
	prevViper := viper.Get("encryptKey.data")
	viper.Set("encryptKey.data", "")
	t.Cleanup(func() {
		dataKeyMu.Lock()
		dataEncryptKey = prev
		dataKeyMu.Unlock()
		viper.Set("encryptKey.data", prevViper)
	})
}

func TestEncryptRequiresConfiguredKey(t *testing.T) {
	resetKey(t)
	t.Setenv(EnvDataEncryptKey, "")
	if _, err := EncryptData("13800138000"); !errors.Is(err, ErrDataEncryptKeyNotSet) {
		t.Fatalf("encrypt without key: err=%v", err)
	}
	if _, err := DecryptData("AAAA"); !errors.Is(err, ErrDataEncryptKeyNotSet) {
		t.Fatalf("decrypt without key: err=%v", err)
	}
}

func TestLegacyDefaultKeyRejected(t *testing.T) {
	resetKey(t)
	legacy := "0000000000000000legacy-test-key!"
	sum := sha256.Sum256([]byte(legacy))
	prev := legacyDefaultKeySHA256
	legacyDefaultKeySHA256 = hex.EncodeToString(sum[:])
	t.Cleanup(func() { legacyDefaultKeySHA256 = prev })
	if err := SetDataEncryptKey(legacy); err == nil {
		t.Fatal("former built-in key must be rejected")
	}
	t.Setenv(EnvDataEncryptKey, legacy)
	if _, err := EncryptData("13800138000"); err == nil {
		t.Fatal("former built-in key via env must be rejected")
	}
}

func TestEncryptKeyFromEnvAndRotation(t *testing.T) {
	resetKey(t)
	t.Setenv(EnvDataEncryptKey, "0123456789abcdef0123456789abcdef")
	enc, err := EncryptData("13800138000")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := DecryptData(enc)
	if err != nil || dec != "13800138000" {
		t.Fatalf("roundtrip: %q %v", dec, err)
	}

	// 迁移：旧密钥解密 -> 新密钥加密
	newKey := []byte("fedcba9876543210")
	plain, err := DecryptDataWithKey(enc, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	re, err := EncryptDataWithKey(plain, newKey)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := DecryptDataWithKey(re, newKey); err != nil || out != "13800138000" {
		t.Fatalf("rotated: %q %v", out, err)
	}
	if err := SetDataEncryptKey("short"); err == nil {
		t.Fatal("bad length must be rejected")
	}
}

func loginRecorder(t *testing.T) (*httptest.ResponseRecorder, []byte) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	key := []byte("test-key")
	RenderLogin(c, middle.LoginInfo{UserID: "u1", AccountID: "a1"}, hash, "pw", key, "example.com", nil, 0)
	return w, key
}

func TestRenderLoginCookieFlags(t *testing.T) {
	t.Setenv(EnvJWTCookieSecure, "")
	w, key := loginRecorder(t)
	sc := w.Header().Get("Set-Cookie")
	for _, want := range []string{"jwt=", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(sc, want) {
			t.Fatalf("Set-Cookie %q missing %q", sc, want)
		}
	}
	var tok string
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "jwt" {
			tok = ck.Value
		}
	}
	claims := &jwt.RegisteredClaims{}
	if _, err := jwt.ParseWithClaims(tok, claims, func(*jwt.Token) (interface{}, error) { return key, nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired()); err != nil {
		t.Fatalf("token invalid: %v", err)
	}
	if claims.Issuer == "" {
		t.Fatal("issuer should carry login info")
	}

	t.Setenv(EnvJWTCookieSecure, "false")
	w, _ = loginRecorder(t)
	sc = w.Header().Get("Set-Cookie")
	if strings.Contains(sc, "Secure") || !strings.Contains(sc, "HttpOnly") {
		t.Fatalf("dev override: %q", sc)
	}
}
