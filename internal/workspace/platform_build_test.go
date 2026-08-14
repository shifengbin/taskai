package workspace

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDarwinWorkspacePackageDoesNotUseCgo(t *testing.T) {
	command := exec.Command("go", "list", "-f", `{{join .CgoFiles "\n"}}`, ".")
	command.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=amd64", "CGO_ENABLED=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list darwin workspace error = %v, output = %s", err, output)
	}
	if cgoFiles := strings.TrimSpace(string(output)); cgoFiles != "" {
		t.Fatalf("darwin workspace CgoFiles = %q, want empty", cgoFiles)
	}
}
