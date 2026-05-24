package desktop

import (
	"fmt"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// CommandEmitter is what each menu item's Click callback invokes to
// notify the frontend that a command fired. Extracted so tests can
// drop in a capture-emitter and verify which IDs the click closures
// reach without standing up a Wails runtime.
type CommandEmitter func(id string)

func BuildMenu(app *App, disabled map[string]bool) *menu.Menu {
	return buildMenuForGOOS(app, disabled, goruntime.GOOS, defaultEmitter(app))
}

// defaultEmitter is the production emitter — fires a Wails event
// that the frontend's dispatcher-registry listens on.
func defaultEmitter(app *App) CommandEmitter {
	return func(id string) {
		runtime.EventsEmit(app.ctx, eventCommandExecute, id)
	}
}

func buildMenuForGOOS(app *App, disabled map[string]bool, goos string, emit CommandEmitter) *menu.Menu {
	root := menu.NewMenu()

	if goos == "darwin" {
		root.Append(menu.AppMenu())
	}

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

		if g.name == "Edit" && goos != "linux" {
			submenu.Append(menu.EditMenu())
		}

		appendGroupItems(submenu, g.items, disabled, emit)
		root.AddSubmenu(g.name).Merge(submenu)
	}

	if goos != "linux" {
		root.Append(menu.WindowMenu())
	}

	return root
}

func appendGroupItems(parent *menu.Menu, items []Command, disabled map[string]bool, emit CommandEmitter) {
	type entry struct {
		kind       string
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
			addLeaf(parent, items[e.cmdIdx], disabled, emit)
			continue
		}
		nested := menu.NewMenu()
		for _, cmd := range buckets[e.submenuKey] {
			addLeaf(nested, cmd, disabled, emit)
		}
		parent.AddSubmenu(e.submenuKey).Merge(nested)
	}
}

func addLeaf(parent *menu.Menu, cmd Command, disabled map[string]bool, emit CommandEmitter) {
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
	id := cmd.ID
	item := parent.AddText(cmd.Label, accel, func(_ *menu.CallbackData) {
		emit(id)
	})
	if disabled[cmd.ID] {
		item.Disabled = true
	}
}
