package desktop

import (
	"fmt"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// BuildMenu walks the command catalog and produces a native menu tree.
// Items with an empty MenuPath are excluded (palette-only / programmatic).
// Top-level groups are visited in first-appearance order, as are items
// within each group. BeforeSeparator emits a separator line before the
// current item. Accelerator strings are parsed through Wails' own
// parser; a bad string panics at startup rather than silently losing a
// shortcut.
//
// Native roles are slotted in by name:
//
//   - "Edit" is prefixed with menu.EditMenu() so undo/redo/cut/copy/paste
//     come from the OS, followed by our catalog entries.
//   - "Window" is rendered entirely as menu.WindowMenu() (the catalog
//     has no Window entries; the role owns it).
//   - The macOS app menu (About/Services/Hide/Quit) is prepended via
//     menu.AppMenu() so macOS users get the native "Downstage Write"
//     top-level.
//
// Each non-separator leaf's Click callback emits
// eventCommandExecute with the command ID. The frontend's EventsOn
// subscriber dispatches from there.
//
// The `disabled` set, when non-nil, marks listed IDs as Disabled on
// the resulting menu. Used by SetDisabledCommands to rebuild the menu
// on state change.
func BuildMenu(app *App, disabled map[string]bool) *menu.Menu {
	root := menu.NewMenu()

	// AppMenu role renders the macOS "Downstage Write" top-level
	// (About / Services / Hide / Quit). It is NOT harmless on Linux
	// GTK — the empty placeholder it produces triggers
	// `gtk_menu_shell_insert: assertion 'GTK_IS_MENU_ITEM (child)'
	// failed` at runtime. Gate on GOOS so other platforms don't pay.
	if goruntime.GOOS == "darwin" {
		root.Append(menu.AppMenu())
	}

	// Group commands by top-level menu name in first-appearance order so
	// the catalog's declaration order drives the bar's left-to-right
	// layout.
	type group struct {
		name  string
		items []Command
	}
	var groups []group
	groupIdx := map[string]int{}
	for _, cmd := range Commands() {
		if len(cmd.MenuPath) == 0 {
			continue
		}
		top := cmd.MenuPath[0]
		if idx, ok := groupIdx[top]; ok {
			groups[idx].items = append(groups[idx].items, cmd)
		} else {
			groupIdx[top] = len(groups)
			groups = append(groups, group{name: top, items: []Command{cmd}})
		}
	}

	for _, g := range groups {
		submenu := menu.NewMenu()

		// Edit gets the native cut/copy/paste/undo/redo prefix on
		// platforms where the role renders cleanly. Linux GTK chokes on
		// the role placeholder widget (same GTK_IS_MENU_ITEM assertion as
		// AppMenu / WindowMenu) — on Linux the CodeMirror defaults cover
		// undo/redo/cut/copy/paste anyway.
		if g.name == "Edit" && goruntime.GOOS != "linux" {
			submenu.Append(menu.EditMenu())
		}

		for _, cmd := range g.items {
			if cmd.BeforeSeparator {
				submenu.AddSeparator()
			}
			var accel *keys.Accelerator
			if cmd.Accelerator != "" {
				parsed, err := keys.Parse(cmd.Accelerator)
				if err != nil {
					panic(fmt.Sprintf("desktop: invalid accelerator %q on %s: %v", cmd.Accelerator, cmd.ID, err))
				}
				accel = parsed
			}
			item := submenu.AddText(cmd.Label, accel, emitCommand(app, cmd.ID))
			if disabled[cmd.ID] {
				item.Disabled = true
			}
		}

		root.AddSubmenu(g.name).Merge(submenu)
	}

	// WindowMenu role (Minimize / Zoom / Bring-All-to-Front) is native on
	// macOS and Windows. On Linux GTK it also triggers the GTK_IS_MENU_ITEM
	// assertion, so gate it the same way as AppMenu.
	if goruntime.GOOS != "linux" {
		root.Append(menu.WindowMenu())
	}

	return root
}

// emitCommand builds a menu click callback that publishes the command
// ID on the runtime event bus. A single frontend subscriber
// (EventsOn("command:execute")) dispatches by ID through the TS
// command dispatcher.
func emitCommand(app *App, id string) func(_ *menu.CallbackData) {
	return func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, eventCommandExecute, id)
	}
}
