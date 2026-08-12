package dto

const (
	ResponsesToolKindFunction   = "function"
	ResponsesToolKindNamespace  = "namespace"
	ResponsesToolKindCustom     = "custom"
	ResponsesToolKindToolSearch = "tool_search"
)

// ResponsesToolMapping records how a Responses tool was represented on a
// Chat Completions-only upstream so the response can be restored losslessly.
type ResponsesToolMapping struct {
	Kind      string
	Name      string
	Namespace string
}
