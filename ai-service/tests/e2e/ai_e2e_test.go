// Package e2e contains end-to-end tests for ai-service.
//
// These tests drive the full microservice stack through the API Gateway:
//
//	auth-service    → JWT
//	storage-service → owner's file tree (needed so the AI has something to
//	                  reason about)
//	ai-service      → /ai/v1/commands  (this service)
//	Yandex GPT      → real LLM round-trip
//
// They are run with:
//
//	E2E_BASE_URL=http://api.cloud-storage.local go test ./tests/e2e/... -v
//
// Tests that require a real LLM round-trip are gated behind E2E_AI_LIVE=1.
// The "lite" tests (validation, auth, method enforcement, status machine
// edge cases) run unconditionally.
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

// ---------------------------------------------------------------------------
// Shared helpers (mirror the storage-service e2e helpers)
// ---------------------------------------------------------------------------

func baseURL(t *testing.T) string {
	t.Helper()
	if u := os.Getenv("E2E_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://api.cloud-storage.local"
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	Code   string          `json:"code,omitempty"`
}

type tokenPair struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
}

type nodeView struct {
	ID        string     `json:"id"`
	OwnerID   string     `json:"owner_id"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Kind      string     `json:"kind"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Depth     int        `json:"depth"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type operationDTO struct {
	Kind        string  `json:"kind"`
	NodeID      string  `json:"node_id"`
	NewName     string  `json:"new_name,omitempty"`
	NewParentID *string `json:"new_parent_id,omitempty"`
}

type operationResultDTO struct {
	Index        int    `json:"index"`
	Kind         string `json:"kind"`
	NodeID       string `json:"node_id"`
	Success      bool   `json:"success"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type commandView struct {
	ID           string               `json:"id"`
	UserID       string               `json:"user_id"`
	Input        string               `json:"input"`
	Plan         []operationDTO       `json:"plan"`
	Explanation  string               `json:"explanation"`
	Status       string               `json:"status"`
	LLMModel     string               `json:"llm_model,omitempty"`
	LLMTokensIn  int                  `json:"llm_tokens_in,omitempty"`
	LLMTokensOut int                  `json:"llm_tokens_out,omitempty"`
	Results      []operationResultDTO `json:"results,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	ExecutedAt   *time.Time           `json:"executed_at,omitempty"`
	CancelledAt  *time.Time           `json:"cancelled_at,omitempty"`
}

func do(t *testing.T, method, url, token string, body any) (*http.Response, apiResponse) {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, url, err)
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var ar apiResponse
	if len(respBody) > 0 {
		if jerr := json.Unmarshal(respBody, &ar); jerr != nil {
			ar.Error = string(respBody)
		}
	}
	return resp, ar
}

func mustUnmarshal(t *testing.T, ar apiResponse, v any) {
	t.Helper()
	if ar.Status != "success" {
		t.Fatalf("expected status=success, got %q: %s", ar.Status, ar.Error)
	}
	if len(ar.Data) == 0 {
		t.Fatalf("empty data field: %+v", ar)
	}
	if err := json.Unmarshal(ar.Data, v); err != nil {
		t.Fatalf("unmarshal: %v (raw=%s)", err, string(ar.Data))
	}
}

const validPassword = "TestPass1!"

var counter int64

func nextID() int64 { counter++; return counter }

func uniqueEmail() string {
	return fmt.Sprintf("e2e_ai_%d_%d@test.com", time.Now().UnixNano(), nextID())
}

func registerAndLogin(t *testing.T) string {
	t.Helper()
	email := uniqueEmail()

	// register
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/auth/v1/register", "", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
	// login
	resp, ar = do(t, http.MethodPost, baseURL(t)+"/auth/v1/login", "", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var tp tokenPair
	mustUnmarshal(t, ar, &tp)
	if tp.AccessToken == "" {
		t.Fatal("empty AccessToken")
	}
	return tp.AccessToken
}

// initRoot ensures the owner's root folder exists and returns it.
func initRoot(t *testing.T, token string) nodeView {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/me/root", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("initRoot: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	return n
}

// createFolder creates a folder under parentID and returns it.
func createFolder(t *testing.T, token, parentID, name string) nodeView {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]string{
		"parent_id": parentID,
		"name":      name,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("createFolder %q: expected 201, got %d: %s", name, resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	return n
}

// requireLLM gates a test on the E2E_AI_LIVE=1 env-var.
//
// The Plan step is a real Yandex GPT round-trip; running it in a CI that
// doesn't have an LLM key wired up will flake. Lite tests still run.
func requireLLM(t *testing.T) {
	t.Helper()
	if os.Getenv("E2E_AI_LIVE") != "1" {
		t.Skip("E2E_AI_LIVE!=1, skipping test that hits a real LLM")
	}
}

// planCommand POSTs a natural-language input and returns the planned command.
func planCommand(t *testing.T, token, input string) commandView {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands", token, map[string]string{
		"input": input,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("plan: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
	var cmd commandView
	mustUnmarshal(t, ar, &cmd)
	if cmd.Status != "awaiting_confirmation" {
		t.Fatalf("plan: expected status=awaiting_confirmation, got %q", cmd.Status)
	}
	if cmd.ID == "" {
		t.Fatal("plan: empty command ID")
	}
	return cmd
}

func getCommand(t *testing.T, token, id string) (*http.Response, commandView) {
	t.Helper()
	resp, ar := do(t, http.MethodGet, baseURL(t)+"/ai/v1/commands/"+id, token, nil)
	var cmd commandView
	if resp.StatusCode == http.StatusOK {
		mustUnmarshal(t, ar, &cmd)
	}
	return resp, cmd
}

func executeCommand(t *testing.T, token, id string) (*http.Response, commandView) {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/"+id+"/execute", token, nil)
	var cmd commandView
	if resp.StatusCode == http.StatusOK {
		mustUnmarshal(t, ar, &cmd)
	}
	return resp, cmd
}

func cancelCommand(t *testing.T, token, id string) (*http.Response, commandView) {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/"+id+"/cancel", token, nil)
	var cmd commandView
	if resp.StatusCode == http.StatusOK {
		mustUnmarshal(t, ar, &cmd)
	}
	return resp, cmd
}

// ---------------------------------------------------------------------------
// 1.  Healthz / readyz
// ---------------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/healthz", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz: expected 200, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 2.  POST /ai/v1/commands  — validation + auth (no LLM needed)
// ---------------------------------------------------------------------------

func TestPlan_Unauthorized(t *testing.T) {
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands", "", map[string]string{
		"input": "delete reports",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestPlan_InvalidToken(t *testing.T) {
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands", "not.a.jwt", map[string]string{
		"input": "delete reports",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestPlan_EmptyInput(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands", token, map[string]string{
		"input": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPlan_WhitespaceOnlyInput(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands", token, map[string]string{
		"input": "   \t\n  ",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPlan_MissingInputField(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands", token, map[string]string{
		"prompt": "this is the wrong field name",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 (DisallowUnknownFields/missing input), got %d", resp.StatusCode)
	}
}

func TestPlan_MalformedJSON(t *testing.T) {
	token := registerAndLogin(t)
	req, _ := http.NewRequest(http.MethodPost, baseURL(t)+"/ai/v1/commands", strings.NewReader("{not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPlan_InputOverLimit(t *testing.T) {
	token := registerAndLogin(t)
	// Default cap is 2000 chars; send 4000.
	big := strings.Repeat("а", 4000)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands", token, map[string]string{
		"input": big,
	})
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 4xx (oversized input), got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 3.  GET /ai/v1/commands/{id}
// ---------------------------------------------------------------------------

func TestGet_Unauthorized(t *testing.T) {
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestGet_InvalidUUID(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/ai/v1/commands/not-a-uuid", token, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGet_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 4.  POST /ai/v1/commands/{id}/execute  — validation + auth
// ---------------------------------------------------------------------------

func TestExecute_Unauthorized(t *testing.T) {
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000/execute", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestExecute_InvalidUUID(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/not-uuid/execute", token, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestExecute_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000/execute", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 5.  POST /ai/v1/commands/{id}/cancel  — validation + auth
// ---------------------------------------------------------------------------

func TestCancel_Unauthorized(t *testing.T) {
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000/cancel", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestCancel_InvalidUUID(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/not-uuid/cancel", token, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCancel_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000/cancel", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 6.  Method enforcement (no LLM)
// ---------------------------------------------------------------------------

func TestWrongMethod_Commands(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/ai/v1/commands", token, nil)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestWrongMethod_Execute(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000/execute", token, nil)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestWrongMethod_Cancel(t *testing.T) {
	token := registerAndLogin(t)
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/ai/v1/commands/00000000-0000-0000-0000-000000000000/cancel", token, nil)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 7.  Plan happy paths — real LLM round-trips
// ---------------------------------------------------------------------------

func TestPlan_DeleteSuccess(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	docs := createFolder(t, token, root.ID, "Документы")
	report := createFolder(t, token, docs.ID, "отчёт за май")
	_ = report

	cmd := planCommand(t, token, "Удали папку «отчёт за май» из Документов")
	if len(cmd.Plan) == 0 {
		t.Fatalf("expected non-empty plan, got %+v", cmd.Plan)
	}
	hasDelete := false
	for _, op := range cmd.Plan {
		if op.Kind == "delete" {
			hasDelete = true
			break
		}
	}
	if !hasDelete {
		t.Errorf("expected a delete op in plan, got %+v", cmd.Plan)
	}
	if cmd.Explanation == "" {
		t.Error("expected non-empty explanation")
	}
	if cmd.LLMModel == "" {
		t.Error("expected non-empty llm_model")
	}
	if cmd.ExpiresAt.Before(time.Now()) {
		t.Errorf("expires_at is in the past: %v", cmd.ExpiresAt)
	}
}

func TestPlan_RenameSuccess(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "Старая папка")

	cmd := planCommand(t, token, "Переименуй папку «Старая папка» в «Новая папка»")
	if len(cmd.Plan) == 0 {
		t.Fatalf("expected non-empty plan, got %+v", cmd.Plan)
	}
	hasRename := false
	for _, op := range cmd.Plan {
		if op.Kind == "rename" {
			hasRename = true
			if op.NewName == "" {
				t.Errorf("rename op missing new_name: %+v", op)
			}
		}
	}
	if !hasRename {
		t.Errorf("expected a rename op in plan, got %+v", cmd.Plan)
	}
}

func TestPlan_MoveSuccess(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	src := createFolder(t, token, root.ID, "Source")
	dst := createFolder(t, token, root.ID, "Destination")
	createFolder(t, token, src.ID, "TargetMove")
	_ = dst

	cmd := planCommand(t, token, "Перемести папку TargetMove из Source в Destination")
	hasMove := false
	for _, op := range cmd.Plan {
		if op.Kind == "move" {
			hasMove = true
			if op.NewParentID == nil || *op.NewParentID == "" {
				t.Errorf("move op missing new_parent_id: %+v", op)
			}
		}
	}
	if !hasMove {
		t.Errorf("expected a move op in plan, got %+v", cmd.Plan)
	}
}

// ---------------------------------------------------------------------------
// 8.  Lifecycle: Plan → Execute → verify
// ---------------------------------------------------------------------------

func TestExecute_DeleteSuccess(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	target := createFolder(t, token, root.ID, "DeleteMeViaAI")

	cmd := planCommand(t, token, "Удали папку DeleteMeViaAI")

	resp, exec := executeCommand(t, token, cmd.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("execute: expected 200, got %d", resp.StatusCode)
	}
	if exec.Status != "executed" {
		t.Fatalf("expected status=executed, got %q", exec.Status)
	}
	if exec.ExecutedAt == nil {
		t.Error("expected non-nil executed_at")
	}
	if len(exec.Results) == 0 {
		t.Fatal("expected non-empty results")
	}

	// Verify the target is now soft-deleted in storage-service.
	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/nodes/"+target.ID+"?include_deleted=true", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get target after delete: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.DeletedAt == nil {
		t.Fatal("expected target to be soft-deleted")
	}
}

func TestExecute_AlreadyExecuted(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "DoubleExecute")

	cmd := planCommand(t, token, "Удали папку DoubleExecute")
	if r, _ := executeCommand(t, token, cmd.ID); r.StatusCode != http.StatusOK {
		t.Fatalf("first execute: expected 200, got %d", r.StatusCode)
	}
	// Second execute on the same command MUST fail (conflict/state-violation).
	resp, _ := executeCommand(t, token, cmd.ID)
	if resp.StatusCode < 400 {
		t.Fatalf("second execute: expected 4xx, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 9.  Lifecycle: Plan → Cancel → verify no side effects
// ---------------------------------------------------------------------------

func TestCancel_Success(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "CancelMe")

	cmd := planCommand(t, token, "Удали папку CancelMe")
	resp, cancelled := cancelCommand(t, token, cmd.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: expected 200, got %d", resp.StatusCode)
	}
	if cancelled.Status != "cancelled" {
		t.Fatalf("expected status=cancelled, got %q", cancelled.Status)
	}
	if cancelled.CancelledAt == nil {
		t.Error("expected non-nil cancelled_at")
	}
	if cancelled.ExecutedAt != nil {
		t.Errorf("expected nil executed_at, got %v", *cancelled.ExecutedAt)
	}
	if len(cancelled.Results) != 0 {
		t.Errorf("expected empty results on cancelled command, got %+v", cancelled.Results)
	}
}

func TestCancel_VerifyNoSideEffects(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	target := createFolder(t, token, root.ID, "ProtectMe")

	cmd := planCommand(t, token, "Удали папку ProtectMe")
	if r, _ := cancelCommand(t, token, cmd.ID); r.StatusCode != http.StatusOK {
		t.Fatalf("cancel: expected 200, got %d", r.StatusCode)
	}

	// Verify the target was NOT touched in storage-service.
	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/nodes/"+target.ID, token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get target after cancel: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.DeletedAt != nil {
		t.Fatal("target was modified despite cancel")
	}
}

func TestCancel_AfterExecute(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "ExecThenCancel")

	cmd := planCommand(t, token, "Удали папку ExecThenCancel")
	if r, _ := executeCommand(t, token, cmd.ID); r.StatusCode != http.StatusOK {
		t.Fatalf("execute: expected 200, got %d", r.StatusCode)
	}
	// Cancelling an already-executed command MUST fail (state-violation).
	resp, _ := cancelCommand(t, token, cmd.ID)
	if resp.StatusCode < 400 {
		t.Fatalf("cancel after execute: expected 4xx, got %d", resp.StatusCode)
	}
}

func TestExecute_AfterCancel(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "CancelThenExec")

	cmd := planCommand(t, token, "Удали папку CancelThenExec")
	if r, _ := cancelCommand(t, token, cmd.ID); r.StatusCode != http.StatusOK {
		t.Fatalf("cancel: expected 200, got %d", r.StatusCode)
	}
	resp, _ := executeCommand(t, token, cmd.ID)
	if resp.StatusCode < 400 {
		t.Fatalf("execute after cancel: expected 4xx, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 10. GET /ai/v1/commands/{id} after lifecycle transitions
// ---------------------------------------------------------------------------

func TestGet_AfterPlan(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "GetAfterPlan")

	planned := planCommand(t, token, "Удали папку GetAfterPlan")

	resp, fetched := getCommand(t, token, planned.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", resp.StatusCode)
	}
	if fetched.ID != planned.ID {
		t.Fatalf("id mismatch: %q vs %q", fetched.ID, planned.ID)
	}
	if fetched.Status != "awaiting_confirmation" {
		t.Fatalf("expected status=awaiting_confirmation, got %q", fetched.Status)
	}
	if len(fetched.Plan) != len(planned.Plan) {
		t.Errorf("plan size mismatch: %d vs %d", len(fetched.Plan), len(planned.Plan))
	}
}

func TestGet_OtherUserForbidden(t *testing.T) {
	requireLLM(t)
	tokenA := registerAndLogin(t)
	rootA := initRoot(t, tokenA)
	createFolder(t, tokenA, rootA.ID, "PrivateToA")
	cmdA := planCommand(t, tokenA, "Удали папку PrivateToA")

	tokenB := registerAndLogin(t)
	_ = initRoot(t, tokenB)

	resp, _ := getCommand(t, tokenB, cmdA.ID)
	// User B must not see A's command — either 403 or 404 is acceptable.
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403 or 404, got %d", resp.StatusCode)
	}
}

func TestExecute_OtherUserForbidden(t *testing.T) {
	requireLLM(t)
	tokenA := registerAndLogin(t)
	rootA := initRoot(t, tokenA)
	createFolder(t, tokenA, rootA.ID, "ExecPrivateToA")
	cmdA := planCommand(t, tokenA, "Удали папку ExecPrivateToA")

	tokenB := registerAndLogin(t)
	_ = initRoot(t, tokenB)

	resp, _ := executeCommand(t, tokenB, cmdA.ID)
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403 or 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 11. Full flow: Plan → Get → Execute → verify via storage-service
// ---------------------------------------------------------------------------

func TestFullFlow_PlanExecuteVerify(t *testing.T) {
	requireLLM(t)
	token := registerAndLogin(t)
	root := initRoot(t, token)
	src := createFolder(t, token, root.ID, "Documents")
	dst := createFolder(t, token, root.ID, "Archive")
	movable := createFolder(t, token, src.ID, "Quarterly")

	// 1. Plan the move.
	planned := planCommand(t, token, "Перемести папку Quarterly из Documents в Archive")
	if len(planned.Plan) == 0 {
		t.Fatal("plan is empty")
	}

	// 2. Get it back by ID.
	resp, fetched := getCommand(t, token, planned.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", resp.StatusCode)
	}
	if fetched.Status != "awaiting_confirmation" {
		t.Fatalf("expected awaiting_confirmation, got %q", fetched.Status)
	}

	// 3. Execute it.
	resp, executed := executeCommand(t, token, planned.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("execute: expected 200, got %d", resp.StatusCode)
	}
	if executed.Status != "executed" {
		t.Fatalf("expected executed, got %q", executed.Status)
	}

	// 4. Verify the move took effect in storage-service.
	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/nodes/"+movable.ID, token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get after move: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.ParentID == nil || *n.ParentID != dst.ID {
		t.Fatalf("expected parent_id=%q (Archive), got %v", dst.ID, n.ParentID)
	}
}
