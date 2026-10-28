package rest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestClassifyError(t *testing.T) {
	code, msg := classifyError(mongo.ErrNoDocuments)
	if code != http.StatusNotFound || msg != "not found" {
		t.Fatalf("not found: %d %s", code, msg)
	}
	code, msg = classifyError(errors.New("Key: 'CreateRequest.Name' Error:Field validation for 'Name' failed"))
	if code != http.StatusBadRequest || msg != "bad request" {
		t.Fatalf("validation: %d %s", code, msg)
	}
	code, msg = classifyError(errors.New("dial tcp: lookup store"))
	if code != http.StatusInternalServerError || msg != "internal error" {
		t.Fatalf("internal: %d %s", code, msg)
	}
}

func TestMakeResponseHidesRawError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("RequestID", "rid-1")
	MakeResponse(c, mongo.ErrNoDocuments, "get detail", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d", w.Code)
	}
	body := w.Body.String()
	if contains(body, "no documents") {
		t.Fatalf("raw mongo error leaked: %s", body)
	}
	if !contains(body, "not found") {
		t.Fatalf("expected client-safe error: %s", body)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}
