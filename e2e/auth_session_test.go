package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

var fakeIPCounter struct {
	sync.Mutex
	n int
}

// nextFakeIP returns a unique public client IP so tests do not share the per-IP
// login/register rate limit buckets (the test server's loopback peer is a
// trusted proxy, so X-Forwarded-For is honored).
func nextFakeIP() string {
	fakeIPCounter.Lock()
	defer fakeIPCounter.Unlock()
	fakeIPCounter.n++
	return fmt.Sprintf("198.18.%d.%d", fakeIPCounter.n/250, fakeIPCounter.n%250+1)
}

func doJSONFromIP(t *testing.T, ip, method, path, token string, payload any) apiResponse {
	t.Helper()
	return doJSONWithHeaders(t, map[string]string{"X-Forwarded-For": ip}, method, path, token, payload)
}

func doJSONWithHeaders(t *testing.T, headers map[string]string, method, path, token string, payload any) apiResponse {
	t.Helper()
	env := requireShared(t)
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(method, env.server.URL+path, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return apiResponse{StatusCode: resp.StatusCode, Header: resp.Header, Body: body}
}

func loginFresh(t *testing.T, username, password string) (apiResponse, string) {
	t.Helper()
	resp := doJSONFromIP(t, nextFakeIP(), http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": username,
		"password": password,
	})
	var parsed struct {
		Token string `json:"token"`
	}
	if resp.StatusCode == http.StatusOK {
		resp.decodeData(t, &parsed)
	}
	return resp, parsed.Token
}

func TestPasswordChangeRevokesTokens(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()

	t.Run("self service change", func(t *testing.T) {
		username := "selfpw_" + suffix
		createUser(t, env.adminToken, username, "original-password")
		resp, oldToken := loginFresh(t, username, "original-password")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login failed: %d %s", resp.StatusCode, resp.Body)
		}
		if r := doRequest(t, http.MethodGet, "/api/auth/me", oldToken, nil, ""); r.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 before change, got %d", r.StatusCode)
		}

		resp = doJSON(t, http.MethodPut, "/api/auth/me", oldToken, map[string]string{
			"current_password": "original-password",
			"new_password":     "brand-new-password",
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("password change failed: %d %s", resp.StatusCode, resp.Body)
		}

		r := doRequest(t, http.MethodGet, "/api/auth/me", oldToken, nil, "")
		if r.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected old token to be revoked (401), got %d", r.StatusCode)
		}
		assertErrorShape(t, r)

		resp, newToken := loginFresh(t, username, "brand-new-password")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login with new password failed: %d %s", resp.StatusCode, resp.Body)
		}
		if r := doRequest(t, http.MethodGet, "/api/auth/me", newToken, nil, ""); r.StatusCode != http.StatusOK {
			t.Fatalf("expected new token to work, got %d", r.StatusCode)
		}
	})

	t.Run("admin reset", func(t *testing.T) {
		username := "adminpw_" + suffix
		id := createUser(t, env.adminToken, username, "original-password")
		_, oldToken := loginFresh(t, username, "original-password")

		resp := doJSON(t, http.MethodPut, fmt.Sprintf("/api/admin/users/%d", id), env.adminToken, map[string]string{
			"password": "admin-reset-password",
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("admin reset failed: %d %s", resp.StatusCode, resp.Body)
		}
		if r := doRequest(t, http.MethodGet, "/api/auth/me", oldToken, nil, ""); r.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected old token to be revoked (401), got %d", r.StatusCode)
		}
		// the admin's own token must be unaffected
		if r := doRequest(t, http.MethodGet, "/api/auth/me", env.adminToken, nil, ""); r.StatusCode != http.StatusOK {
			t.Fatalf("admin token should still work, got %d", r.StatusCode)
		}
	})

	t.Run("garbage tokens do not leak parser errors", func(t *testing.T) {
		r := doRequest(t, http.MethodGet, "/api/auth/me", "not.a.jwt", nil, "")
		if r.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", r.StatusCode)
		}
		if strings.Contains(strings.ToLower(string(r.Body)), "malformed") || strings.Contains(string(r.Body), "token is") {
			t.Fatalf("response leaks parser error text: %s", r.Body)
		}
	})
}

func createInviteCode(t *testing.T, payload map[string]any) (uint, string) {
	t.Helper()
	resp := doJSON(t, http.MethodPost, "/api/admin/invite-codes", requireShared(t).adminToken, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create invite code: %d %s", resp.StatusCode, resp.Body)
	}
	var code struct {
		ID   uint   `json:"id"`
		Code string `json:"code"`
	}
	resp.decodeData(t, &code)
	return code.ID, code.Code
}

func TestRegisterInviteCodeAtomic(t *testing.T) {
	requireShared(t)
	suffix := randomSuffix()
	_, code := createInviteCode(t, map[string]any{"max_uses": 1})

	const attempts = 8
	statuses := make([]int, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp := doJSONFromIP(t, nextFakeIP(), http.MethodPost, "/api/auth/register", "", map[string]string{
				"username":    fmt.Sprintf("racer_%s_%d", suffix, i),
				"password":    "racer-password",
				"first_name":  "Race",
				"last_name":   "Er",
				"invite_code": code,
			})
			statuses[i] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	created := 0
	for _, s := range statuses {
		switch s {
		case http.StatusCreated:
			created++
		case http.StatusForbidden:
		default:
			t.Fatalf("unexpected status %d in %v", s, statuses)
		}
	}
	if created != 1 {
		t.Fatalf("expected exactly one registration with a max_uses=1 code, got %d (%v)", created, statuses)
	}
}

func TestRegisterAndUserInputValidation(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	_, code := createInviteCode(t, map[string]any{"max_uses": 10})

	register := func(username, password string) apiResponse {
		return doJSONFromIP(t, nextFakeIP(), http.MethodPost, "/api/auth/register", "", map[string]string{
			"username":    username,
			"password":    password,
			"first_name":  "A",
			"last_name":   "B",
			"invite_code": code,
		})
	}

	name := "dup_" + suffix
	if resp := register(name, "long-enough-pw"); resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, resp.Body)
	}
	resp := register(name, "long-enough-pw")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate username, got %d: %s", resp.StatusCode, resp.Body)
	}
	assertErrorShape(t, resp)
	if strings.Contains(string(resp.Body), "SQLSTATE") || strings.Contains(string(resp.Body), "duplicate key") {
		t.Fatalf("raw database error leaked: %s", resp.Body)
	}

	if resp := register("short_"+suffix, "short"); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password, got %d: %s", resp.StatusCode, resp.Body)
	}
	if resp := register("long_"+suffix, strings.Repeat("a", 73)); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for >72 byte password, got %d: %s", resp.StatusCode, resp.Body)
	}

	// admin create: duplicate, short password
	resp = doJSON(t, http.MethodPost, "/api/admin/users", env.adminToken, map[string]any{
		"username": name, "password": "long-enough-pw", "first_name": "A", "last_name": "B",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 creating duplicate user, got %d: %s", resp.StatusCode, resp.Body)
	}
	resp = doJSON(t, http.MethodPost, "/api/admin/users", env.adminToken, map[string]any{
		"username": "shortpw_" + suffix, "password": "short", "first_name": "A", "last_name": "B",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for short admin-set password, got %d: %s", resp.StatusCode, resp.Body)
	}

	// admin update: duplicate username, empty username, album-scoped key as global
	other := createUser(t, env.adminToken, "other_"+suffix, "long-enough-pw")
	path := fmt.Sprintf("/api/admin/users/%d", other)
	if resp := doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"username": name}); resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 renaming to duplicate, got %d: %s", resp.StatusCode, resp.Body)
	}
	if resp := doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"username": "  "}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty username, got %d: %s", resp.StatusCode, resp.Body)
	}
	if resp := doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"global_permissions": []string{"album.photo.upload"}}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for album-scoped key as global, got %d: %s", resp.StatusCode, resp.Body)
	}
	if resp := doJSON(t, http.MethodPut, path, env.adminToken, map[string]any{"password": "short"}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for short password on update, got %d: %s", resp.StatusCode, resp.Body)
	}

	// self-service profile: duplicate username
	_, tok := loginFresh(t, "other_"+suffix, "long-enough-pw")
	if resp := doJSON(t, http.MethodPut, "/api/auth/me", tok, map[string]any{"username": name}); resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for profile rename to duplicate, got %d: %s", resp.StatusCode, resp.Body)
	}
}

func TestLoginAndRegisterRateLimited(t *testing.T) {
	requireShared(t)

	ip := nextFakeIP()
	got429 := false
	for i := 0; i < 30; i++ {
		resp := doJSONFromIP(t, ip, http.MethodPost, "/api/auth/login", "", map[string]string{
			"username": "nobody", "password": "wrong-password",
		})
		if resp.StatusCode == http.StatusTooManyRequests {
			assertErrorShape(t, resp)
			got429 = true
			break
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 or 429, got %d: %s", resp.StatusCode, resp.Body)
		}
	}
	if !got429 {
		t.Fatal("expected login to be rate limited")
	}

	// a different client is unaffected
	if resp := doJSONFromIP(t, nextFakeIP(), http.MethodPost, "/api/auth/login", "", map[string]string{
		"username": "nobody", "password": "wrong-password",
	}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a different client, got %d", resp.StatusCode)
	}

	ip = nextFakeIP()
	got429 = false
	for i := 0; i < 30; i++ {
		resp := doJSONFromIP(t, ip, http.MethodPost, "/api/auth/register", "", map[string]string{})
		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("expected register to be rate limited")
	}
}

// TestLoginRateLimitClientIP checks that the rate limit bucket follows the
// X-Forwarded-For hop appended by the trusted proxy, not client-controlled headers.
func TestLoginRateLimitClientIP(t *testing.T) {
	requireShared(t)
	login := func(headers map[string]string) int {
		return doJSONWithHeaders(t, headers, http.MethodPost, "/api/auth/login", "", map[string]string{
			"username": "nobody", "password": "wrong-password",
		}).StatusCode
	}

	// a fresh True-Client-IP / X-Real-IP per attempt does not escape the bucket
	ip := nextFakeIP()
	got429 := false
	for i := 0; i < 30; i++ {
		spoof := nextFakeIP()
		status := login(map[string]string{"X-Forwarded-For": ip, "True-Client-IP": spoof, "X-Real-IP": spoof})
		if status == http.StatusTooManyRequests {
			got429 = true
			break
		}
		if status != http.StatusUnauthorized {
			t.Fatalf("expected 401 or 429, got %d", status)
		}
	}
	if !got429 {
		t.Fatal("expected login to be rate limited despite spoofed client IP headers")
	}

	// entries the client prepends to X-Forwarded-For are ignored
	if status := login(map[string]string{"X-Forwarded-For": nextFakeIP() + ", " + ip}); status != http.StatusTooManyRequests {
		t.Fatalf("expected 429 with a spoofed X-Forwarded-For prefix, got %d", status)
	}

	// behind an outer proxy (whose hop is trusted) the bucket still follows the client
	if status := login(map[string]string{"X-Forwarded-For": ip + ", 172.18.0.1"}); status != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for the same client behind an outer proxy, got %d", status)
	}
	if status := login(map[string]string{"X-Forwarded-For": nextFakeIP() + ", 172.18.0.1"}); status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for another client behind the outer proxy, got %d", status)
	}
}

func TestInviteCodeUpdateSemantics(t *testing.T) {
	env := requireShared(t)
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	id, _ := createInviteCode(t, map[string]any{"max_uses": 3, "expires_at": future})
	path := fmt.Sprintf("/api/admin/invite-codes/%d", id)

	type inviteDTO struct {
		ExpiresAt *string `json:"expires_at"`
		MaxUses   *int    `json:"max_uses"`
		IsActive  bool    `json:"is_active"`
	}
	update := func(payload map[string]any) (apiResponse, inviteDTO) {
		resp := doJSON(t, http.MethodPut, path, env.adminToken, payload)
		var dto inviteDTO
		if resp.StatusCode == http.StatusOK {
			resp.decodeData(t, &dto)
		}
		return resp, dto
	}

	// omitted fields stay untouched
	resp, dto := update(map[string]any{"is_active": false})
	if resp.StatusCode != http.StatusOK || dto.IsActive || dto.MaxUses == nil || *dto.MaxUses != 3 || dto.ExpiresAt == nil {
		t.Fatalf("unexpected result: %d %s", resp.StatusCode, resp.Body)
	}

	// validation
	if resp, _ := update(map[string]any{"max_uses": 0}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for max_uses 0, got %d", resp.StatusCode)
	}
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	if resp, _ := update(map[string]any{"expires_at": past}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for past expiry, got %d", resp.StatusCode)
	}

	// explicit null clears
	resp, dto = update(map[string]any{"max_uses": nil, "expires_at": nil})
	if resp.StatusCode != http.StatusOK || dto.MaxUses != nil || dto.ExpiresAt != nil {
		t.Fatalf("expected max_uses and expires_at cleared: %d %s", resp.StatusCode, resp.Body)
	}
	if dto.IsActive {
		t.Fatalf("is_active should be unchanged (false): %s", resp.Body)
	}

	// create-time validation
	if resp := doJSON(t, http.MethodPost, "/api/admin/invite-codes", env.adminToken, map[string]any{"max_uses": 0}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 creating with max_uses 0, got %d", resp.StatusCode)
	}
}

func TestAlbumUsersReturnMinimalDTO(t *testing.T) {
	env := requireShared(t)
	suffix := randomSuffix()
	album := createAlbum(t, env.adminToken, "Minimal DTO "+suffix, "minimal-dto-"+suffix, "")
	userID := createUser(t, env.adminToken, "member_"+suffix, "long-enough-pw")
	grantAlbumPermission(t, env.adminToken, album.ID, userID, []string{"album.photo.upload"})

	// adding again is a conflict
	resp := doJSON(t, http.MethodPost, fmt.Sprintf("/api/admin/albums/%d/users", album.ID), env.adminToken, map[string]any{
		"user_id": userID, "permissions": []string{"album.photo.upload"},
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, resp.Body)
	}

	assertMinimalUser := func(raw json.RawMessage) {
		t.Helper()
		var user map[string]json.RawMessage
		if err := json.Unmarshal(raw, &user); err != nil {
			t.Fatalf("decode user: %v", err)
		}
		for _, banned := range []string{"roles", "global_permissions", "effective_permissions", "album_permissions_map"} {
			if _, ok := user[banned]; ok {
				t.Fatalf("user DTO leaks %q: %s", banned, raw)
			}
		}
		if _, ok := user["id"]; !ok {
			t.Fatalf("user DTO missing id: %s", raw)
		}
		if _, ok := user["username"]; !ok {
			t.Fatalf("user DTO missing username: %s", raw)
		}
	}

	resp = doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d/users", album.ID), env.adminToken, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
	}
	var members []struct {
		User json.RawMessage `json:"user"`
	}
	resp.decodeData(t, &members)
	if len(members) == 0 {
		t.Fatal("expected at least one album member")
	}
	for _, m := range members {
		assertMinimalUser(m.User)
	}

	other := createUser(t, env.adminToken, "available_"+suffix, "long-enough-pw")
	_ = other
	resp = doRequest(t, http.MethodGet, fmt.Sprintf("/api/admin/albums/%d/users/available", album.ID), env.adminToken, nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Body)
	}
	var available []json.RawMessage
	resp.decodeData(t, &available)
	if len(available) == 0 {
		t.Fatal("expected available users")
	}
	for _, u := range available {
		assertMinimalUser(u)
	}
}

func TestPaginationPastEndAndOverflow(t *testing.T) {
	env := requireShared(t)
	for _, page := range []string{"1000", "9223372036854775807", "4611686018427387904"} {
		resp := doRequest(t, http.MethodGet, "/api/admin/users?per_page=100&page="+page, env.adminToken, nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("page=%s: expected 200, got %d: %s", page, resp.StatusCode, resp.Body)
		}
		var body struct {
			Data []json.RawMessage `json:"data"`
			Meta struct {
				Pagination struct {
					Count       int `json:"count"`
					CurrentPage int `json:"current_page"`
				} `json:"pagination"`
			} `json:"meta"`
		}
		resp.decode(t, &body)
		if len(body.Data) != 0 || body.Meta.Pagination.Count != 0 {
			t.Fatalf("page=%s: expected empty page, got %s", page, resp.Body)
		}
	}
}

func TestCORSPreflightOnPeopleAndFaceRoutes(t *testing.T) {
	env := requireShared(t)
	for _, path := range []string{"/api/people/", "/api/people/1", "/api/faces/1/tag", "/api/images/faces/"} {
		req, err := http.NewRequest(http.MethodOptions, env.server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", "http://example.test")
		req.Header.Set("Access-Control-Request-Method", "PUT")
		req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://example.test" {
			t.Fatalf("%s: expected ACAO on preflight, got %q (status %d)", path, got, resp.StatusCode)
		}
	}
}

func TestLogoutRevokesToken(t *testing.T) {
	env := requireShared(t)
	username := fmt.Sprintf("logout_%d", time.Now().UnixNano())
	createUser(t, env.adminToken, username, "original-password")
	_, token := loginFresh(t, username, "original-password")

	if r := doRequest(t, http.MethodGet, "/api/auth/me", token, nil, ""); r.StatusCode != http.StatusOK {
		t.Fatalf("expected fresh token to work, got %d", r.StatusCode)
	}
	if r := doRequest(t, http.MethodPost, "/api/auth/logout", token, nil, ""); r.StatusCode != http.StatusOK {
		t.Fatalf("logout failed: %d %s", r.StatusCode, r.Body)
	}
	if r := doRequest(t, http.MethodGet, "/api/auth/me", token, nil, ""); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected token to be revoked after logout (401), got %d", r.StatusCode)
	}

	// logging out again, or without a usable token, still succeeds
	for _, tok := range []string{token, "", "not.a.jwt"} {
		if r := doRequest(t, http.MethodPost, "/api/auth/logout", tok, nil, ""); r.StatusCode != http.StatusOK {
			t.Fatalf("logout with token %q: expected 200, got %d", tok, r.StatusCode)
		}
	}
}

func TestOversizedJSONBodyRejected(t *testing.T) {
	env := requireShared(t)
	big := `{"username":"` + strings.Repeat("a", 128<<10) + `","password":"x"}`
	cases := []struct{ path, token string }{
		{"/api/auth/login", ""},
		{"/api/auth/register", ""},
		{"/api/admin/users", env.adminToken},
	}
	for _, c := range cases {
		r := doRequest(t, http.MethodPost, c.path, c.token, strings.NewReader(big), "application/json")
		if r.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("%s: expected 413, got %d %s", c.path, r.StatusCode, r.Body)
		}
	}
}
