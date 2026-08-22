package feedback

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// configDir resolves the nn config directory using the same order as the
// gitlocal backend's defaultNNConfigDir: NN_CONFIG_DIR, then XDG_CONFIG_HOME/nn,
// then ~/.config/nn.
func configDir() string {
	if d := os.Getenv("NN_CONFIG_DIR"); d != "" {
		return d
	}
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, _ := os.UserHomeDir()
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "nn")
}

// SessionDir returns the directory holding a feedback session's files:
// <configdir>/feedback/<id>/.
func SessionDir(id string) string {
	return filepath.Join(configDir(), "feedback", id)
}

// Artifact is one output produced by a feedback surface, referenced by path.
// The format names how to interpret the file; the surface writes it natively.
type Artifact struct {
	Format string `json:"format"`
	Path   string `json:"path"`
}

// FeedbackRequest is prepared before a session launches. It describes what the
// human is being asked to do; the surface renders it and never infers intent.
// An absent Workspace means bootstrap/create.
type FeedbackRequest struct {
	ID           string   `json:"id"`
	Surface      string   `json:"surface"`
	Mode         string   `json:"mode"`
	Instructions string   `json:"instructions"`
	Context      []string `json:"context"`
	Workspace    string   `json:"workspace,omitempty"`
	// Mermaid is an optional diagram source used to seed the canvas surface:
	// the frontend converts it to editable Excalidraw elements as initialData
	// when no prior draft exists.
	Mermaid string `json:"mermaid,omitempty"`
	Output  string `json:"output"`
	// Focus is the ego note id for the graph surface: the neighborhood the human
	// is asked to react to. AllowedNodes is the resolved scope — the exact set of
	// note ids the surface may show. The agent supplies the scope; the server
	// never widens it, so what the human sees is bounded by what the agent chose.
	Focus        string   `json:"focus,omitempty"`
	AllowedNodes []string `json:"allowed_nodes,omitempty"`
}

// FeedbackResult is the thin envelope returned after submission. Surface-specific
// shape lives inside the referenced artifact files, not in this struct.
type FeedbackResult struct {
	ID        string     `json:"id"`
	Surface   string     `json:"surface"`
	Status    string     `json:"status"`
	Artifacts []Artifact `json:"artifacts"`
}

const (
	requestFile = "request.json"
	resultFile  = "result.json"
)

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// WriteRequest persists q to <dir>/request.json.
func WriteRequest(dir string, q FeedbackRequest) error {
	return writeJSON(filepath.Join(dir, requestFile), q)
}

// ReadRequest reads <dir>/request.json.
func ReadRequest(dir string) (FeedbackRequest, error) {
	var q FeedbackRequest
	err := readJSON(filepath.Join(dir, requestFile), &q)
	return q, err
}

// WriteResult persists r to <dir>/result.json.
func WriteResult(dir string, r FeedbackResult) error {
	return writeJSON(filepath.Join(dir, resultFile), r)
}

// ReadResult reads <dir>/result.json.
func ReadResult(dir string) (FeedbackResult, error) {
	var r FeedbackResult
	err := readJSON(filepath.Join(dir, resultFile), &r)
	return r, err
}
