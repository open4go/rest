package rest

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// IsolateMerchantID resolves the merchant used for list isolation.
// Middleware writes the resolved tenant into X-Tenant-ID:
//   - missing / "*" → empty, meaning all tenants (super-admin 所有)
//   - other         → that merchant
//
// Deprecated fallback: when X-Tenant-ID is empty this still reads client
// X-Merchant-ID for gateway compatibility. New code MUST use
// IsolateMerchantIDStrict (pilots and anything security-sensitive). The
// X-Merchant-ID fallback will be removed once all callers migrate.
func IsolateMerchantID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	tenant := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	if tenant == "*" {
		return ""
	}
	if tenant != "" {
		return tenant
	}
	// Legacy unsafe fallback — prefer IsolateMerchantIDStrict.
	switcher := strings.TrimSpace(c.GetHeader("X-Merchant-ID"))
	if switcher == "*" {
		return ""
	}
	return switcher
}

// IsolateMerchantIDStrict resolves the merchant for list isolation using ONLY
// the JWT-written X-Tenant-ID header. Client-supplied X-Merchant-ID is ignored.
// Empty / "*" means view-all (super-admin). Prefer this over IsolateMerchantID.
func IsolateMerchantIDStrict(c *gin.Context) string {
	if c == nil {
		return ""
	}
	tenant := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	if tenant == "*" {
		return ""
	}
	return tenant
}
