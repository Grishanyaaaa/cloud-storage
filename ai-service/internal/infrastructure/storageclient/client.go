// Package storageclient implements port.StorageClient against storage-service's
// HTTP API. The caller's JWT is propagated as the Authorization header on
// every request — ai-service has no service-to-service credentials of its own.
package storageclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/config"
)

// Compile-time check: StorageClient implements port.StorageClient
var _ port.StorageClient = (*StorageClient)(nil)

// StorageClient is a thin REST client over storage-service.
type StorageClient struct {
	cfg  config.StorageServiceConfig
	http *http.Client
}

// NewStorageClient builds a StorageClient from the storage-service config.
func NewStorageClient(cfg config.StorageServiceConfig) *StorageClient {
	return &StorageClient{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// envelope is storage-service's standard JSON wrapper.
type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	Code   string          `json:"code,omitempty"`
}

// treeResponse mirrors dto.TreeNodeResponse (recursive).
type treeResponse struct {
	ID       string         `json:"id"`
	OwnerID  string         `json:"owner_id"`
	ParentID *string        `json:"parent_id,omitempty"`
	Kind     string         `json:"kind"`
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Depth    int            `json:"depth"`
	Children []treeResponse `json:"children,omitempty"`
}

// GetTree calls GET /storage/v1/tree and flattens the response.
//
// maxDepth ≤ 0 → omit the `max_depth` query param (storage-service uses its default).
// maxNodes ≤ 0 → no client-side cap (still bounded by storage-service).
func (c *StorageClient) GetTree(ctx context.Context, jwt string, maxDepth, maxNodes int) ([]port.TreeNode, error) {
	if maxDepth <= 0 {
		maxDepth = c.cfg.TreeMaxDepth
	}
	q := "?max_depth=" + strconv.Itoa(maxDepth)
	resp, err := c.do(ctx, http.MethodGet, "/storage/v1/tree"+q, jwt, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := decodeEnvelope(resp)
	if err != nil {
		return nil, err
	}

	var tree treeResponse
	if err := json.Unmarshal(body, &tree); err != nil {
		return nil, fmt.Errorf("decode tree response: %w", err)
	}

	if maxNodes <= 0 {
		maxNodes = c.cfg.TreeMaxNodes
	}
	flat := flattenTree(tree, maxNodes)
	return flat, nil
}

// DeleteNode calls DELETE /storage/v1/nodes/{id}.
func (c *StorageClient) DeleteNode(ctx context.Context, jwt string, nodeID valueobject.NodeID) error {
	resp, err := c.do(ctx, http.MethodDelete, "/storage/v1/nodes/"+nodeID.String(), jwt, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkStatusOnly(resp)
}

type renameBody struct {
	Name string `json:"name"`
}

// RenameNode calls PATCH /storage/v1/nodes/{id}/rename.
func (c *StorageClient) RenameNode(ctx context.Context, jwt string, nodeID valueobject.NodeID, newName string) error {
	body, err := json.Marshal(renameBody{Name: newName})
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPatch, "/storage/v1/nodes/"+nodeID.String()+"/rename", jwt, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkStatusOnly(resp)
}

type moveBody struct {
	NewParentID string `json:"new_parent_id"`
}

// MoveNode calls PATCH /storage/v1/nodes/{id}/move.
func (c *StorageClient) MoveNode(ctx context.Context, jwt string, nodeID valueobject.NodeID, newParentID valueobject.NodeID) error {
	body, err := json.Marshal(moveBody{NewParentID: newParentID.String()})
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPatch, "/storage/v1/nodes/"+nodeID.String()+"/move", jwt, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkStatusOnly(resp)
}

// do issues the HTTP request with the JWT as Authorization Bearer header.
func (c *StorageClient) do(ctx context.Context, method, path, jwt string, body []byte) (*http.Response, error) {
	url := strings.TrimRight(c.cfg.BaseURL, "/") + path
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, fmt.Errorf("build storage-service request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, domainerr.New(
			domainerr.CodeStorageServiceUnavailable,
			"storage-service request failed",
			err,
		)
	}
	return resp, nil
}

// decodeEnvelope reads the response body and validates the storage-service
// envelope. Returns the unwrapped `data` JSON or a domain error.
func decodeEnvelope(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, domainerr.New(
			domainerr.CodeStorageServiceUnavailable,
			"read storage-service response",
			err,
		)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, domainerr.ErrUnauthorized
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, domainerr.ErrForbidden
	}
	if resp.StatusCode == http.StatusNotFound {
		// No structured way to know which entity is missing — surface as INVALID_NODE_ID
		// when the caller knows the route was a node-targeting one. For tree we
		// treat 404 as a clear unavailability of the user's tree.
		var env envelope
		_ = json.Unmarshal(body, &env)
		return nil, domainerr.New(
			domainerr.CodeStorageServiceUnavailable,
			fmt.Sprintf("storage-service 404: %s", env.Error),
			nil,
		)
	}
	if resp.StatusCode >= 500 {
		return nil, domainerr.New(
			domainerr.CodeStorageServiceUnavailable,
			fmt.Sprintf("storage-service %d", resp.StatusCode),
			nil,
		)
	}
	if resp.StatusCode >= 400 {
		var env envelope
		_ = json.Unmarshal(body, &env)
		return nil, domainerr.New(
			pickCode(env.Code, domainerr.CodeBadRequest),
			fmt.Sprintf("storage-service %d: %s", resp.StatusCode, env.Error),
			nil,
		)
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, domainerr.New(
			domainerr.CodeStorageServiceUnavailable,
			"malformed storage-service envelope",
			err,
		)
	}
	if env.Status != "success" {
		return nil, domainerr.New(
			pickCode(env.Code, domainerr.CodeBadRequest),
			fmt.Sprintf("storage-service: %s", env.Error),
			nil,
		)
	}
	return env.Data, nil
}

// checkStatusOnly is decodeEnvelope for endpoints that return 204 No Content
// or do not need the data payload.
func checkStatusOnly(resp *http.Response) error {
	if resp.StatusCode == http.StatusNoContent || (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		// drain body to allow connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return domainerr.ErrUnauthorized
	case http.StatusForbidden:
		return domainerr.ErrForbidden
	case http.StatusNotFound:
		return domainerr.New(domainerr.CodeInvalidNodeID, "storage-service 404", nil)
	default:
		var env envelope
		_ = json.Unmarshal(body, &env)
		if resp.StatusCode >= 500 {
			return domainerr.New(
				domainerr.CodeStorageServiceUnavailable,
				fmt.Sprintf("storage-service %d", resp.StatusCode),
				nil,
			)
		}
		return domainerr.New(
			pickCode(env.Code, domainerr.CodeBadRequest),
			fmt.Sprintf("storage-service %d: %s", resp.StatusCode, env.Error),
			nil,
		)
	}
}

func pickCode(provided, fallback string) string {
	if provided != "" {
		return provided
	}
	return fallback
}

// flattenTree walks the recursive treeResponse and returns at most maxNodes
// flat TreeNode entries. Walks BFS so that pruning preserves shallow nodes.
func flattenTree(root treeResponse, maxNodes int) []port.TreeNode {
	queue := []treeResponse{root}
	out := make([]port.TreeNode, 0, 64)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		nodeID, err := valueobject.ParseNodeID(cur.ID)
		if err != nil {
			// Skip malformed ids defensively (should never happen).
			continue
		}
		var parent *valueobject.NodeID
		if cur.ParentID != nil && *cur.ParentID != "" {
			pid, err := valueobject.ParseNodeID(*cur.ParentID)
			if err == nil {
				parent = &pid
			}
		}

		out = append(out, port.TreeNode{
			ID:       nodeID,
			ParentID: parent,
			Kind:     cur.Kind,
			Name:     cur.Name,
			Path:     cur.Path,
			Depth:    cur.Depth,
		})
		if maxNodes > 0 && len(out) >= maxNodes {
			break
		}
		queue = append(queue, cur.Children...)
	}
	return out
}
