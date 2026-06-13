package inspect

import (
	"testing"

	"github.com/nucleuskit/contract/errors"
	"github.com/nucleuskit/contract/openapi"
)

func TestBuildFlowGraphFromContracts(t *testing.T) {
	graph := BuildFlowGraphFromContracts(
		[]openapi.Route{{
			Method:              "GET",
			Path:                "/widgets/{id}",
			OperationID:         "get_widget",
			RequestBodyRequired: true,
			Parameters: []openapi.Parameter{{
				Name:       "id",
				In:         "path",
				Required:   true,
				SchemaType: "string",
			}},
		}},
		[]errors.Code{{Code: 4001, Message: "invalid widget", HTTPStatus: 400}},
	)

	if got := graph.SchemaVersion; got != "1.0" {
		t.Fatalf("SchemaVersion = %q, want 1.0", got)
	}
	if len(graph.Nodes) != 3 {
		t.Fatalf("len(Nodes) = %d, want 3", len(graph.Nodes))
	}
	if len(graph.Edges) != 2 {
		t.Fatalf("len(Edges) = %d, want 2", len(graph.Edges))
	}
	if len(graph.Params) != 2 {
		t.Fatalf("len(Params) = %d, want 2", len(graph.Params))
	}
	if got := graph.Params[0].Name; got != "path.id" {
		t.Fatalf("first param = %q, want path.id", got)
	}
	if len(graph.ErrorPaths) != 1 {
		t.Fatalf("len(ErrorPaths) = %d, want 1", len(graph.ErrorPaths))
	}
	if got := graph.ErrorPaths[0].Code; got != 4001 {
		t.Fatalf("error code = %d, want 4001", got)
	}
}
