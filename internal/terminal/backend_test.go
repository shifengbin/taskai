package terminal

import (
	"strings"
	"testing"
)

func TestEmbeddedTerminalEnvironmentDoesNotDisableColorOutput(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	environment := embeddedTerminalEnvironment(nil)
	for _, entry := range environment {
		if strings.HasPrefix(entry, "NO_COLOR=") {
			t.Fatal("embeddedTerminalEnvironment() contains NO_COLOR")
		}
	}
}
