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
// X-Merchant-ID is the fallback when the gateway skipped JWT header rewrite.
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
	switcher := strings.TrimSpace(c.GetHeader("X-Merchant-ID"))
	if switcher == "*" {
		return ""
	}
	return switcher
}
