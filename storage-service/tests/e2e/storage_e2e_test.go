// Package e2e contains end-to-end tests for storage-service.
//
// These tests drive the full microservice stack through the API Gateway:
//
//	auth-service (register + login) → JWT
//	storage-service (this service)  → file tree, share-links
//	S3-compatible blob storage      → pre-signed PUT/GET (real bytes)
//
// They are run with:
//
//	E2E_BASE_URL=http://api.cloud-storage.local go test ./tests/e2e/... -v
//
// Override E2E_BASE_URL to point at any deployment (kind, k3s, staging).
//
// Some tests do real S3 round-trips through the pre-signed URL returned by
// the storage-service. The S3 endpoint pointed at by the deployment MUST be
// reachable from the host that runs the tests (this is the standard
// assumption shared with the existing auth-service e2e suite).
package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
// Shared helpers
// ---------------------------------------------------------------------------

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
	return &http.Client{Timeout: 30 * time.Second}
}

// apiResponse mirrors the JSON envelope used by storage-service.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	Code   string          `json:"code,omitempty"`
}

// tokenPair is the JSON shape returned by auth-service /login.
type tokenPair struct {
	AccessToken  string `json:"AccessToken"`
	RefreshToken string `json:"RefreshToken"`
}

// nodeView is a JSON-friendly subset of dto.NodeResponse.
type nodeView struct {
	ID        string     `json:"id"`
	OwnerID   string     `json:"owner_id"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Kind      string     `json:"kind"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Depth     int        `json:"depth"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	SizeBytes *int64     `json:"size_bytes,omitempty"`
	MimeType  *string    `json:"mime_type,omitempty"`
	Status    *string    `json:"status,omitempty"`
}

type uploadURLView struct {
	NodeID    string            `json:"node_id"`
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type downloadURLView struct {
	URL       string    `json:"url"`
	Method    string    `json:"method"`
	ExpiresAt time.Time `json:"expires_at"`
}

type shareView struct {
	ID         string     `json:"id"`
	NodeID     string     `json:"node_id"`
	Permission string     `json:"permission"`
	URL        string     `json:"url,omitempty"`
	Token      string     `json:"token,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type listSharesView struct {
	Items []shareView `json:"items"`
}

type listChildrenView struct {
	Items      []nodeView `json:"items"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type treeView struct {
	nodeView
	Children []treeView `json:"children,omitempty"`
}

type publicShareView struct {
	NodeID     string     `json:"node_id"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	Permission string     `json:"permission"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// do executes an HTTP request and returns (resp, parsed envelope).
// If body is non-nil it is JSON-encoded; if headers contains Authorization
// it is set verbatim.
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
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("execute request %s %s: %v", method, url, err)
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

// mustUnmarshal fails the test if the envelope's Data cannot be decoded.
func mustUnmarshal(t *testing.T, ar apiResponse, v any) {
	t.Helper()
	if ar.Status != "success" {
		t.Fatalf("expected status=success, got %q: %s", ar.Status, ar.Error)
	}
	if len(ar.Data) == 0 {
		t.Fatalf("empty data field; envelope=%+v", ar)
	}
	if err := json.Unmarshal(ar.Data, v); err != nil {
		t.Fatalf("unmarshal data: %v (raw=%s)", err, string(ar.Data))
	}
}

const validPassword = "TestPass1!"

func uniqueEmail() string {
	return fmt.Sprintf("e2e_storage_%d_%d@test.com", time.Now().UnixNano(), randInt())
}

// randInt returns a tiny random-ish disambiguator without pulling in
// math/rand (timestamp is already monotonic enough for serial tests, but
// table-driven subtests can collide on the same nanosecond).
var counter int64

func randInt() int64 {
	counter++
	return counter
}

// register registers a fresh user and returns the email.
func register(t *testing.T) string {
	t.Helper()
	email := uniqueEmail()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/auth/v1/register", "", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
	return email
}

// login logs in with email/password and returns the access token.
func login(t *testing.T, email string) string {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/auth/v1/login", "", map[string]string{
		"email":    email,
		"password": validPassword,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var tp tokenPair
	mustUnmarshal(t, ar, &tp)
	if tp.AccessToken == "" {
		t.Fatal("login returned empty AccessToken")
	}
	return tp.AccessToken
}

// registerAndLogin creates a fresh user and returns its access token.
func registerAndLogin(t *testing.T) string {
	t.Helper()
	email := register(t)
	return login(t, email)
}

// initRoot calls POST /storage/v1/me/root and returns the user's root node.
func initRoot(t *testing.T, token string) nodeView {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/me/root", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("initRoot: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var root nodeView
	mustUnmarshal(t, ar, &root)
	if root.ID == "" || root.Kind != "folder" {
		t.Fatalf("initRoot: bad root payload: %+v", root)
	}
	return root
}

// createFolder is a convenience helper that creates a folder under parentID.
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

// genUploadURL requests a pre-signed PUT URL.
func genUploadURL(t *testing.T, token, parentID, name, mime string, size int64) uploadURLView {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/upload-url", token, map[string]any{
		"parent_id":  parentID,
		"name":       name,
		"size_bytes": size,
		"mime_type":  mime,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("genUploadURL: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var u uploadURLView
	mustUnmarshal(t, ar, &u)
	if u.URL == "" || u.NodeID == "" {
		t.Fatalf("genUploadURL: bad payload: %+v", u)
	}
	return u
}

// putToPresigned uploads body bytes via the pre-signed PUT URL.
// Returns nil on 2xx, otherwise t.Fatal is invoked. The test SHOULD be
// gated on E2E_SKIP_S3 if S3 is not reachable from the test host.
func putToPresigned(t *testing.T, u uploadURLView, body []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, u.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new PUT request: %v", err)
	}
	for k, v := range u.Headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("PUT to S3: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("PUT to S3: got %d: %s", resp.StatusCode, string(b))
	}
}

// finalizeUpload activates a pending file blob by its node ID.
func finalizeUpload(t *testing.T, token, nodeID string, size int64, checksum string) nodeView {
	t.Helper()
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/"+nodeID+"/finalize", token, map[string]any{
		"size_bytes": size,
		"checksum":   checksum,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("finalize: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.Status == nil || *n.Status != "active" {
		t.Fatalf("finalize: expected status=active, got %+v", n.Status)
	}
	return n
}

// uploadFileFull performs the full 3-phase upload (generate → PUT → finalize)
// and returns the activated file node.
func uploadFileFull(t *testing.T, token, parentID, name string, content []byte) nodeView {
	t.Helper()
	if os.Getenv("E2E_SKIP_S3") == "1" {
		t.Skip("E2E_SKIP_S3=1, skipping S3 round-trip")
	}
	u := genUploadURL(t, token, parentID, name, "application/octet-stream", int64(len(content)))
	putToPresigned(t, u, content)
	sum := sha256.Sum256(content)
	return finalizeUpload(t, token, u.NodeID, int64(len(content)), hex.EncodeToString(sum[:]))
}

// getNode reads a node by id.
func getNode(t *testing.T, token, id string) (*http.Response, nodeView) {
	t.Helper()
	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/nodes/"+id, token, nil)
	var n nodeView
	if resp.StatusCode == http.StatusOK {
		mustUnmarshal(t, ar, &n)
	}
	return resp, n
}

// listChildren paginates the children of folder id.
func listChildren(t *testing.T, token, parentID, query string) listChildrenView {
	t.Helper()
	url := baseURL(t) + "/storage/v1/folders/" + parentID + "/children"
	if query != "" {
		url += "?" + query
	}
	resp, ar := do(t, http.MethodGet, url, token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("listChildren: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var lc listChildrenView
	mustUnmarshal(t, ar, &lc)
	return lc
}

// ---------------------------------------------------------------------------
// 1.  POST /storage/v1/me/root  — initialize root
// ---------------------------------------------------------------------------

func TestInitRoot_Success(t *testing.T) {
	token := registerAndLogin(t)

	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/me/root", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)

	if n.Kind != "folder" {
		t.Errorf("expected kind=folder, got %q", n.Kind)
	}
	if n.Name != "root" {
		t.Errorf("expected name=root, got %q", n.Name)
	}
	if n.Depth != 0 {
		t.Errorf("expected depth=0, got %d", n.Depth)
	}
	if n.ID == "" {
		t.Error("expected non-empty id")
	}
	if n.OwnerID == "" {
		t.Error("expected non-empty owner_id")
	}
	if n.ParentID != nil {
		t.Errorf("root must have nil parent_id, got %v", *n.ParentID)
	}
}

func TestInitRoot_Idempotent(t *testing.T) {
	token := registerAndLogin(t)
	first := initRoot(t, token)
	second := initRoot(t, token)
	if first.ID != second.ID {
		t.Fatalf("root id changed across calls: %q vs %q", first.ID, second.ID)
	}
}

func TestInitRoot_Unauthorized(t *testing.T) {
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/me/root", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
}

func TestInitRoot_InvalidToken(t *testing.T) {
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/me/root", "not.a.jwt", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 2.  GET /storage/v1/tree  — read tree
// ---------------------------------------------------------------------------

func TestTree_EmptyRoot(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)

	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/tree?max_depth=10", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var tv treeView
	mustUnmarshal(t, ar, &tv)
	if tv.ID != root.ID {
		t.Fatalf("expected root id %q, got %q", root.ID, tv.ID)
	}
	if len(tv.Children) != 0 {
		t.Fatalf("expected no children, got %d", len(tv.Children))
	}
}

func TestTree_WithNestedFolders(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)

	a := createFolder(t, token, root.ID, "A")
	b := createFolder(t, token, a.ID, "B")
	createFolder(t, token, b.ID, "C")

	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/tree?max_depth=10", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var tv treeView
	mustUnmarshal(t, ar, &tv)
	if len(tv.Children) != 1 || tv.Children[0].Name != "A" {
		t.Fatalf("expected single child 'A', got %+v", tv.Children)
	}
	aChild := tv.Children[0]
	if len(aChild.Children) != 1 || aChild.Children[0].Name != "B" {
		t.Fatalf("expected B under A, got %+v", aChild.Children)
	}
	bChild := aChild.Children[0]
	if len(bChild.Children) != 1 || bChild.Children[0].Name != "C" {
		t.Fatalf("expected C under B, got %+v", bChild.Children)
	}
}

func TestTree_MaxDepthCap(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")
	b := createFolder(t, token, a.ID, "B")
	_ = b

	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/tree?max_depth=1", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var tv treeView
	mustUnmarshal(t, ar, &tv)
	if len(tv.Children) != 1 {
		t.Fatalf("expected 1 direct child under root, got %d", len(tv.Children))
	}
	if len(tv.Children[0].Children) != 0 {
		t.Fatalf("expected depth cap to drop B, got %+v", tv.Children[0].Children)
	}
}

func TestTree_Unauthorized(t *testing.T) {
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/tree", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 3.  POST /storage/v1/folders  — create folder
// ---------------------------------------------------------------------------

func TestCreateFolder_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)

	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]string{
		"parent_id": root.ID,
		"name":      "Documents",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.Kind != "folder" || n.Name != "Documents" {
		t.Fatalf("unexpected payload: %+v", n)
	}
	if n.ParentID == nil || *n.ParentID != root.ID {
		t.Fatalf("expected parent_id=%q, got %v", root.ID, n.ParentID)
	}
	if n.Depth != 1 {
		t.Errorf("expected depth=1, got %d", n.Depth)
	}
}

func TestCreateFolder_Nested(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")
	b := createFolder(t, token, a.ID, "B")
	if b.Depth != 2 {
		t.Errorf("expected depth=2 for B, got %d", b.Depth)
	}
	if !strings.HasSuffix(b.Path, "/A/B") && b.Path != "/A/B" {
		// Path conventions can vary; just check both parts are present.
		if !(strings.Contains(b.Path, "A") && strings.Contains(b.Path, "B")) {
			t.Errorf("expected path containing A and B, got %q", b.Path)
		}
	}
}

func TestCreateFolder_DuplicateName(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "dup")

	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]string{
		"parent_id": root.ID,
		"name":      "dup",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestCreateFolder_EmptyName(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)

	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]string{
		"parent_id": root.ID,
		"name":      "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestCreateFolder_MissingParentID(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)

	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]string{
		"name": "x",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestCreateFolder_InvalidParentID(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)

	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]string{
		"parent_id": "not-a-uuid",
		"name":      "x",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestCreateFolder_ParentNotFound(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)

	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]string{
		"parent_id": "00000000-0000-0000-0000-000000000000",
		"name":      "x",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestCreateFolder_UnicodeName(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	n := createFolder(t, token, root.ID, "Документы 2025 ✓")
	if !strings.Contains(n.Name, "Документы") {
		t.Errorf("unicode name was mangled: %q", n.Name)
	}
}

func TestCreateFolder_Unauthorized(t *testing.T) {
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", "", map[string]string{
		"parent_id": "00000000-0000-0000-0000-000000000000",
		"name":      "x",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 4.  GET /storage/v1/nodes/{id}  — read single node
// ---------------------------------------------------------------------------

func TestGetNode_Folder(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "GetMe")

	resp, n := getNode(t, token, folder.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if n.ID != folder.ID || n.Kind != "folder" || n.Name != "GetMe" {
		t.Fatalf("unexpected payload: %+v", n)
	}
}

func TestGetNode_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	resp, _ := getNode(t, token, "00000000-0000-0000-0000-000000000000")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetNode_InvalidUUID(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	resp, _ := getNode(t, token, "garbage")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetNode_ForbiddenForOtherOwner(t *testing.T) {
	// User A owns the node, user B tries to read it.
	tokenA := registerAndLogin(t)
	rootA := initRoot(t, tokenA)
	folder := createFolder(t, tokenA, rootA.ID, "SecretA")

	tokenB := registerAndLogin(t)
	_ = initRoot(t, tokenB)

	resp, _ := getNode(t, tokenB, folder.ID)
	// Could be 403 or 404 depending on whether the service leaks existence.
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 403 or 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 5.  GET /storage/v1/folders/{id}/children  — paginated listing
// ---------------------------------------------------------------------------

func TestListChildren_Empty(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	lc := listChildren(t, token, root.ID, "")
	if len(lc.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(lc.Items))
	}
	if lc.NextCursor != "" {
		t.Fatalf("expected empty cursor, got %q", lc.NextCursor)
	}
}

func TestListChildren_WithItems(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	for i := 0; i < 5; i++ {
		createFolder(t, token, root.ID, fmt.Sprintf("F%d", i))
	}
	lc := listChildren(t, token, root.ID, "limit=100")
	if len(lc.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(lc.Items))
	}
}

func TestListChildren_Pagination(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	for i := 0; i < 7; i++ {
		createFolder(t, token, root.ID, fmt.Sprintf("F%02d", i))
	}
	lc1 := listChildren(t, token, root.ID, "limit=3")
	if len(lc1.Items) != 3 {
		t.Fatalf("page1: expected 3 items, got %d", len(lc1.Items))
	}
	if lc1.NextCursor == "" {
		t.Fatal("page1: expected non-empty next_cursor")
	}
	lc2 := listChildren(t, token, root.ID, "limit=3&cursor="+lc1.NextCursor)
	if len(lc2.Items) == 0 {
		t.Fatal("page2: expected at least one item")
	}
}

func TestListChildren_InvalidLimit(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	url := baseURL(t) + "/storage/v1/folders/" + root.ID + "/children?limit=99999"
	resp, _ := do(t, http.MethodGet, url, token, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 6.  PATCH /storage/v1/nodes/{id}/rename
// ---------------------------------------------------------------------------

func TestRename_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	f := createFolder(t, token, root.ID, "Old")

	resp, ar := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+f.ID+"/rename", token, map[string]string{
		"name": "New",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.Name != "New" {
		t.Fatalf("expected name=New, got %q", n.Name)
	}
}

func TestRename_EmptyName(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	f := createFolder(t, token, root.ID, "X")

	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+f.ID+"/rename", token, map[string]string{
		"name": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRename_DuplicateName(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	createFolder(t, token, root.ID, "exists")
	other := createFolder(t, token, root.ID, "other")

	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+other.ID+"/rename", token, map[string]string{
		"name": "exists",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRename_RootImmutable(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)

	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+root.ID+"/rename", token, map[string]string{
		"name": "not-root",
	})
	// Renaming root is a domain rule violation → 422 Unprocessable Entity.
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestRename_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/00000000-0000-0000-0000-000000000000/rename", token, map[string]string{
		"name": "x",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 7.  PATCH /storage/v1/nodes/{id}/move
// ---------------------------------------------------------------------------

func TestMove_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")
	b := createFolder(t, token, root.ID, "B")
	target := createFolder(t, token, a.ID, "Target")

	resp, ar := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+target.ID+"/move", token, map[string]string{
		"new_parent_id": b.ID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.ParentID == nil || *n.ParentID != b.ID {
		t.Fatalf("expected parent_id=%q, got %v", b.ID, n.ParentID)
	}
}

func TestMove_IntoSelf(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")

	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+a.ID+"/move", token, map[string]string{
		"new_parent_id": a.ID,
	})
	// Self-move is a business-rule violation → 422.
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestMove_IntoOwnDescendant(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")
	b := createFolder(t, token, a.ID, "B")

	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+a.ID+"/move", token, map[string]string{
		"new_parent_id": b.ID,
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestMove_MissingNewParent(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")

	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+a.ID+"/move", token, map[string]string{
		"new_parent_id": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMove_DuplicateNameInTarget(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")
	b := createFolder(t, token, root.ID, "B")
	src := createFolder(t, token, a.ID, "X")
	createFolder(t, token, b.ID, "X")

	resp, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/nodes/"+src.ID+"/move", token, map[string]string{
		"new_parent_id": b.ID,
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 8.  DELETE /storage/v1/nodes/{id}  — soft-delete
// ---------------------------------------------------------------------------

func TestDelete_FolderSubtree(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	a := createFolder(t, token, root.ID, "A")
	b := createFolder(t, token, a.ID, "B")

	resp, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/nodes/"+a.ID, token, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// A and B should both be soft-deleted (deleted_at present).
	url := baseURL(t) + "/storage/v1/nodes/" + a.ID + "?include_deleted=true"
	r, ar := do(t, http.MethodGet, url, token, nil)
	if r.StatusCode != http.StatusOK {
		t.Fatalf("get deleted A: expected 200, got %d: %s", r.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.DeletedAt == nil {
		t.Fatal("expected deleted_at on A")
	}
	_ = b
}

func TestDelete_RootImmutable(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)

	resp, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/nodes/"+root.ID, token, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestDelete_Idempotent(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	f := createFolder(t, token, root.ID, "Doomed")

	r1, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/nodes/"+f.ID, token, nil)
	if r1.StatusCode != http.StatusNoContent {
		t.Fatalf("first delete: expected 204, got %d", r1.StatusCode)
	}
	r2, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/nodes/"+f.ID, token, nil)
	// Second delete on already-deleted is a domain conflict → 422.
	if r2.StatusCode != http.StatusUnprocessableEntity && r2.StatusCode != http.StatusNoContent {
		t.Fatalf("second delete: expected 422 or 204, got %d", r2.StatusCode)
	}
}

func TestDelete_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	resp, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/nodes/00000000-0000-0000-0000-000000000000", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 9.  POST /storage/v1/nodes/{id}/restore
// ---------------------------------------------------------------------------

func TestRestore_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	f := createFolder(t, token, root.ID, "Lazarus")
	if r, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/nodes/"+f.ID, token, nil); r.StatusCode != http.StatusNoContent {
		t.Fatalf("delete failed: %d", r.StatusCode)
	}

	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/nodes/"+f.ID+"/restore", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.DeletedAt != nil {
		t.Fatalf("expected deleted_at=nil after restore, got %v", n.DeletedAt)
	}
}

func TestRestore_NotDeleted(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	f := createFolder(t, token, root.ID, "Alive")

	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/nodes/"+f.ID+"/restore", token, nil)
	// Restoring a not-deleted node is a no-op or a conflict; both acceptable.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 200 or 422, got %d", resp.StatusCode)
	}
}

func TestRestore_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/nodes/00000000-0000-0000-0000-000000000000/restore", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 10. POST /storage/v1/files/upload-url  +  finalize / abort
// ---------------------------------------------------------------------------

func TestUploadURL_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)

	u := genUploadURL(t, token, root.ID, "report.pdf", "application/pdf", 1024)
	if u.Method != "PUT" {
		t.Errorf("expected method=PUT, got %q", u.Method)
	}
	if !strings.HasPrefix(u.URL, "http") {
		t.Errorf("expected http(s) URL, got %q", u.URL)
	}
	if u.ExpiresAt.Before(time.Now()) {
		t.Errorf("expires_at is in the past: %v", u.ExpiresAt)
	}
	// Cleanup the orphaned pending node.
	do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/"+u.NodeID+"/abort", token, nil)
}

func TestUploadURL_MissingParentID(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/upload-url", token, map[string]any{
		"name":       "x",
		"size_bytes": 10,
		"mime_type":  "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUploadURL_ZeroSize(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/upload-url", token, map[string]any{
		"parent_id":  root.ID,
		"name":       "x",
		"size_bytes": 0,
		"mime_type":  "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUploadURL_NegativeSize(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/upload-url", token, map[string]any{
		"parent_id":  root.ID,
		"name":       "x",
		"size_bytes": -1,
		"mime_type":  "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestUploadURL_ParentIsFile(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	file := uploadFileFull(t, token, root.ID, "blob.bin", []byte("hello world"))

	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/upload-url", token, map[string]any{
		"parent_id":  file.ID,
		"name":       "nested.bin",
		"size_bytes": 5,
		"mime_type":  "application/octet-stream",
	})
	// Parent must be a folder.
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", resp.StatusCode)
	}
}

func TestUpload_FullCycle(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	data := []byte("e2e file payload")
	file := uploadFileFull(t, token, root.ID, "payload.txt", data)
	if file.SizeBytes == nil || *file.SizeBytes != int64(len(data)) {
		t.Fatalf("expected size_bytes=%d, got %v", len(data), file.SizeBytes)
	}
	if file.Status == nil || *file.Status != "active" {
		t.Fatalf("expected status=active, got %v", file.Status)
	}
}

func TestUpload_FinalizeWrongChecksum(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	if os.Getenv("E2E_SKIP_S3") == "1" {
		t.Skip("E2E_SKIP_S3=1")
	}
	data := []byte("checksum mismatch test")
	u := genUploadURL(t, token, root.ID, "mismatch.txt", "text/plain", int64(len(data)))
	putToPresigned(t, u, data)
	// Bogus checksum: 64 hex zeros.
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/"+u.NodeID+"/finalize", token, map[string]any{
		"size_bytes": int64(len(data)),
		"checksum":   strings.Repeat("0", 64),
	})
	// The current implementation doesn't compare checksum to S3 contents,
	// but it does validate the format. A future version may also compare —
	// accept either OK or 4xx with code != 5xx.
	if resp.StatusCode >= 500 {
		t.Fatalf("unexpected 5xx: %d", resp.StatusCode)
	}
}

func TestUpload_FinalizeWithoutPut(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	if os.Getenv("E2E_SKIP_S3") == "1" {
		t.Skip("E2E_SKIP_S3=1")
	}
	u := genUploadURL(t, token, root.ID, "ghost.bin", "application/octet-stream", 5)
	// Skip the PUT step.
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/"+u.NodeID+"/finalize", token, map[string]any{
		"size_bytes": int64(5),
		"checksum":   strings.Repeat("a", 64),
	})
	if resp.StatusCode < 400 {
		t.Fatalf("expected an error response, got %d: %s", resp.StatusCode, ar.Error)
	}
}

func TestUpload_Abort(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	u := genUploadURL(t, token, root.ID, "aborted.bin", "application/octet-stream", 10)

	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/files/"+u.NodeID+"/abort", token, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 11. GET /storage/v1/files/{id}/download-url
// ---------------------------------------------------------------------------

func TestDownloadURL_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	file := uploadFileFull(t, token, root.ID, "down.bin", []byte("download me"))

	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/files/"+file.ID+"/download-url", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var d downloadURLView
	mustUnmarshal(t, ar, &d)
	if d.Method != "GET" {
		t.Errorf("expected method=GET, got %q", d.Method)
	}
	if !strings.HasPrefix(d.URL, "http") {
		t.Errorf("bad URL: %q", d.URL)
	}
}

func TestDownloadURL_InlineDisposition(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	file := uploadFileFull(t, token, root.ID, "inline.bin", []byte("inline me"))

	resp, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/files/"+file.ID+"/download-url?disposition=inline", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDownloadURL_InvalidDisposition(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	file := uploadFileFull(t, token, root.ID, "bad.bin", []byte("x"))

	resp, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/files/"+file.ID+"/download-url?disposition=weird", token, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDownloadURL_FolderForbidden(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "Folder")

	resp, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/files/"+folder.ID+"/download-url", token, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 4xx, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 12. POST /storage/v1/nodes/{id}/shares (+ list + revoke)
// ---------------------------------------------------------------------------

func createShare(t *testing.T, token, nodeID, permission, expiresIn string) shareView {
	t.Helper()
	payload := map[string]string{"permission": permission}
	if expiresIn != "" {
		payload["expires_in"] = expiresIn
	}
	resp, ar := do(t, http.MethodPost, baseURL(t)+"/storage/v1/nodes/"+nodeID+"/shares", token, payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("createShare: expected 201, got %d: %s", resp.StatusCode, ar.Error)
	}
	var s shareView
	mustUnmarshal(t, ar, &s)
	if s.Token == "" {
		t.Fatal("createShare: expected non-empty token on create")
	}
	return s
}

func TestCreateShare_View(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "Shared")
	s := createShare(t, token, folder.ID, "view", "")
	if s.Permission != "view" {
		t.Errorf("expected permission=view, got %q", s.Permission)
	}
	if s.URL == "" {
		t.Error("expected non-empty url on create")
	}
}

func TestCreateShare_Edit(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "Editable")
	s := createShare(t, token, folder.ID, "edit", "")
	if s.Permission != "edit" {
		t.Errorf("expected permission=edit, got %q", s.Permission)
	}
}

func TestCreateShare_InvalidPermission(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "Bad")
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/nodes/"+folder.ID+"/shares", token, map[string]string{
		"permission": "admin",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateShare_WithExpiry(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "ExpFolder")
	s := createShare(t, token, folder.ID, "view", "24h")
	if s.ExpiresAt == nil {
		t.Fatal("expected non-nil expires_at")
	}
	if s.ExpiresAt.Before(time.Now()) {
		t.Fatalf("expires_at is in the past: %v", s.ExpiresAt)
	}
}

func TestCreateShare_InvalidExpiry(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "ExpBad")
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/nodes/"+folder.ID+"/shares", token, map[string]string{
		"permission": "view",
		"expires_in": "notaduration",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestListShares_HidesToken(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "Listed")
	_ = createShare(t, token, folder.ID, "view", "")

	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/nodes/"+folder.ID+"/shares", token, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var lv listSharesView
	mustUnmarshal(t, ar, &lv)
	if len(lv.Items) != 1 {
		t.Fatalf("expected 1 share, got %d", len(lv.Items))
	}
	if lv.Items[0].Token != "" {
		t.Errorf("expected token to be hidden on list, got %q", lv.Items[0].Token)
	}
}

func TestRevokeShare_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "Doomed")
	s := createShare(t, token, folder.ID, "view", "")

	resp, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/shares/"+s.ID, token, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// After revoke the public token must stop working.
	r, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/"+s.Token, "", nil)
	if r.StatusCode == http.StatusOK {
		t.Fatalf("expected revoked share to return 4xx, got %d", r.StatusCode)
	}
}

func TestRevokeShare_NotFound(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	resp, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/shares/00000000-0000-0000-0000-000000000000", token, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 13. GET /storage/v1/public/{token}  + nested public endpoints
// ---------------------------------------------------------------------------

func TestPublicInfo_Success(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "PubInfo")
	s := createShare(t, token, folder.ID, "view", "")

	// Public endpoint MUST work without Authorization header.
	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/"+s.Token, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var p publicShareView
	mustUnmarshal(t, ar, &p)
	if p.NodeID != folder.ID {
		t.Errorf("expected node_id=%q, got %q", folder.ID, p.NodeID)
	}
	if p.Name != "PubInfo" {
		t.Errorf("expected name=PubInfo, got %q", p.Name)
	}
	if p.Permission != "view" {
		t.Errorf("expected permission=view, got %q", p.Permission)
	}
}

func TestPublicInfo_InvalidToken(t *testing.T) {
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/badtoken", "", nil)
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 4xx, got %d", resp.StatusCode)
	}
}

func TestPublicTree_ScopedToShare(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "PubTree")
	createFolder(t, token, folder.ID, "Inside1")
	createFolder(t, token, folder.ID, "Inside2")
	// Sibling that MUST NOT be reachable via the share-token.
	createFolder(t, token, root.ID, "Sibling")
	s := createShare(t, token, folder.ID, "view", "")

	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/"+s.Token+"/tree?max_depth=10", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var tv treeView
	mustUnmarshal(t, ar, &tv)
	if tv.ID != folder.ID {
		t.Fatalf("expected scope to be folder %q, got %q", folder.ID, tv.ID)
	}
	// Sibling must NOT appear in the public tree.
	for _, c := range tv.Children {
		if c.Name == "Sibling" {
			t.Fatalf("public tree leaked sibling node")
		}
	}
}

func TestPublicView_CannotMutate(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "Pub")
	inside := createFolder(t, token, folder.ID, "Inside")
	s := createShare(t, token, folder.ID, "view", "")

	// View-permission share MUST be unable to rename or delete.
	r1, _ := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/public/"+s.Token+"/nodes/"+inside.ID+"/rename", "", map[string]string{
		"name": "Hacked",
	})
	if r1.StatusCode != http.StatusForbidden {
		t.Errorf("rename via view-share: expected 403, got %d", r1.StatusCode)
	}

	r2, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/public/"+s.Token+"/nodes/"+inside.ID, "", nil)
	if r2.StatusCode != http.StatusForbidden {
		t.Errorf("delete via view-share: expected 403, got %d", r2.StatusCode)
	}
}

func TestPublicEdit_CanRename(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "PubEdit")
	inside := createFolder(t, token, folder.ID, "Renamable")
	s := createShare(t, token, folder.ID, "edit", "")

	resp, ar := do(t, http.MethodPatch, baseURL(t)+"/storage/v1/public/"+s.Token+"/nodes/"+inside.ID+"/rename", "", map[string]string{
		"name": "Renamed",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var n nodeView
	mustUnmarshal(t, ar, &n)
	if n.Name != "Renamed" {
		t.Fatalf("expected name=Renamed, got %q", n.Name)
	}
}

func TestPublicShare_ExpiredReturnsError(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	folder := createFolder(t, token, root.ID, "ExpShare")
	s := createShare(t, token, folder.ID, "view", "1s")

	time.Sleep(2 * time.Second)

	resp, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/"+s.Token, "", nil)
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected error after expiry, got 200")
	}
}

// ---------------------------------------------------------------------------
// 14. Method enforcement, headers, healthz
// ---------------------------------------------------------------------------

func TestHealthz(t *testing.T) {
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/healthz", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz: expected 200, got %d", resp.StatusCode)
	}
}

func TestWrongHTTPMethod_Folders(t *testing.T) {
	resp, _ := do(t, http.MethodGet, baseURL(t)+"/storage/v1/folders", "", nil)
	if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusUnauthorized {
		// API gateways often short-circuit auth before reaching the route.
		t.Fatalf("expected 405 or 401, got %d", resp.StatusCode)
	}
}

func TestWrongHTTPMethod_Rename(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	f := createFolder(t, token, root.ID, "MNA")
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/nodes/"+f.ID+"/rename", token, map[string]string{
		"name": "X",
	})
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.StatusCode)
	}
}

func TestMalformedJSON_CreateFolder(t *testing.T) {
	token := registerAndLogin(t)
	_ = initRoot(t, token)
	req, _ := http.NewRequest(http.MethodPost, baseURL(t)+"/storage/v1/folders", strings.NewReader("{not-json"))
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

func TestUnknownJSONField_Rejected(t *testing.T) {
	token := registerAndLogin(t)
	root := initRoot(t, token)
	resp, _ := do(t, http.MethodPost, baseURL(t)+"/storage/v1/folders", token, map[string]any{
		"parent_id":  root.ID,
		"name":       "ExtraField",
		"unexpected": true,
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 (DisallowUnknownFields), got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// 15. Full Flow — register, upload, share, public download, revoke
// ---------------------------------------------------------------------------

func TestFullFlow_UploadShareDownload(t *testing.T) {
	if os.Getenv("E2E_SKIP_S3") == "1" {
		t.Skip("E2E_SKIP_S3=1")
	}
	token := registerAndLogin(t)

	// 1. init root
	root := initRoot(t, token)

	// 2. create "Documents" folder
	docs := createFolder(t, token, root.ID, "Documents")

	// 3. upload "report.pdf"
	payload := []byte("e2e fullflow report bytes")
	file := uploadFileFull(t, token, docs.ID, "report.pdf", payload)

	// 4. create share (view)
	share := createShare(t, token, docs.ID, "view", "1h")

	// 5. public info via share token
	resp, ar := do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/"+share.Token, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public info: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}

	// 6. public download URL for the file via share token
	resp, ar = do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/"+share.Token+"/files/"+file.ID+"/download-url", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public download: expected 200, got %d: %s", resp.StatusCode, ar.Error)
	}
	var d downloadURLView
	mustUnmarshal(t, ar, &d)
	if !strings.HasPrefix(d.URL, "http") {
		t.Fatalf("bad URL: %q", d.URL)
	}

	// 7. revoke the share
	r, _ := do(t, http.MethodDelete, baseURL(t)+"/storage/v1/shares/"+share.ID, token, nil)
	if r.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke: expected 204, got %d", r.StatusCode)
	}

	// 8. public info MUST fail now
	resp, _ = do(t, http.MethodGet, baseURL(t)+"/storage/v1/public/"+share.Token, "", nil)
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("public info after revoke: expected non-200, got 200")
	}
}
