package desktop

import (
	"fmt"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// BuildMenu walks the command catalog and produces a native menu tree.
// Items with an empty MenuPath or a Platforms restriction that excludes
// the current GOOS are skipped. Top-level groups are visited in
// first-appearance order, as are items within each group.
//
// MenuPath of length 1 places the item directly under the top-level
// menu. MenuPath of length 2 nests the item under a sub-submenu named
// MenuPath[1]; the sub-submenu appears at the point where its first
// catalog entry falls.
//
// BeforeSeparator emits a separator line immediately before the current
// item (applied whether the item is top-level or nested).
//
// Native roles are slotted in by name:
//
//   - "Edit" is prefixed with menu.EditMenu() on macOS/Windows so undo/
//     redo/cut/copy/paste/select-all come from the OS. On Linux the
//     role crashes webkit2gtk, so explicit Linux-only catalog entries
//     (Platforms: ["linux"]) fill the same slots.
//   - "Window" renders as menu.WindowMenu() on macOS/Windows only.
//   - The macOS app menu (About/Services/Hide/Quit) is prepended via
//     menu.AppMenu() on darwin.
//
// Each non-separator leaf's Click callback emits
// eventCommandExecute with the command ID. The frontend's EventsOn
// subscriber dispatches from there.
//
// The `disabled` set, when non-nil, marks listed IDs as Disabled on
// the resulting menu. Used by SetDisabledCommands to rebuild the menu
// on state change.
func BuildMenu(app *App, disabled map[string]bool) *menu.Menu {
	return buildMenuForGOOS(app, disabled, goruntime.GOOS)
}

// buildMenuForGOOS is the testable core of BuildMenu. Pinning the GOOS
// argument lets unit tests verify the platform-filter + sub-submenu
// logic on any host OS without invoking the real Wails runtime.
func buildMenuForGOOS(app *App, disabled map[string]bool, goos string) *menu.Menu {
	root := menu.NewMenu()

	// AppMenu role renders the macOS "Downstage Write" top-level
	// (About / Services / Hide / Quit). It is NOT harmless on Linux
	// GTK — the empty placeholder it produces triggers
	// `gtk_menu_shell_insert: assertion 'GTK_IS_MENU_ITEM (child)'
	// failed` at runtime. Gate on GOOS so other platforms don't pay.
	if goos == "darwin" {
		root.Append(menu.AppMenu())
	}

	// Group commands by top-level menu name in first-appearance order so
	// the catalog's declaration order drives the bar's left-to-right
	// layout. Platform-filtered commands are dropped here so they don't
	// contribute to ordering (a group whose only item is filtered out
	// shouldn't appear at all).
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
		if !cmd.PlatformAllows(goos) {
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
		// AppMenu / WindowMenu) — on Linux the catalog's explicit
		// Linux-only entries fill the same slots.
		if g.name == "Edit" && goos != "linux" {
			submenu.Append(menu.EditMenu())
		}

		appendGroupItems(app, submenu, g.items, disabled)
		root.AddSubmenu(g.name).Merge(submenu)
	}

	// WindowMenu role (Minimize / Zoom / Bring-All-to-Front) is native on
	// macOS and Windows. On Linux GTK it also triggers the GTK_IS_MENU_ITEM
	// assertion, so gate it the same way as AppMenu.
	if goos != "linux" {
		root.Append(menu.WindowMenu())
	}

	return root
}

// appendGroupItems renders a top-level group's items into `parent`.
// Items whose MenuPath has a second segment get collected into a named
// sub-submenu that's emitted at the position of its first appearance.
func appendGroupItems(app *App, parent *menu.Menu, items []Command, disabled map[string]bool) {
	// Per-sub-submenu item buckets, keyed by MenuPath[1]. The entries
	// slice below carries the sequencing: each "leaf" entry points at
	// an index in items; each "submenu" entry points at a key whose
	// bucket will be drained when the entry is processed.
	type entry struct {
		kind       string // "leaf" | "submenu"
		cmdIdx     int
		submenuKey string
	}
	entries := []entry{}
	buckets := map[string][]Command{}
	bucketSeen := map[string]bool{}

	for i, cmd := range items {
		if len(cmd.MenuPath) >= 2 {
			key := cmd.MenuPath[1]
			buckets[key] = append(buckets[key], cmd)
			if !bucketSeen[key] {
				entries = append(entries, entry{kind: "submenu", submenuKey: key})
				bucketSeen[key] = true
			}
			continue
		}
		entries = append(entries, entry{kind: "leaf", cmdIdx: i})
	}

	for _, e := range entries {
		if e.kind == "leaf" {
			addLeaf(app, parent, items[e.cmdIdx], disabled)
			continue
		}
		nested := menu.NewMenu()
		for _, cmd := range buckets[e.submenuKey] {
			addLeaf(app, nested, cmd, disabled)
		}
		parent.AddSubmenu(e.submenuKey).Merge(nested)
	}
}

// addLeaf emits one catalog entry as a text menu item, inserting a
// leading separator if BeforeSeparator is set and applying the disabled
// flag from the caller's set.
func addLeaf(app *App, parent *menu.Menu, cmd Command, disabled map[string]bool) {
	if cmd.BeforeSeparator {
		parent.AddSeparator()
	}
	var accel *keys.Accelerator
	if cmd.Accelerator != "" {
		parsed, err := keys.Parse(cmd.Accelerator)
		if err != nil {
			panic(fmt.Sprintf("desktop: invalid accelerator %q on %s: %v", cmd.Accelerator, cmd.ID, err))
		}
		accel = parsed
	}
	item := parent.AddText(cmd.Label, accel, emitCommand(app, cmd.ID))
	if disabled[cmd.ID] {
		item.Disabled = true
	}
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
