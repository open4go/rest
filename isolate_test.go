package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func isolateCtx(headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/v1/hlj/info", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func TestIsolateMerchantIDViewAll(t *testing.T) {
	c := isolateCtx(map[string]string{"X-Merchant-ID": "*"})
	if got := IsolateMerchantID(c); got != "" {
		t.Fatalf("view-all IsolateMerchantID=%q want empty", got)
	}
	c = isolateCtx(map[string]string{"X-Tenant-ID": "*"})
	if got := IsolateMerchantID(c); got != "" {
		t.Fatalf("tenant=* IsolateMerchantID=%q want empty", got)
	}
}

func TestIsolateMerchantIDSwitcherWhenTenantMissing(t *testing.T) {
	c := isolateCtx(map[string]string{"X-Merchant-ID": "tenant-b"})
	if got := IsolateMerchantID(c); got != "tenant-b" {
		t.Fatalf("IsolateMerchantID=%q want tenant-b", got)
	}
}

func TestIsolateMerchantIDTenantHeaderWins(t *testing.T) {
	c := isolateCtx(map[string]string{
		"X-Merchant-ID": "tenant-b",
		"X-Tenant-ID":   "st",
	})
	if got := IsolateMerchantID(c); got != "st" {
		t.Fatalf("IsolateMerchantID=%q want st from X-Tenant-ID", got)
	}
}

func TestIsolateMerchantIDStrictIgnoresSwitcher(t *testing.T) {
	c := isolateCtx(map[string]string{"X-Merchant-ID": "tenant-b"})
	if got := IsolateMerchantIDStrict(c); got != "" {
		t.Fatalf("Strict without tenant=%q want empty (no X-Merchant-ID fallback)", got)
	}
	c = isolateCtx(map[string]string{
		"X-Merchant-ID": "tenant-b",
		"X-Tenant-ID":   "st",
	})
	if got := IsolateMerchantIDStrict(c); got != "st" {
		t.Fatalf("Strict IsolateMerchantIDStrict=%q want st", got)
	}
	c = isolateCtx(map[string]string{"X-Tenant-ID": "*"})
	if got := IsolateMerchantIDStrict(c); got != "" {
		t.Fatalf("Strict tenant=* =%q want empty", got)
	}
}

func TestIsolateMerchantIDLegacyStillFallsBack(t *testing.T) {
	c := isolateCtx(map[string]string{"X-Merchant-ID": "legacy-m"})
	if got := IsolateMerchantID(c); got != "legacy-m" {
		t.Fatalf("legacy IsolateMerchantID=%q want legacy-m", got)
	}
}
