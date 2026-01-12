package knowledge

import (
	"fmt"
	"strings"

	"github.com/thinktwice/agentForge/src/core"
	"github.com/thinktwice/agentForge/src/llms"
)

func NewSearchTool(kp *KnowledgePlugin) llms.Tool {
	desc := `Search for semantic information in the documents in the documents
	Documents identifier: %s.
	Doument Paths: %s.

	This tool uses semantic search on a previously indexed documents.
	`
	desc = fmt.Sprintf(desc, kp.knowledgeIdentifier, strings.Join(kp.documentPaths, ", "))
	return &core.Tool{
		Name:        "semantic-search-tool",
		Description: desc,
		AdvanceDesc: `Advanced Details:
		`,
		Parameters: []core.Parameter{
			{
				Name:        "query",
				Type:        "string",
				Description: "The query to search the knowledge base for",
				Required:    true,
			},
			{
				Name:        "limit",
				Type:        "int",
				Description: "The maximum number of results to return",
				Required:    false,
			},
		},
		Handler: func(agentContext map[string]any, args map[string]any) llms.ToolReturn {
			query := args["query"].(string)
			limit := args["limit"].(int)
			documentChunks, err := kp.search(query, limit)
			if err != nil {
				return core.NewErrorResponse(err.Error())
			}

			textResults := make([]string, len(documentChunks))
			for i, chunk := range documentChunks {
				textResults[i] = chunk.Content
			}

			// Join the text results into a single string
			results := strings.Join(textResults, "\n")

			return core.NewSuccessResponse(results)
		},
	}
}
