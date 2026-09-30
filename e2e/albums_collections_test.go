package e2e_test

import (
	"net/http"
	"testing"
)

func TestCollectionNotFoundReturns404(t *testing.T) {
	env := requireShared(t)
	const missing = "/api/admin/collections/999999"

	cases := []struct {
		name, method, path string
		payload            any
	}{
		{"update", http.MethodPut, missing, map[string]any{"name": "x"}},
		{"delete", http.MethodDelete, missing, nil},
		{"inherit", http.MethodPut, missing + "/banners/inherit", map[string]any{"inherit": true}},
		{"filters", http.MethodPut, missing + "/filters", map[string]any{"filters": []any{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doJSON(t, tc.method, tc.path, env.adminToken, tc.payload)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404, got %d %s", resp.StatusCode, resp.Body)
			}
			assertErrorShape(t, resp)
		})
	}
}
