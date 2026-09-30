package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

// apiResponse is a convenience wrapper around an HTTP response for assertions.
type apiResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// decode unmarshals the response body into v.
func (r apiResponse) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("failed to decode response body %q: %v", r.Body, err)
	}
}

// decodeData unmarshals the "data" field of a handlers.WriteAPIResponse-shaped body
// ({"data": ...}) into v. Every JSON success response uses that envelope.
func (r apiResponse) decodeData(t *testing.T, v any) {
	t.Helper()
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &wrapper); err != nil {
		t.Fatalf("failed to decode envelope from response body %q: %v", r.Body, err)
	}
	if err := json.Unmarshal(wrapper.Data, v); err != nil {
		t.Fatalf("failed to decode data field %q: %v", wrapper.Data, err)
	}
}

// apiErrorBody mirrors handlers.APIErrorResponse.
type apiErrorBody struct {
	Errors []struct {
		Code   string `json:"code"`
		Status string `json:"status"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

// assertErrorShape asserts the response body is the standard {"errors":[{code,status,detail}]}
// shape with at least one error entry.
func assertErrorShape(t *testing.T, resp apiResponse) apiErrorBody {
	t.Helper()
	var body apiErrorBody
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		t.Fatalf("response body is not the standard error shape: %v (body: %s)", err, resp.Body)
	}
	if len(body.Errors) == 0 {
		t.Fatalf("expected at least one error in response body, got: %s", resp.Body)
	}
	for _, e := range body.Errors {
		if e.Code == "" || e.Status == "" || e.Detail == "" {
			t.Fatalf("error entry missing code/status/detail: %+v (body: %s)", e, resp.Body)
		}
	}
	return body
}

// doRequest issues an HTTP request against the shared test server. token may be
// empty for an anonymous request.
func doRequest(t *testing.T, method, path, token string, body io.Reader, contentType string) apiResponse {
	t.Helper()
	env := requireShared(t)

	req, err := http.NewRequest(method, env.server.URL+path, body)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	return apiResponse{StatusCode: resp.StatusCode, Header: resp.Header, Body: data}
}

// doJSON issues a request with a JSON-encoded body.
func doJSON(t *testing.T, method, path, token string, payload any) apiResponse {
	t.Helper()
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		body = bytes.NewReader(data)
	}
	return doRequest(t, method, path, token, body, "application/json")
}

// createInitialAdmin calls POST /api/setup/initial-admin directly (used before the
// shared token exists yet).
func createInitialAdmin(baseURL, username, password string) error {
	data, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(baseURL+"/api/setup/initial-admin", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, b)
	}
	return nil
}

// login calls POST /api/auth/login directly and returns the issued token.
func login(baseURL, username, password string) (string, error) {
	data, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(baseURL+"/api/auth/login", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}
	var parsed struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Data.Token == "" {
		return "", fmt.Errorf("login response had no token: %s", body)
	}
	return parsed.Data.Token, nil
}

// multipartUpload builds a multipart/form-data body uploading a single file under the
// "files" field, matching what handlers.UploadImages expects.
func multipartUpload(t *testing.T, filename string, data []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("files", filename)
	if err != nil {
		t.Fatalf("failed to create multipart field: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("failed to write multipart data: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	return &buf, w.FormDataContentType()
}
