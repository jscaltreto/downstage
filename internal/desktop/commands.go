package desktop

// The command catalog is the single source of truth for app-level command
// metadata: ID, label, menu path, accelerator, and palette visibility.
// menu.go consumes it to build the native menu; the frontend fetches the
// palette-facing projection via GetCommands() and registers handlers by
// ID. Adding a command is one entry here plus one handler in the
// frontend's commands.ts — nothing else.
//
// Category is the palette grouping (for display only; the menu structure
// is driven by MenuPath).
//
// MenuPath places the item in the native menu. MenuPath[0] is the
// top-level menu label. A non-leaf MenuPath[1...] nests into submenus.
// A nil or empty MenuPath means "not on the menu" — typically used for
// palette-only or programmatically-dispatched commands.
//
// BeforeSeparator inserts a separator line in the menu immediately
// before this item. Used for visual grouping within a top-level menu.
//
// Accelerator is a Wails accelerator string like "cmdorctrl+n" parsed
// by github.com/wailsapp/wails/v2/pkg/menu/keys. Empty means no
// keyboard shortcut.

type CommandCategory string

const (
	CategoryFile     CommandCategory = "file"
	CategoryEdit     CommandCategory = "edit"
	CategoryView     CommandCategory = "view"
	CategoryNavigate CommandCategory = "navigate"
	CategoryInsert   CommandCategory = "insert"
	CategoryFormat   CommandCategory = "format"
	CategoryHelp     CommandCategory = "help"
)

// Command is the full catalog entry used by menu.go. Not exposed to the
// frontend verbatim — GetCommands returns CommandMeta instead.
type Command struct {
	ID          string
	Label       string
	Category    CommandCategory
	Accelerator string
	// MenuPath places the item in the native menu. MenuPath[0] is the
	// top-level menu label; MenuPath[1] (optional) nests into a submenu
	// by that name. A nil/empty MenuPath means "not on the menu" —
	// typically used for palette-only or programmatically-dispatched
	// commands.
	MenuPath        []string
	BeforeSeparator bool
	// PaletteVisible defaults to true; set false to hide from the palette
	// (e.g. programmatic commands that shouldn't appear as a user-selectable
	// entry).
	PaletteHidden bool
	// Platforms restricts this command's menu AND palette visibility to
	// the listed GOOS values. Empty = all platforms. Used when a native
	// menu role (macOS EditMenu, AppMenu's Quit) covers the same ground
	// on some OS but the rest need manual catalog entries.
	Platforms []string
}

// CommandMeta is the palette-facing projection of a Command. Labels,
// categories, and accelerators come from Go; TS owns handlers.
type CommandMeta struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Category    string `json:"category"`
	Accelerator string `json:"accelerator,omitempty"`
	// True when this command should not be shown in the palette (it's
	// menu-only or programmatic).
	PaletteHidden bool `json:"paletteHidden,omitempty"`
}

// Command IDs. Exported as constants so menu.go, tests, and anyone
// grep-navigating the codebase can reach them without reopening this file.
const (
	CmdFileNewPlay            = "file.newPlay"
	CmdFileOpen               = "file.open"
	CmdFileSaveVersion        = "file.saveVersion"
	CmdFileExportPDF          = "file.exportPdf"
	CmdFileExportDs           = "file.exportDs"
	CmdFileSettings           = "file.settings"
	CmdFileSettingsSpellcheck = "file.settings.spellcheck"
	CmdFileQuit               = "file.quit"

	// Edit basics — Undo/Redo/Cut/Copy/Paste/Select All. On macOS and
	// Windows the native EditMenu role already provides these, so these
	// catalog entries are Linux-only. See commands.go's Platforms field.
	CmdEditUndo      = "edit.undo"
	CmdEditRedo      = "edit.redo"
	CmdEditCut       = "edit.cut"
	CmdEditCopy      = "edit.copy"
	CmdEditPaste     = "edit.paste"
	CmdEditSelectAll = "edit.selectAll"

	CmdEditFind        = "edit.find"
	CmdEditFindReplace = "edit.findReplace"
	CmdEditCopyAll     = "edit.copyAll"

	CmdViewCommandPalette = "view.commandPalette"
	CmdViewTogglePreview  = "view.togglePreview"
	CmdViewToggleSidebar  = "view.toggleSidebar"
	CmdViewToggleIssues   = "view.toggleIssues"
	CmdViewToggleOutline  = "view.toggleOutline"
	CmdViewToggleStats    = "view.toggleStats"

	CmdNavigateNextFile = "navigate.nextFile"
	CmdNavigatePrevFile = "navigate.prevFile"
	CmdNavigateGoToFile = "navigate.goToFile"

	CmdFormatBold          = "format.bold"
	CmdFormatItalic        = "format.italic"
	CmdFormatUnderline     = "format.underline"
	CmdFormatStrikethrough = "format.strikethrough"

	CmdInsertCue       = "insert.cue"
	CmdInsertDirection = "insert.direction"
	CmdInsertAct       = "insert.act"
	CmdInsertScene     = "insert.scene"
	CmdInsertSong      = "insert.song"
	CmdInsertPageBreak = "insert.pageBreak"

	CmdLibraryNewFolder = "library.newFolder"

	CmdHelpToggle = "help.toggle"
	CmdHelpGitHub = "help.github"
	CmdHelpDocs   = "help.docs"
	CmdHelpAbout  = "help.about"
)

var (
	linuxOnly       = []string{"linux"}
	nonMacPlatforms = []string{"linux", "windows"}
)

// Commands returns the canonical ordered catalog. The order governs the
// palette's default sort and the order items appear within their menu
// submenus. Keep related commands grouped and use BeforeSeparator for
// visual breaks.
func Commands() []Command {
	return []Command{
		// File
		{ID: CmdFileNewPlay, Label: "New Play", Category: CategoryFile, Accelerator: "cmdorctrl+n", MenuPath: []string{"File"}},
		{ID: CmdFileOpen, Label: "Open…", Category: CategoryFile, Accelerator: "cmdorctrl+o", MenuPath: []string{"File"}},
		{ID: CmdFileSaveVersion, Label: "Save Version", Category: CategoryFile, Accelerator: "cmdorctrl+s", MenuPath: []string{"File"}, BeforeSeparator: true},
		// Export submenu: PDF + raw .ds file save-as. Labels mirror common
		// editor conventions (File > Export > ...).
		{ID: CmdFileExportPDF, Label: "PDF…", Category: CategoryFile, Accelerator: "cmdorctrl+e", MenuPath: []string{"File", "Export"}},
		{ID: CmdFileExportDs, Label: "Downstage File…", Category: CategoryFile, MenuPath: []string{"File", "Export"}},
		{ID: CmdFileSettings, Label: "Settings…", Category: CategoryFile, Accelerator: "cmdorctrl+,", MenuPath: []string{"File"}, BeforeSeparator: true},
		// Palette-hidden programmatic command used by the in-editor
		// SpellCheck toolbar button to open Settings on the Spellcheck tab.
		{ID: CmdFileSettingsSpellcheck, Label: "Spellcheck Settings", Category: CategoryFile, PaletteHidden: true},
		// Quit lives in File on Windows/Linux. macOS's AppMenu role already
		// provides "Quit Downstage Write" in the app menu, so we skip this
		// entry on darwin to avoid duplicates.
		{ID: CmdFileQuit, Label: "Quit Downstage Write", Category: CategoryFile, Accelerator: "cmdorctrl+q", MenuPath: []string{"File"}, BeforeSeparator: true, Platforms: nonMacPlatforms},

		// Edit — macOS/Windows get the native EditMenu role prepended with
		// Undo/Redo/Cut/Copy/Paste/Select All. Linux's webkit2gtk chokes
		// on the role (GTK_IS_MENU_ITEM assertion), so these six are
		// reinstated as Linux-only catalog entries. The handlers dispatch
		// via the editor's imperative API.
		{ID: CmdEditUndo, Label: "Undo", Category: CategoryEdit, Accelerator: "cmdorctrl+z", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditRedo, Label: "Redo", Category: CategoryEdit, Accelerator: "cmdorctrl+shift+z", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditCut, Label: "Cut", Category: CategoryEdit, Accelerator: "cmdorctrl+x", MenuPath: []string{"Edit"}, BeforeSeparator: true, Platforms: linuxOnly},
		{ID: CmdEditCopy, Label: "Copy", Category: CategoryEdit, Accelerator: "cmdorctrl+c", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditPaste, Label: "Paste", Category: CategoryEdit, Accelerator: "cmdorctrl+v", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditSelectAll, Label: "Select All", Category: CategoryEdit, Accelerator: "cmdorctrl+a", MenuPath: []string{"Edit"}, BeforeSeparator: true, Platforms: linuxOnly},

		{ID: CmdEditFind, Label: "Find", Category: CategoryEdit, Accelerator: "cmdorctrl+f", MenuPath: []string{"Edit"}, BeforeSeparator: true},
		{ID: CmdEditFindReplace, Label: "Find & Replace", Category: CategoryEdit, Accelerator: "cmdorctrl+optionoralt+f", MenuPath: []string{"Edit"}},
		{ID: CmdEditCopyAll, Label: "Copy Whole Document", Category: CategoryEdit, MenuPath: []string{"Edit"}, BeforeSeparator: true},

		// View
		{ID: CmdViewCommandPalette, Label: "Command Palette…", Category: CategoryView, Accelerator: "cmdorctrl+k", MenuPath: []string{"View"}},
		{ID: CmdViewTogglePreview, Label: "Toggle Preview", Category: CategoryView, Accelerator: "cmdorctrl+\\", MenuPath: []string{"View"}, BeforeSeparator: true},
		{ID: CmdViewToggleSidebar, Label: "Toggle Sidebar", Category: CategoryView, Accelerator: "cmdorctrl+shift+b", MenuPath: []string{"View"}},
		{ID: CmdViewToggleIssues, Label: "Issues", Category: CategoryView, MenuPath: []string{"View"}, BeforeSeparator: true},
		{ID: CmdViewToggleOutline, Label: "Outline", Category: CategoryView, MenuPath: []string{"View"}},
		{ID: CmdViewToggleStats, Label: "Stats", Category: CategoryView, MenuPath: []string{"View"}},

		// Navigate
		{ID: CmdNavigateNextFile, Label: "Next File", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},
		{ID: CmdNavigatePrevFile, Label: "Previous File", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},
		{ID: CmdNavigateGoToFile, Label: "Go to File…", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},

		// Insert — structural elements that splice a template into the
		// document at the cursor. Lives in its own top-level menu because
		// these are document-structure operations, not character styles.
		{ID: CmdInsertCue, Label: "Dialogue", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertDirection, Label: "Stage Direction", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertAct, Label: "Act Heading", Category: CategoryInsert, MenuPath: []string{"Insert"}, BeforeSeparator: true},
		{ID: CmdInsertScene, Label: "Scene Heading", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertSong, Label: "Song Block", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertPageBreak, Label: "Page Break", Category: CategoryInsert, MenuPath: []string{"Insert"}, BeforeSeparator: true},

		// Format — character styles only. Structural insertions moved to
		// the Insert menu; keeping these four (bold/italic/underline/
		// strikethrough) mirrors what a user expects from "Format" in
		// any word processor.
		{ID: CmdFormatBold, Label: "Bold", Category: CategoryFormat, Accelerator: "cmdorctrl+b", MenuPath: []string{"Format"}},
		{ID: CmdFormatItalic, Label: "Italic", Category: CategoryFormat, Accelerator: "cmdorctrl+i", MenuPath: []string{"Format"}},
		{ID: CmdFormatUnderline, Label: "Underline", Category: CategoryFormat, Accelerator: "cmdorctrl+u", MenuPath: []string{"Format"}},
		{ID: CmdFormatStrikethrough, Label: "Strikethrough", Category: CategoryFormat, Accelerator: "cmdorctrl+shift+x", MenuPath: []string{"Format"}},

		// Library (palette + sidebar only; no menu entry)
		{ID: CmdLibraryNewFolder, Label: "New Folder", Category: CategoryFile},

		// Help
		{ID: CmdHelpToggle, Label: "View Help", Category: CategoryHelp, Accelerator: "cmdorctrl+shift+/", MenuPath: []string{"Help"}},
		{ID: CmdHelpGitHub, Label: "GitHub", Category: CategoryHelp, MenuPath: []string{"Help"}, BeforeSeparator: true},
		{ID: CmdHelpDocs, Label: "Documentation", Category: CategoryHelp, MenuPath: []string{"Help"}},
		{ID: CmdHelpAbout, Label: "About Downstage Write…", Category: CategoryHelp, MenuPath: []string{"Help"}, BeforeSeparator: true},
	}
}

// PlatformAllows reports whether the command should be visible on the
// given GOOS value. Used by both menu.go (menu rendering) and
// GetCommands (palette surface) so the two stay in sync.
func (c Command) PlatformAllows(goos string) bool {
	if len(c.Platforms) == 0 {
		return true
	}
	for _, p := range c.Platforms {
		if p == goos {
			return true
		}
	}
	return false
}
