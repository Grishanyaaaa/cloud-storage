package usecase

import (
	"fmt"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// planValidator double-checks each operation against the user's tree before
// the plan is persisted as awaiting_confirmation.
//
// What it catches:
//   - hallucinated node_id (id not in tree);
//   - rename/delete/move of the user's root (forbidden);
//   - move where new_parent_id is not a folder;
//   - move where new_parent_id is the node itself or a descendant of it;
//   - rename with a name that is obviously unsafe (contains '/', '\', '\0').
//
// Anything subtler (NodeName length, name collision in target folder, etc.)
// is left to storage-service to validate at execution time — there is no
// point duplicating that here.
type planValidator struct{}

func newPlanValidator() *planValidator { return &planValidator{} }

// Validate returns nil on success, otherwise a domainerr describing the issue.
func (v *planValidator) Validate(ops []entity.Operation, tree []port.TreeNode) error {
	if len(ops) == 0 {
		// Empty plan is allowed (LLM declined or asked for clarification).
		return nil
	}

	idx := indexTree(tree)

	for i, op := range ops {
		if err := v.validateOne(i, op, idx); err != nil {
			return err
		}
	}
	return nil
}

func (v *planValidator) validateOne(idx int, op entity.Operation, tree map[valueobject.NodeID]port.TreeNode) error {
	target, ok := tree[op.NodeID()]
	if !ok {
		return domainerr.New(
			domainerr.CodeInvalidPlan,
			fmt.Sprintf("operation #%d: node_id %s not found in tree", idx, op.NodeID().String()),
			nil,
		)
	}
	// Disallow operations on the user's root.
	if target.Depth == 1 || target.ParentID == nil {
		return domainerr.New(
			domainerr.CodeInvalidPlan,
			fmt.Sprintf("operation #%d: cannot modify root node", idx),
			nil,
		)
	}

	switch {
	case op.Kind().IsDelete():
		return nil

	case op.Kind().IsRename():
		if err := validateNewName(op.NewName()); err != nil {
			return domainerr.New(
				domainerr.CodeInvalidPlan,
				fmt.Sprintf("operation #%d: %s", idx, err.Error()),
				err,
			)
		}
		return nil

	case op.Kind().IsMove():
		newParent := op.NewParentID()
		if newParent == nil {
			return domainerr.New(
				domainerr.CodeInvalidPlan,
				fmt.Sprintf("operation #%d: move requires new_parent_id", idx),
				nil,
			)
		}
		parent, ok := tree[*newParent]
		if !ok {
			return domainerr.New(
				domainerr.CodeInvalidPlan,
				fmt.Sprintf("operation #%d: new_parent_id %s not found in tree", idx, newParent.String()),
				nil,
			)
		}
		if parent.Kind != "folder" {
			return domainerr.New(
				domainerr.CodeInvalidPlan,
				fmt.Sprintf("operation #%d: new_parent_id %s is not a folder", idx, newParent.String()),
				nil,
			)
		}
		// Self / descendant check.
		if op.NodeID().Equals(*newParent) {
			return domainerr.New(
				domainerr.CodeInvalidPlan,
				fmt.Sprintf("operation #%d: cannot move node into itself", idx),
				nil,
			)
		}
		if isDescendant(tree, *newParent, op.NodeID()) {
			return domainerr.New(
				domainerr.CodeInvalidPlan,
				fmt.Sprintf("operation #%d: cannot move node into its descendant", idx),
				nil,
			)
		}
		return nil

	default:
		return domainerr.ErrInvalidOperationKind
	}
}

// indexTree builds a fast id → node lookup.
func indexTree(tree []port.TreeNode) map[valueobject.NodeID]port.TreeNode {
	idx := make(map[valueobject.NodeID]port.TreeNode, len(tree))
	for _, n := range tree {
		idx[n.ID] = n
	}
	return idx
}

// isDescendant reports whether `candidate` is a descendant of `ancestor` in the tree.
// Walks parent pointers from candidate up; bounded by tree size.
func isDescendant(idx map[valueobject.NodeID]port.TreeNode, candidate, ancestor valueobject.NodeID) bool {
	cur, ok := idx[candidate]
	if !ok {
		return false
	}
	hops := 0
	for cur.ParentID != nil {
		if hops > len(idx) {
			return false // safety against malformed input
		}
		hops++
		if cur.ParentID.Equals(ancestor) {
			return true
		}
		next, ok := idx[*cur.ParentID]
		if !ok {
			return false
		}
		cur = next
	}
	return false
}

// validateNewName performs a minimal sanity check on a rename target.
func validateNewName(s string) error {
	t := strings.TrimSpace(s)
	if t == "" {
		return fmt.Errorf("new_name is empty")
	}
	if strings.ContainsAny(t, "/\\\x00") {
		return fmt.Errorf("new_name contains forbidden characters")
	}
	return nil
}
