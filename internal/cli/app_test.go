package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddErrorsOnDuplicate(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env, "add", "github", "--command", "echo github")

	// second add without --update must fail
	var stderr bytes.Buffer
	err := Run([]string{"add", "github", "--command", "echo github"}, nil, &bytes.Buffer{}, &stderr)
	if err == nil {
		t.Fatalf("expected second add to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected 'already exists' error, got: %v", err)
	}

	// second add with --update must succeed and report updated=true
	secondAdd := runCLI(t, env, "--json", "add", "github", "--command", "echo github", "--update")
	addPayload := parseJSONResult(t, secondAdd)
	if updated, _ := addPayload.Data["updated"].(bool); !updated {
		t.Fatalf("expected --update add to report updated=true")
	}
}

func TestRemoveErrorsOnMissing(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")

	var stderr bytes.Buffer
	err := Run([]string{"remove", "nonexistent"}, nil, &bytes.Buffer{}, &stderr)
	if err == nil {
		t.Fatalf("expected remove of missing server to fail")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestEnableIsIdempotent(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env, "add", "github", "--command", "echo github")

	// enable twice: second call should be no-op at adapter level
	firstEnable := runCLI(t, env, "--json", "enable", "github", "--client", "cursor")
	firstPayload := parseJSONResult(t, firstEnable)
	secondEnable := runCLI(t, env, "--json", "enable", "github", "--client", "cursor")
	secondPayload := parseJSONResult(t, secondEnable)

	firstChanges := int(firstPayload.Data["changeCount"].(float64))
	secondChanges := int(secondPayload.Data["changeCount"].(float64))
	if firstChanges == 0 {
		t.Fatalf("expected first enable to produce changes")
	}
	if secondChanges != 0 {
		t.Fatalf("expected second enable to be idempotent with zero changes, got %d", secondChanges)
	}
}

func TestDryRunDoesNotWriteClientConfig(t *testing.T) {
	env := setupTestEnv(t)
	runCLI(t, env, "init")
	runCLI(t, env, "add", "github", "--command", "echo github")

	cursorPath := filepath.Join(env.home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(cursorPath), 0o755); err != nil {
		t.Fatalf("mkdir cursor dir: %v", err)
	}
	initial := []byte(`{"mcpServers":{"github":{"enabled":false}}}` + "\n")
	if err := os.WriteFile(cursorPath, initial, 0o644); err != nil {
		t.Fatalf("write cursor config: %v", err)
	}

	runCLI(t, env, "--dry-run", "enable", "github", "--client", "cursor")

	after, err := os.ReadFile(cursorPath)
	if err != nil {
		t.Fatalf("read cursor config: %v", err)
	}
	if string(after) != string(initial) {
		t.Fatalf("dry-run modified cursor config")
	}
}

func TestGoldenTextOutput(t *testing.T) {
	env := setupTestEnv(t)
	out := runCLI(t, env, "clients", "list")
	golden := mustRead(t, filepath.Join("testdata", "golden", "clients_list.txt"))
	if normalizeNewlines(out) != normalizeNewlines(string(golden)) {
		t.Fatalf("text output mismatch\nexpected:\n%s\ngot:\n%s", string(golden), out)
	}
}

func TestGoldenJSONOutput(t *testing.T) {
	env := setupTestEnv(t)
	out := runCLI(t, env, "--json", "clients", "list")
	golden := mustRead(t, filepath.Join("testdata", "golden", "clients_list.json"))
	if normalizeNewlines(out) != normalizeNewlines(string(golden)) {
		t.Fatalf("json output mismatch\nexpected:\n%s\ngot:\n%s", string(golden), out)
	}
}

type cliTestEnv struct {
	home       string
	configPath string
}

type jsonResult struct {
	Command string         `json:"command"`
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func setupTestEnv(t *testing.T) cliTestEnv {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	configPath := filepath.Join(root, "config.json")

	t.Setenv("HOME", home)
	t.Setenv("MCPUP_CONFIG", configPath)

	return cliTestEnv{
		home:       home,
		configPath: configPath,
	}
}

func runCLI(t *testing.T, env cliTestEnv, args ...string) string {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run(args, nil, &stdout, &stderr); err != nil {
		t.Fatalf("run failed for args %v: %v\nstderr=%s", args, err, stderr.String())
	}
	return stdout.String()
}

func parseJSONResult(t *testing.T, body string) jsonResult {
	t.Helper()
	var result jsonResult
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("parse json result: %v\nbody=%s", err, body)
	}
	return result
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
