//go:build unit

package examples_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/examples"
)

func TestExample(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		t.Run("lists the examples that validate, in the order of their names", func(t *testing.T) {
			require.Equal(t, []string{"all_patterns", "dcb_model", "specs_hotel"}, namesOf(examples.All()))
		})

		t.Run("gives each example the text of its file", func(t *testing.T) {
			for _, example := range examples.All() {
				source, err := os.ReadFile(example.Name + ".emod")
				require.NoError(t, err)

				require.Equal(t, string(source), example.Source, "example %s", example.Name)
			}
		})

		t.Run("lists each .emod file of the folder that is not authored to fail, so that a new example needs a description", func(t *testing.T) {
			entries, err := os.ReadDir(".")
			require.NoError(t, err)
			var validating []string
			for _, entry := range entries {
				name := entry.Name()
				if filepath.Ext(name) == ".emod" && !strings.HasSuffix(name, "_test.emod") {
					validating = append(validating, strings.TrimSuffix(name, ".emod"))
				}
			}

			require.Equal(t, validating, namesOf(examples.All()))
		})

		t.Run("gives each example a description", func(t *testing.T) {
			for _, example := range examples.All() {
				require.NotEmpty(t, example.Description, "example %s", example.Name)
			}
		})

		t.Run("describes what each example shows", func(t *testing.T) {
			descriptions := map[string]string{}
			for _, example := range examples.All() {
				descriptions[example.Name] = example.Description
			}

			require.Contains(t, descriptions["all_patterns"], "four slice patterns")
			require.Contains(t, descriptions["dcb_model"], "decides_on")
			require.Contains(t, descriptions["specs_hotel"], "Given-When-Then specs")
		})
	})
}

func namesOf(all []examples.Example) []string {
	names := make([]string, 0, len(all))
	for _, example := range all {
		names = append(names, example.Name)
	}
	return names
}
