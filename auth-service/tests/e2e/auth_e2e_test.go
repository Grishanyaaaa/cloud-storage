package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// baseURL returns the API Gateway URL used for e2e testing.
// Override with E2E_BASE_URL environment variable.
func baseURL(t *testing.T) string {
	t.Helper()
	if u := os.Getenv("E2E_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://api.cloud-storage.local"
}

// httpClient returns an HTTP client with a reasonable timeout for e2e tests.
func httpClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

// apiResponse is the generic envelope returned by auth-service.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type registerData struct {
	UserID string `json:"UserID"`
}

type tokenPairData struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
}

// doJSON is a helper that sends a JSON request and returns the parsed response.
func doJSON(t *testing.T, method, url string, body interface{}) (*http.Response, apiResponse) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("execute request %s %s: %v", method, url, err)
	}

	var ar apiResponse
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &ar); err != nil {
			// Not JSON — store raw body in error field for debugging.
			ar.Error = string(respBody)
		}
	}
	return resp, ar
}

// uniqueEmail generates a unique email for each test to avoid conflicts.
func uniqueEmail() string {
	return fmt.Sprintf("e2e_%d@test.com", time.Now().UnixNano())
}

const validPassword = "TestPass1!"

// registerAndLogin is a helper that registers a user and logs in,
// returning the token pair and the email used.
func registerAndLogin(t *testing.T) (email string, tokens tokenPairData) {
	t.Helper()
	base := baseURL(t)
	email = uniqueEmail()

	// Register
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}

	// Login
	resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}

	if err := json.Unmarshal(ar.Data, &tokens); err != nil {
		t.Fatalf("unmarshal token pair: %v", err)
	}
	return email, tokens
}

// ---------------------------------------------------------------------------
// REGISTER
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()

	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
	if ar.Status != "success" {
		t.Fatalf("expected status=success, got %q", ar.Status)
	}

	var data registerData
	if err := json.Unmarshal(ar.Data, &data); err != nil {
		t.Fatalf("unmarshal register data: %v", err)
	}
	if data.UserID == "" {
		t.Fatal("expected non-empty UserID")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()

	// First registration
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first register: expected 201, got %d", resp.StatusCode)
	}

	// Duplicate registration
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register: expected 409, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "already exists") {
		t.Fatalf("expected 'already exists' error, got: %s", ar.Error)
	}
}

func TestRegister_EmptyEmail(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    "",
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "email is required") {
		t.Fatalf("expected 'email is required', got: %s", ar.Error)
	}
}

func TestRegister_InvalidEmailFormat(t *testing.T) {
	base := baseURL(t)
	invalidEmails := []string{
		"not-an-email",
		"@missing-local.com",
		"missing-at-sign.com",
		"spaces in@email.com",
		"double@@at.com",
	}
	for _, email := range invalidEmails {
		t.Run(email, func(t *testing.T) {
			resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
				"email":    email,
				"password": validPassword,
			})
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("email=%q: expected 400, got %d: %s", email, resp.StatusCode, ar.Error)
			}
		})
	}
}

func TestRegister_EmailTooLong(t *testing.T) {
	base := baseURL(t)
	longLocal := strings.Repeat("a", 250)
	email := longLocal + "@test.com"

	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestRegister_EmptyPassword(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "password is required") {
		t.Fatalf("expected 'password is required', got: %s", ar.Error)
	}
}

func TestRegister_PasswordTooLong(t *testing.T) {
	base := baseURL(t)
	longPassword := strings.Repeat("A", 73)

	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": longPassword,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "72 bytes") {
		t.Fatalf("expected '72 bytes' in error, got: %s", ar.Error)
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "Ab1!",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "at least") {
		t.Fatalf("expected 'at least' in error, got: %s", ar.Error)
	}
}

func TestRegister_PasswordNoUppercase(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "testpass1!",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "uppercase") {
		t.Fatalf("expected 'uppercase' in error, got: %s", ar.Error)
	}
}

func TestRegister_PasswordNoLowercase(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "TESTPASS1!",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "lowercase") {
		t.Fatalf("expected 'lowercase' in error, got: %s", ar.Error)
	}
}

func TestRegister_PasswordNoNumber(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "TestPass!!",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "number") {
		t.Fatalf("expected 'number' in error, got: %s", ar.Error)
	}
}

func TestRegister_PasswordNoSpecialChar(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "TestPass11",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "special") {
		t.Fatalf("expected 'special' in error, got: %s", ar.Error)
	}
}

func TestRegister_PasswordMultipleViolations(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "weak",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	// Should contain multiple violation descriptions
	if !strings.Contains(ar.Error, "password validation failed") {
		t.Fatalf("expected 'password validation failed', got: %s", ar.Error)
	}
}

func TestRegister_EmptyBody(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/register", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/register", strings.NewReader("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRegister_EmailNormalization(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()
	upperEmail := strings.ToUpper(email)

	// Register with uppercase email
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    upperEmail,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register uppercase: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}

	// Login with lowercase — should work because email is normalized
	resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login lowercase: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestRegister_EmailWithWhitespace(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()

	// Register with leading/trailing spaces
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    "  " + email + "  ",
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register with spaces: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}

	// Login with trimmed email
	resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login trimmed: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
}

// ---------------------------------------------------------------------------
// LOGIN
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()

	// Register first
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}

	// Login
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	if ar.Status != "success" {
		t.Fatalf("expected status=success, got %q", ar.Status)
	}

	var tokens tokenPairData
	if err := json.Unmarshal(ar.Data, &tokens); err != nil {
		t.Fatalf("unmarshal tokens: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Fatal("expected non-empty AccessToken")
	}
	if tokens.RefreshToken == "" {
		t.Fatal("expected non-empty RefreshToken")
	}
	if len(tokens.RefreshToken) != 64 {
		t.Fatalf("expected 64-char RefreshToken, got %d chars", len(tokens.RefreshToken))
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()

	// Register
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}

	// Login with wrong password
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": "WrongPass1!",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "invalid") {
		t.Fatalf("expected 'invalid' in error, got: %s", ar.Error)
	}
}

func TestLogin_NonExistentUser(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    "nonexistent_" + uniqueEmail(),
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "invalid") {
		t.Fatalf("expected 'invalid' error, got: %s", ar.Error)
	}
}

func TestLogin_EmptyEmail(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    "",
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestLogin_EmptyPassword(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    uniqueEmail(),
		"password": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestLogin_EmptyBody(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/login", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/login", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogin_MultipleLogins(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()

	// Register
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}

	// Login multiple times — each should produce different refresh tokens
	var refreshTokens []string
	for i := 0; i < 3; i++ {
		resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
			"email":    email,
			"password": validPassword,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("login %d: expected 200, got %d", i, resp.StatusCode)
		}
		var tokens tokenPairData
		if err := json.Unmarshal(ar.Data, &tokens); err != nil {
			t.Fatalf("unmarshal tokens %d: %v", i, err)
		}
		refreshTokens = append(refreshTokens, tokens.RefreshToken)
	}

	// All refresh tokens should be different
	for i := 0; i < len(refreshTokens); i++ {
		for j := i + 1; j < len(refreshTokens); j++ {
			if refreshTokens[i] == refreshTokens[j] {
				t.Fatalf("refresh tokens %d and %d are identical", i, j)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// REFRESH
// ---------------------------------------------------------------------------

func TestRefresh_Success(t *testing.T) {
	base := baseURL(t)
	_, tokens := registerAndLogin(t)

	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	if ar.Status != "success" {
		t.Fatalf("expected status=success, got %q", ar.Status)
	}

	var newTokens tokenPairData
	if err := json.Unmarshal(ar.Data, &newTokens); err != nil {
		t.Fatalf("unmarshal tokens: %v", err)
	}
	if newTokens.AccessToken == "" {
		t.Fatal("expected non-empty new AccessToken")
	}
	if newTokens.RefreshToken == "" {
		t.Fatal("expected non-empty new RefreshToken")
	}
	// New refresh token should differ from the old one (token rotation)
	if newTokens.RefreshToken == tokens.RefreshToken {
		t.Fatal("expected new RefreshToken to differ from old one")
	}
}

func TestRefresh_OldTokenInvalidated(t *testing.T) {
	base := baseURL(t)
	_, tokens := registerAndLogin(t)

	// First refresh — should succeed
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first refresh: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}

	// Second refresh with the SAME old token — should fail (token was rotated/revoked)
	resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reuse old token: expected 401, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestRefresh_ChainRotation(t *testing.T) {
	base := baseURL(t)
	_, tokens := registerAndLogin(t)

	currentRefresh := tokens.RefreshToken
	for i := 0; i < 3; i++ {
		resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
			"refresh_token": currentRefresh,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("refresh %d: expected 200, got %d: %s", i, resp.StatusCode, ar.Error)
		}
		var newTokens tokenPairData
		if err := json.Unmarshal(ar.Data, &newTokens); err != nil {
			t.Fatalf("unmarshal %d: %v", i, err)
		}
		if newTokens.RefreshToken == currentRefresh {
			t.Fatalf("refresh %d: token was not rotated", i)
		}
		currentRefresh = newTokens.RefreshToken
	}
}

func TestRefresh_EmptyToken(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "refresh_token is required") {
		t.Fatalf("expected 'refresh_token is required', got: %s", ar.Error)
	}
}

func TestRefresh_InvalidFormat_TooShort(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": "tooshort",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "invalid refresh_token format") {
		t.Fatalf("expected 'invalid refresh_token format', got: %s", ar.Error)
	}
}

func TestRefresh_InvalidFormat_NonHex(t *testing.T) {
	base := baseURL(t)
	// 64 chars but not valid hex
	invalidToken := strings.Repeat("g", 64)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": invalidToken,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "invalid refresh_token format") {
		t.Fatalf("expected 'invalid refresh_token format', got: %s", ar.Error)
	}
}

func TestRefresh_NonExistentToken(t *testing.T) {
	base := baseURL(t)
	// Valid hex format but doesn't exist in the database
	fakeToken := strings.Repeat("ab", 32) // 64 hex chars
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": fakeToken,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestRefresh_EmptyBody(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/refresh", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRefresh_InvalidJSON(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/refresh", strings.NewReader("{broken"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// LOGOUT
// ---------------------------------------------------------------------------

func TestLogout_Success(t *testing.T) {
	base := baseURL(t)
	_, tokens := registerAndLogin(t)

	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestLogout_TokenInvalidatedAfterLogout(t *testing.T) {
	base := baseURL(t)
	_, tokens := registerAndLogin(t)

	// Logout
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d", resp.StatusCode)
	}

	// Try to refresh with the revoked token — should fail
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: expected 401, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestLogout_Idempotent(t *testing.T) {
	base := baseURL(t)
	_, tokens := registerAndLogin(t)

	// First logout
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("first logout: expected 204, got %d", resp.StatusCode)
	}

	// Second logout with the same token — should still succeed (idempotent)
	resp, _ = doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("second logout: expected 204, got %d", resp.StatusCode)
	}
}

func TestLogout_NonExistentToken(t *testing.T) {
	base := baseURL(t)
	fakeToken := strings.Repeat("cd", 32)
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": fakeToken,
	})
	// Logout is idempotent — non-existent token returns 204
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestLogout_EmptyToken(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestLogout_InvalidFormat(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": "invalid-format",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
	if !strings.Contains(ar.Error, "invalid refresh_token format") {
		t.Fatalf("expected 'invalid refresh_token format', got: %s", ar.Error)
	}
}

func TestLogout_EmptyBody(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/logout", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLogout_InvalidJSON(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/logout", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// JWKS
// ---------------------------------------------------------------------------

func TestJWKS_Success(t *testing.T) {
	base := baseURL(t)
	resp, err := httpClient().Get(base + "/.well-known/jwks.json")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var jwks struct {
		Keys []struct {
			Kty string `json:"kty"`
			Use string `json:"use"`
			Alg string `json:"alg"`
			Crv string `json:"crv"`
			Kid string `json:"kid"`
			X   string `json:"x"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		t.Fatalf("unmarshal JWKS: %v", err)
	}

	if len(jwks.Keys) == 0 {
		t.Fatal("expected at least one key in JWKS")
	}

	key := jwks.Keys[0]
	if key.Kty != "OKP" {
		t.Fatalf("expected kty=OKP, got %q", key.Kty)
	}
	if key.Alg != "EdDSA" {
		t.Fatalf("expected alg=EdDSA, got %q", key.Alg)
	}
	if key.Crv != "Ed25519" {
		t.Fatalf("expected crv=Ed25519, got %q", key.Crv)
	}
	if key.Use != "sig" {
		t.Fatalf("expected use=sig, got %q", key.Use)
	}
	if key.Kid == "" {
		t.Fatal("expected non-empty kid")
	}
	if key.X == "" {
		t.Fatal("expected non-empty x (public key)")
	}
}

func TestJWKS_CacheControlHeader(t *testing.T) {
	base := baseURL(t)
	resp, err := httpClient().Get(base + "/.well-known/jwks.json")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if cc == "" {
		t.Fatal("expected Cache-Control header")
	}
	if !strings.Contains(cc, "max-age") {
		t.Fatalf("expected Cache-Control to contain max-age, got: %s", cc)
	}
}

func TestJWKS_MethodNotAllowed(t *testing.T) {
	base := baseURL(t)
	req, _ := http.NewRequest(http.MethodPost, base+"/.well-known/jwks.json", nil)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	// chi returns 405 for wrong method on registered routes
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// FULL FLOW
// ---------------------------------------------------------------------------

func TestFullFlow_RegisterLoginRefreshLogout(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()

	// 1. Register
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("step 1 register: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}

	// 2. Login
	resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 2 login: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var tokens tokenPairData
	if err := json.Unmarshal(ar.Data, &tokens); err != nil {
		t.Fatalf("unmarshal tokens: %v", err)
	}

	// 3. Refresh
	resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 3 refresh: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var newTokens tokenPairData
	if err := json.Unmarshal(ar.Data, &newTokens); err != nil {
		t.Fatalf("unmarshal new tokens: %v", err)
	}

	// 4. Verify old refresh token is invalid
	resp, _ = doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("step 4 old token: expected 401, got %d", resp.StatusCode)
	}

	// 5. Logout with new token
	resp, _ = doJSON(t, http.MethodPost, base+"/auth/v1/logout", map[string]string{
		"refresh_token": newTokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("step 5 logout: expected 204, got %d", resp.StatusCode)
	}

	// 6. Verify refresh after logout fails
	resp, _ = doJSON(t, http.MethodPost, base+"/auth/v1/refresh", map[string]string{
		"refresh_token": newTokens.RefreshToken,
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("step 6 refresh after logout: expected 401, got %d", resp.StatusCode)
	}

	// 7. Login again — should still work
	resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 7 re-login: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
}

// ---------------------------------------------------------------------------
// EDGE CASES / HTTP METHOD
// ---------------------------------------------------------------------------

func TestRegister_WrongHTTPMethod(t *testing.T) {
	base := baseURL(t)
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, base+"/auth/v1/register", nil)
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("method=%s: expected 405, got %d", method, resp.StatusCode)
			}
		})
	}
}

func TestLogin_WrongHTTPMethod(t *testing.T) {
	base := baseURL(t)
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, base+"/auth/v1/login", nil)
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("method=%s: expected 405, got %d", method, resp.StatusCode)
			}
		})
	}
}

func TestRefresh_WrongHTTPMethod(t *testing.T) {
	base := baseURL(t)
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, base+"/auth/v1/refresh", nil)
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("method=%s: expected 405, got %d", method, resp.StatusCode)
			}
		})
	}
}

func TestLogout_WrongHTTPMethod(t *testing.T) {
	base := baseURL(t)
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, base+"/auth/v1/logout", nil)
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("method=%s: expected 405, got %d", method, resp.StatusCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SPECIAL CHARACTERS & UNICODE
// ---------------------------------------------------------------------------

func TestRegister_PasswordWithUnicode(t *testing.T) {
	base := baseURL(t)
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "Тест1234!Пароль",
	})
	// Should succeed — unicode characters count as uppercase/lowercase letters
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestLogin_PasswordWithUnicode(t *testing.T) {
	base := baseURL(t)
	email := uniqueEmail()
	password := "Тест1234!Пароль"

	// Register
	resp, _ := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    email,
		"password": password,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d", resp.StatusCode)
	}

	// Login
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestRegister_PasswordWithSpecialChars(t *testing.T) {
	base := baseURL(t)
	specialPasswords := []string{
		"Test1234!@#$%",
		"Test1234&*()_+",
		`Test1234<>{}[]`,
		"Test1234|\\~`",
	}
	for _, pw := range specialPasswords {
		t.Run(pw, func(t *testing.T) {
			email := uniqueEmail()
			resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
				"email":    email,
				"password": pw,
			})
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("password=%q: expected 201, got %d: %s", pw, resp.StatusCode, ar.Error)
			}

			// Verify login works
			resp, ar = doJSON(t, http.MethodPost, base+"/auth/v1/login", map[string]string{
				"email":    email,
				"password": pw,
			})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("password=%q login: expected 200, got %d: %s", pw, resp.StatusCode, ar.Error)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CONTENT-TYPE EDGE CASES
// ---------------------------------------------------------------------------

func TestRegister_NoContentType(t *testing.T) {
	base := baseURL(t)
	body := `{"email":"` + uniqueEmail() + `","password":"` + validPassword + `"}`
	req, _ := http.NewRequest(http.MethodPost, base+"/auth/v1/register", strings.NewReader(body))
	// Intentionally not setting Content-Type
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	// Go's json.NewDecoder doesn't check Content-Type, so this should still work
	// or return 400 — either is acceptable behavior.
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 201 or 400, got %d", resp.StatusCode)
	}
}

func TestRegister_ExtraFields(t *testing.T) {
	base := baseURL(t)
	body := map[string]string{
		"email":      uniqueEmail(),
		"password":   validPassword,
		"extra_field": "should be ignored",
	}
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", body)
	// Extra fields should be ignored (json.Decoder default behavior)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestRegister_PasswordExactlyMinLength(t *testing.T) {
	base := baseURL(t)
	// Min password length is 8
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": "Aa1!aaaa", // exactly 8 chars
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestRegister_PasswordExactly72Bytes(t *testing.T) {
	base := baseURL(t)
	// Exactly 72 bytes — should be accepted by the DTO validator
	pw := strings.Repeat("A", 35) + strings.Repeat("a", 33) + "1!"
	if len(pw) != 72 {
		// Adjust: 35 + 33 + 2 = 70, need 72
		pw = strings.Repeat("A", 36) + strings.Repeat("a", 33) + "1!x"
	}
	// Build a valid 72-byte password
	pw = "Aa1!" + strings.Repeat("b", 68) // 4 + 68 = 72
	resp, ar := doJSON(t, http.MethodPost, base+"/auth/v1/register", map[string]string{
		"email":    uniqueEmail(),
		"password": pw,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for 72-byte password, got %d: %s", resp.StatusCode, ar.Error)
	}
}

// ---------------------------------------------------------------------------
// NOT FOUND / UNKNOWN ROUTES
// ---------------------------------------------------------------------------

func TestUnknownRoute(t *testing.T) {
	base := baseURL(t)
	resp, err := httpClient().Get(base + "/auth/v1/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 404 or 405, got %d", resp.StatusCode)
	}
}
