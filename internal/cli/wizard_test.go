package cli

import (
	"os"
	"testing"
)

func TestLoadConfigOrDefaultErrorsOnInvalidConfig(t *testing.T) {
	env := setupTestEnv(t)

	if err := os.WriteFile(env.configPath, []byte(`{"bad":true}`), 0o644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	if _, err := loadConfigOrDefault(); err == nil {
		t.Fatalf("expected invalid config to return error")
	}
}
