package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestIDAndAbortError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("RequestID", "req-42")
	if got := RequestID(c); got != "req-42" {
		t.Fatalf("RequestID=%q", got)
	}
	AbortError(c, http.StatusForbidden, "禁止", "forbidden")
	if !c.IsAborted() {
		t.Fatal("expected abort")
	}
	var body ErrResp
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.RequestID != "req-42" || body.Status != "fail" || body.Title != "禁止" {
		t.Fatalf("body=%+v", body)
	}
}
