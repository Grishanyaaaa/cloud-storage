package usecase

import (
	"sort"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// promptBuilder assembles the system + user messages and the JSON Schema
// passed to Yandex GPT for plan generation.
//
// Design notes:
//   - Tree is serialized as plain text (not JSON) to minimise tokens. Each line
//     is "<indent><id> <kind> <name>" — easy for LLM to read and reference.
//   - All instructions are in Russian (per project requirement).
//   - The JSON Schema enforces the response shape; Yandex GPT returns a JSON
//     string in `result.alternatives[0].message.text` that we then parse.
type promptBuilder struct{}

func newPromptBuilder() *promptBuilder { return &promptBuilder{} }

// systemPrompt — главные инструкции модели (русский).
const systemPrompt = `Ты — ассистент облачного файлового хранилища. Пользователь даёт команду на естественном языке (русский), а ты возвращаешь СТРОГО JSON-объект описывающий план операций над файловой структурой.

Доступные операции:
  - delete: удалить узел (файл или папку).
  - rename: переименовать узел в new_name.
  - move:   переместить узел в new_parent_id (id папки-приёмника).

Правила:
  1. Возвращай ТОЛЬКО валидный JSON по предоставленной схеме. Никаких комментариев, кода, markdown, текста до или после JSON.
  2. Используй ТОЛЬКО узлы с node_id, которые присутствуют в дереве пользователя. Никогда не выдумывай идентификаторы.
  3. Если команда неоднозначна, бессмысленна или невозможна — верни пустой массив "ops" и поясни ситуацию в "explanation".
  4. Нельзя перемещать узел в самого себя или в собственного потомка. Целевой new_parent_id должен быть папкой (kind=folder).
  5. Для rename new_name должен быть непустой строкой без слэшей и других служебных символов.
  6. "explanation" — короткое (1-3 предложения) объяснение результата на русском, обращайся к пользователю на "вы".
  7. Если запрос затрагивает несколько узлов (например "удали все файлы 2024") — перечисли каждую операцию отдельно в массиве "ops".
  8. Корневой узел пользователя (depth=1) удалять/переименовывать/перемещать НЕЛЬЗЯ. Если команда требует этого — верни пустой ops.
`

// Build returns systemMsg, userMsg, jsonSchema.
//
//	tree     — flattened nodes returned by storage-service.
//	rootHint — optional id of the user's root (used to mark it in the prompt).
//	input    — user-supplied natural-language command (already validated).
//	maxNodes — soft cap for how many tree lines to emit. If exceeded, the
//	           deepest nodes are pruned first to keep top-level context.
func (b *promptBuilder) Build(
	tree []port.TreeNode,
	rootHint valueobject.NodeID,
	input string,
	maxNodes int,
) (systemMsg, userMsg string, jsonSchema map[string]any) {
	systemMsg = systemPrompt
	userMsg = b.buildUserMessage(tree, rootHint, input, maxNodes)
	jsonSchema = b.buildJSONSchema()
	return
}

func (b *promptBuilder) buildUserMessage(
	tree []port.TreeNode,
	rootHint valueobject.NodeID,
	input string,
	maxNodes int,
) string {
	var sb strings.Builder
	sb.WriteString("Дерево файлов пользователя (id, тип, имя). Корневой узел отмечен '*':\n")
	sb.WriteString(b.serializeTree(tree, rootHint, maxNodes))
	sb.WriteString("\n\nКоманда пользователя:\n")
	sb.WriteString(input)
	sb.WriteString("\n\nВерни JSON-объект по схеме (см. response_format).")
	return sb.String()
}

// serializeTree produces "<id> [<kind>] <indent><name>" per line, sorted by
// (depth, parent, name) to give the LLM a stable, predictable view.
//
// If len(tree) > maxNodes, the deepest nodes are pruned first (we keep the
// top of the tree because high-level structure is what the user usually means).
func (b *promptBuilder) serializeTree(
	tree []port.TreeNode,
	rootHint valueobject.NodeID,
	maxNodes int,
) string {
	if len(tree) == 0 {
		return "(пусто)\n"
	}

	pruned := pruneByDepth(tree, maxNodes)

	// Stable sort: by depth asc, then name (case-insensitive).
	sort.SliceStable(pruned, func(i, j int) bool {
		if pruned[i].Depth != pruned[j].Depth {
			return pruned[i].Depth < pruned[j].Depth
		}
		return strings.ToLower(pruned[i].Name) < strings.ToLower(pruned[j].Name)
	})

	var sb strings.Builder
	for _, n := range pruned {
		marker := " "
		if !rootHint.IsZero() && n.ID.Equals(rootHint) {
			marker = "*"
		}
		// e.g. "* abc-123 [folder] /Документы"
		indent := strings.Repeat("  ", maxInt(n.Depth-1, 0))
		sb.WriteString(marker)
		sb.WriteString(" ")
		sb.WriteString(n.ID.String())
		sb.WriteString(" [")
		sb.WriteString(n.Kind)
		sb.WriteString("] ")
		sb.WriteString(indent)
		sb.WriteString(n.Name)
		sb.WriteString("\n")
	}
	return sb.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// pruneByDepth returns nodes preserving low-depth (top of tree) until maxNodes.
// If maxNodes ≤ 0 or len(tree) ≤ maxNodes, returns the input as-is.
func pruneByDepth(tree []port.TreeNode, maxNodes int) []port.TreeNode {
	if maxNodes <= 0 || len(tree) <= maxNodes {
		return tree
	}
	sorted := make([]port.TreeNode, len(tree))
	copy(sorted, tree)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Depth < sorted[j].Depth
	})
	return sorted[:maxNodes]
}

// buildJSONSchema returns the schema for the model's response.
//
// Yandex GPT supports `responseFormat: { jsonSchema: { schema: <schema> } }`.
// The shape we expect:
//
//	{ "ops": [ {kind, node_id, new_name?, new_parent_id?}, ... ], "explanation": "..." }
func (b *promptBuilder) buildJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"ops", "explanation"},
		"properties": map[string]any{
			"ops": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"kind", "node_id"},
					"properties": map[string]any{
						"kind": map[string]any{
							"type": "string",
							"enum": []string{"delete", "rename", "move"},
						},
						"node_id": map[string]any{
							"type": "string",
						},
						"new_name": map[string]any{
							"type": "string",
						},
						"new_parent_id": map[string]any{
							"type": "string",
						},
					},
				},
			},
			"explanation": map[string]any{
				"type": "string",
			},
		},
	}
}
