package desktop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

func TestCommands_UniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Commands() {
		require.False(t, seen[c.ID], "duplicate command ID: %s", c.ID)
		seen[c.ID] = true
	}
}

func TestCommands_AcceleratorsParse(t *testing.T) {
	for _, c := range Commands() {
		if c.Accelerator == "" {
			continue
		}
		_, err := keys.Parse(c.Accelerator)
		assert.NoError(t, err, "command %s has invalid accelerator %q", c.ID, c.Accelerator)
	}
}

func TestCommands_LabelsPresent(t *testing.T) {
	for _, c := range Commands() {
		assert.NotEmpty(t, c.Label, "command %s has an empty label", c.ID)
	}
}

func TestCommands_IncludesAboutAndDockToggle(t *testing.T) {
	ids := map[string]Command{}
	for _, c := range Commands() {
		ids[c.ID] = c
	}

	about, ok := ids[CmdHelpAbout]
	require.True(t, ok, "help.about must be in the catalog")
	assert.Equal(t, "About Downstage Write…", about.Label)
	assert.Equal(t, []string{"Help"}, about.MenuPath)
}
