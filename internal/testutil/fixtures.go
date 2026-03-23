package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func ReadFixture(t *testing.T, relativePath string) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve caller path")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	content, err := os.ReadFile(filepath.Join(root, relativePath))
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", relativePath, err)
	}

	return string(content)
}
