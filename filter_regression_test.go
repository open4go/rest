package rest

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func hasEntry(d bson.D, key string, want interface{}) bool {
	for _, e := range d {
		if e.Key == key && e.Value == want {
			return true
		}
	}
	return false
}

// 回归：过滤器里只要出现数组值，其余（身份/租户）条件不能被丢掉。
func TestAsMongoFilterArrayNeverDropsIdentityConditions(t *testing.T) {
	cases := []struct {
		name   string
		filter string
	}{
		{"one array", `{"account":"A","status":["1","2"]}`},
		{"two arrays", `{"account":"A","status":["1","2"],"source":["x","y"]}`},
		{"array on identity key", `{"account":"A","name":["n1","n2"]}`},
	}
	fields := []string{"account=>customer.account", "status", "source", "name"}
	for _, tc := range cases {
		c := newQueryCtx(t, "tenant-a", map[string]string{"filter": tc.filter})
		q := LoadQuery(c)
		f, _ := q.AsMongoFilter(fields, q.Filter)
		d := filterOf(t, f)
		assertTenant(t, d, "tenant-a")
		if !hasEntry(d, "customer.account", "A") {
			t.Fatalf("%s: scalar identity condition dropped: %v", tc.name, d)
		}
	}
}

// 数组值与同名身份键同时出现时，结果仍然是 bson.D，且租户条件只有一个。
func TestAsMongoFilterArrayOnMappedKeyStillScoped(t *testing.T) {
	c := newQueryCtx(t, "tenant-a", map[string]string{"filter": `{"account":["A","B"]}`})
	q := LoadQuery(c)
	f, _ := q.AsMongoFilter([]string{"account=>customer.account"}, q.Filter)
	d := filterOf(t, f)
	assertTenant(t, d, "tenant-a")
	if len(merchantEntries(d)) != 1 {
		t.Fatalf("tenant condition duplicated: %v", d)
	}
}
