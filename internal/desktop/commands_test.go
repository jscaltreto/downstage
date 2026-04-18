package desktop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

// The catalog is the single source of truth for command IDs. A
// duplicate ID would silently overwrite a handler (since the frontend
// keys by ID) — fail loudly at test time instead.
func TestCommands_UniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Commands() {
		require.False(t, seen[c.ID], "duplicate command ID: %s", c.ID)
		seen[c.ID] = true
	}
}

// Every accelerator string must parse through the same parser Wails
// uses at startup. A bad string would panic in BuildMenu; catch it here
// so a typo is a test failure, not a runtime crash.
func TestCommands_AcceleratorsParse(t *testing.T) {
	for _, c := range Commands() {
		if c.Accelerator == "" {
			continue
		}
		_, err := keys.Parse(c.Accelerator)
		assert.NoError(t, err, "command %s has invalid accelerator %q", c.ID, c.Accelerator)
	}
}

// Every command with a MenuPath should have a non-empty Label;
// palette-hidden commands without a MenuPath are also allowed. A
// zero-label menu entry renders as a blank gap.
func TestCommands_LabelsPresent(t *testing.T) {
	for _, c := range Commands() {
		assert.NotEmpty(t, c.Label, "command %s has an empty label", c.ID)
	}
}

// New commands in the desktop polish batch must be reachable from the
// palette so the frontend can register a handler for them.
func TestCommands_IncludesAboutAndDockToggle(t *testing.T) {
	ids := map[string]Command{}
	for _, c := range Commands() {
		ids[c.ID] = c
	}

	about, ok := ids[CmdHelpAbout]
	require.True(t, ok, "help.about must be in the catalog")
	assert.Equal(t, "About Downstage Write…", about.Label)
	assert.Equal(t, []string{"Help"}, about.MenuPath)

	dock, ok := ids[CmdViewToggleDrawerDock]
	require.True(t, ok, "view.toggleDrawerDock must be in the catalog")
	assert.Equal(t, []string{"View"}, dock.MenuPath)
}
