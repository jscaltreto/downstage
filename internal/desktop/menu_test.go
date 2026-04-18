package desktop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/menu"
)

// BuildMenu should produce a tree whose leaves correspond to the
// catalog's non-empty MenuPath entries. Walk the tree, collect IDs
// encountered via the Click callbacks that emitCommand closed over, and
// compare against the catalog.
//
// Because the Click callback is an opaque func, we verify existence by
// walking the tree and confirming the expected LABELS appear under the
// expected top-level groupings. That's sufficient coverage for
// "BuildMenu wired the catalog in correctly" without poking at
// unexported runtime internals.
func TestBuildMenu_PlacesCatalogItemsUnderCorrectTopLevels(t *testing.T) {
	app := NewApp()
	m := BuildMenu(app, nil)
	require.NotNil(t, m)

	// Build a map of top-level name → set of leaf labels present in that
	// submenu. EditMenu and WindowMenu roles add extra items we ignore.
	topLevels := map[string]map[string]bool{}
	for _, item := range m.Items {
		if item.Type != menu.SubmenuType || item.SubMenu == nil {
			continue
		}
		labels := map[string]bool{}
		collectLabels(item.SubMenu, labels)
		topLevels[item.Label] = labels
	}

	// Every catalog entry with a MenuPath must show up under its
	// declared top-level.
	for _, cmd := range Commands() {
		if len(cmd.MenuPath) == 0 {
			continue
		}
		top := cmd.MenuPath[0]
		labels, ok := topLevels[top]
		require.True(t, ok, "top-level menu %q missing", top)
		assert.True(t, labels[cmd.Label], "command %q (label %q) not under %q submenu", cmd.ID, cmd.Label, top)
	}
}

// SetDisabledCommands must produce a menu tree where the listed IDs
// have Disabled: true on their rendered item. Uses the same build path
// BuildMenu uses, so any real wiring break surfaces here.
func TestSetDisabledCommands_RebuildsMenuWithDisabledFlags(t *testing.T) {
	app := NewApp()
	initial := BuildMenu(app, nil)
	app.SetInitialMenu(initial)

	// Disable a couple of commands. Don't bother invoking the Wails
	// MenuUpdateApplicationMenu runtime hook (requires a live ctx); just
	// verify the rebuild produced the correct Disabled flags.
	require.NoError(t, app.SetDisabledCommands([]string{"file.exportPdf", "file.saveVersion"}))

	// Walk the rebuilt tree and find the matching labels under File.
	var file *menu.Menu
	for _, item := range app.currentMenu.Items {
		if item.Label == "File" && item.SubMenu != nil {
			file = item.SubMenu
			break
		}
	}
	require.NotNil(t, file, "File submenu missing after rebuild")

	disabledLabels := map[string]bool{}
	for _, item := range file.Items {
		if item.Type != menu.TextType {
			continue
		}
		if item.Disabled {
			disabledLabels[item.Label] = true
		}
	}
	assert.True(t, disabledLabels["Export PDF…"], "Export PDF should be Disabled")
	assert.True(t, disabledLabels["Save Version"], "Save Version should be Disabled")
}

// GetCommands returns the palette-facing projection. Should produce one
// entry per catalog command, in declaration order, with no Click
// callbacks or MenuPath leaking through.
func TestGetCommands_ReturnsFullCatalogProjection(t *testing.T) {
	app := NewApp()
	metas := app.GetCommands()
	assert.Equal(t, len(Commands()), len(metas), "GetCommands must mirror Commands length")

	// Spot-check a few known IDs.
	ids := map[string]CommandMeta{}
	for _, m := range metas {
		ids[m.ID] = m
	}
	require.Contains(t, ids, CmdFileNewPlay)
	require.Contains(t, ids, CmdFileSettingsSpellcheck)

	assert.Equal(t, "New Play", ids[CmdFileNewPlay].Label)
	// Palette-hidden command surfaces its flag so the palette can filter.
	assert.True(t, ids[CmdFileSettingsSpellcheck].PaletteHidden)
	assert.False(t, ids[CmdFileNewPlay].PaletteHidden)
}

func collectLabels(m *menu.Menu, out map[string]bool) {
	for _, item := range m.Items {
		if item.Type == menu.TextType && item.Label != "" {
			out[item.Label] = true
		}
		if item.SubMenu != nil {
			collectLabels(item.SubMenu, out)
		}
	}
}
