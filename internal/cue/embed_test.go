//go:build unit

package cue

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchema(t *testing.T) {
	t.Run("is non-empty and contains Model definition", func(t *testing.T) {
		require.NotEmpty(t, Schema)
		require.Contains(t, Schema, "#Model:")
	})

	t.Run("covers all model elements", func(t *testing.T) {
		definitions := []string{
			"#Model:",
			"#Actor:",
			"#Context:",
			"#Aggregate:",
			"#Slice:",
			"#Command:",
			"#Event:",
			"#Field:",
			"#Flow:",
			"#Trigger:",
			"#View:",
			"#Automation:",
			"#Translation:",
		}
		for _, def := range definitions {
			require.Contains(t, Schema, def, "schema should define %s", def)
		}
	})

	t.Run("has no external CUE imports", func(t *testing.T) {
		lines := strings.Split(Schema, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, `import "`) {
				t.Errorf("schema should not have external imports, found: %s", line)
			}
		}
	})

	t.Run("is valid CUE syntax", func(t *testing.T) {
		if _, err := exec.LookPath("cue"); err != nil {
			t.Skip("cue tool not available in PATH")
		}

		schemaPath := t.TempDir() + "/schema.cue"
		err := os.WriteFile(schemaPath, []byte(Schema), 0644)
		require.NoError(t, err)

		cmd := exec.Command("cue", "vet", schemaPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cue vet failed: %v\nOutput: %s", err, output)
		}
	})
}
