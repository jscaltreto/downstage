package desktop

import (
	"fmt"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func BuildMenu(app *App, disabled map[string]bool) *menu.Menu {
	return buildMenuForGOOS(app, disabled, goruntime.GOOS)
}

func buildMenuForGOOS(app *App, disabled map[string]bool, goos string) *menu.Menu {
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

		appendGroupItems(app, submenu, g.items, disabled)
		root.AddSubmenu(g.name).Merge(submenu)
	}

	if goos != "linux" {
		root.Append(menu.WindowMenu())
	}

	return root
}

func appendGroupItems(app *App, parent *menu.Menu, items []Command, disabled map[string]bool) {
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

func emitCommand(app *App, id string) func(_ *menu.CallbackData) {
	return func(_ *menu.CallbackData) {
		runtime.EventsEmit(app.ctx, eventCommandExecute, id)
	}
}
