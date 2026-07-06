package recipe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListRejectsExecutableRecipeFields(t *testing.T) {
	dir := t.TempDir()
	writeRecipeTestFile(t, dir, ".nucleus/recipes/gorm.yaml", `schema_version: "recipe.v1"
id: gorm-sql
kind: sql
provider: gorm
language: go
suggest:
  interfaces:
    - keep ORM behind a project-owned interface
  verification:
    - go test ./...
risks:
  - transaction boundary must be explicit
`)
	writeRecipeTestFile(t, dir, ".nucleus/recipes/unsafe.yaml", `schema_version: "recipe.v1"
id: unsafe
kind: sql
language: go
commands:
  - rm -rf .
`)

	output, err := List(dir, Filter{Kind: "sql"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if output.OK {
		t.Fatalf("OK = true, want false because unsafe recipe failed validation")
	}
	if !containsRecipe(output.Recipes, "gorm-sql", recipeSourceProject) {
		t.Fatalf("recipes = %#v, want safe project gorm recipe", output.Recipes)
	}
	if !containsRecipe(output.Recipes, "sql-port-boundary", recipeSourceBuiltin) {
		t.Fatalf("recipes = %#v, want built-in sql recipe", output.Recipes)
	}
	if !output.Diagnostics.Failed() {
		t.Fatalf("diagnostics = %#v, want failed diagnostics", output.Diagnostics)
	}
}

func TestCandidatesAreReadOnlyHints(t *testing.T) {
	dir := t.TempDir()
	writeRecipeTestFile(t, dir, ".nucleus/recipes/gorm.yaml", `schema_version: "recipe.v1"
id: gorm-sql
kind: sql
provider: gorm
language: go
detect:
  files:
    - storage/db.go
suggest:
  interfaces:
    - expose storage through an interface
  verification:
    - go test ./storage
risks:
  - avoid leaking ORM models into API contracts
`)
	writeRecipeTestFile(t, dir, "storage/db.go", "package storage\n")

	output := Candidates(dir, CandidateQuery{Task: "add mysql capability", Kinds: []string{"sql"}})
	if !output.OK {
		t.Fatalf("OK = false: %#v", output.Diagnostics)
	}
	if len(output.Candidates) != 2 {
		t.Fatalf("candidates = %#v, want project and built-in candidates", output.Candidates)
	}
	candidate := findCandidate(t, output.Candidates, "gorm-sql")
	if candidate.Selection != "candidate_only" || !candidate.DecisionRequired {
		t.Fatalf("candidate should remain decision-only hint: %#v", candidate)
	}
	if candidate.ID != "gorm-sql" || candidate.Provider != "gorm" || candidate.Source != recipeSourceProject {
		t.Fatalf("candidate = %#v, want gorm metadata as hint", candidate)
	}
	if len(candidate.SuggestedVerification) != 1 || candidate.SuggestedVerification[0] != "go test ./storage" {
		t.Fatalf("suggested verification = %#v", candidate.SuggestedVerification)
	}
}

func TestCandidatesIgnoreMissingRecipeDir(t *testing.T) {
	output := Candidates(t.TempDir(), CandidateQuery{Task: "add cache capability", Kinds: []string{"redis"}})
	if !output.OK {
		t.Fatalf("OK = false: %#v", output.Diagnostics)
	}
	if len(output.Candidates) != 0 || len(output.Diagnostics) != 0 {
		t.Fatalf("output = %#v, want empty candidate set without missing-dir warning", output)
	}
}

func TestCandidatesIncludeBuiltinReadOnlyRecipes(t *testing.T) {
	output := Candidates(t.TempDir(), CandidateQuery{Task: "add mysql capability", Kinds: []string{"sql"}})
	if !output.OK {
		t.Fatalf("OK = false: %#v", output.Diagnostics)
	}
	if len(output.Candidates) != 1 {
		t.Fatalf("candidates = %#v, want one built-in candidate", output.Candidates)
	}
	candidate := output.Candidates[0]
	if candidate.ID != "sql-port-boundary" || candidate.Source != recipeSourceBuiltin {
		t.Fatalf("candidate = %#v, want built-in sql-port-boundary", candidate)
	}
	if candidate.Provider != "" {
		t.Fatalf("candidate provider = %q, want provider-neutral built-in", candidate.Provider)
	}
	if candidate.Path != "builtin://recipes/sql-port-boundary.yaml" {
		t.Fatalf("candidate path = %q, want built-in URI", candidate.Path)
	}
	if candidate.Selection != "candidate_only" || !candidate.DecisionRequired {
		t.Fatalf("candidate should remain read-only decision hint: %#v", candidate)
	}
}

func TestProjectRecipeOverridesBuiltinByID(t *testing.T) {
	dir := t.TempDir()
	writeRecipeTestFile(t, dir, ".nucleus/recipes/sql.yaml", `schema_version: "recipe.v1"
id: sql-port-boundary
kind: sql
provider: team-sql
language: go
suggest:
  verification:
    - go test ./internal/storage
`)

	output := Candidates(dir, CandidateQuery{Task: "add mysql capability", Kinds: []string{"sql"}})
	if !output.OK {
		t.Fatalf("OK = false: %#v", output.Diagnostics)
	}
	if len(output.Candidates) != 1 {
		t.Fatalf("candidates = %#v, want local override only", output.Candidates)
	}
	candidate := output.Candidates[0]
	if candidate.ID != "sql-port-boundary" || candidate.Source != recipeSourceProject || candidate.Provider != "team-sql" {
		t.Fatalf("candidate = %#v, want project override", candidate)
	}
}

func writeRecipeTestFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsRecipe(values []Summary, id string, source string) bool {
	for _, item := range values {
		if item.ID == id && item.Source == source {
			return true
		}
	}
	return false
}

func findCandidate(t *testing.T, values []Candidate, id string) Candidate {
	t.Helper()
	for _, item := range values {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("candidate %q not found in %#v", id, values)
	return Candidate{}
}
