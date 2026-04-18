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
	CategoryFormat   CommandCategory = "format"
	CategoryHelp     CommandCategory = "help"
)

// Command is the full catalog entry used by menu.go. Not exposed to the
// frontend verbatim — GetCommands returns CommandMeta instead.
type Command struct {
	ID              string
	Label           string
	Category        CommandCategory
	Accelerator     string
	MenuPath        []string
	BeforeSeparator bool
	// PaletteVisible defaults to true; set false to hide from the palette
	// (e.g. programmatic commands that shouldn't appear as a user-selectable
	// entry).
	PaletteHidden bool
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
	CmdFileSettings           = "file.settings"
	CmdFileSettingsSpellcheck = "file.settings.spellcheck"

	CmdEditFind        = "edit.find"
	CmdEditFindReplace = "edit.findReplace"
	CmdEditCopyAll     = "edit.copyAll"

	CmdViewCommandPalette   = "view.commandPalette"
	CmdViewTogglePreview    = "view.togglePreview"
	CmdViewToggleSidebar    = "view.toggleSidebar"
	CmdViewToggleIssues     = "view.toggleIssues"
	CmdViewToggleOutline    = "view.toggleOutline"
	CmdViewToggleStats      = "view.toggleStats"
	CmdViewToggleDrawerDock = "view.toggleDrawerDock"

	CmdNavigateNextFile = "navigate.nextFile"
	CmdNavigatePrevFile = "navigate.prevFile"
	CmdNavigateGoToFile = "navigate.goToFile"

	CmdFormatBold      = "format.bold"
	CmdFormatItalic    = "format.italic"
	CmdFormatUnderline = "format.underline"
	CmdFormatCue       = "format.cue"
	CmdFormatDirection = "format.direction"
	CmdFormatAct       = "format.act"
	CmdFormatScene     = "format.scene"
	CmdFormatSong      = "format.song"
	CmdFormatPageBreak = "format.pageBreak"

	CmdHelpToggle = "help.toggle"
	CmdHelpGitHub = "help.github"
	CmdHelpDocs   = "help.docs"
	CmdHelpAbout  = "help.about"
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
		{ID: CmdFileExportPDF, Label: "Export PDF…", Category: CategoryFile, Accelerator: "cmdorctrl+e", MenuPath: []string{"File"}},
		{ID: CmdFileSettings, Label: "Settings…", Category: CategoryFile, Accelerator: "cmdorctrl+,", MenuPath: []string{"File"}, BeforeSeparator: true},
		// Palette-hidden programmatic command used by the in-editor
		// SpellCheck toolbar button to open Settings on the Spellcheck tab.
		{ID: CmdFileSettingsSpellcheck, Label: "Spellcheck Settings", Category: CategoryFile, PaletteHidden: true},

		// Edit (the EditMenu native role covers cut/copy/paste/undo/redo
		// and gets prepended by BuildMenu — these are the Downstage-specific
		// additions).
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
		{ID: CmdViewToggleDrawerDock, Label: "Toggle Drawer Dock", Category: CategoryView, MenuPath: []string{"View"}, BeforeSeparator: true},

		// Navigate
		{ID: CmdNavigateNextFile, Label: "Next File", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},
		{ID: CmdNavigatePrevFile, Label: "Previous File", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},
		{ID: CmdNavigateGoToFile, Label: "Go to File…", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},

		// Format
		{ID: CmdFormatBold, Label: "Bold", Category: CategoryFormat, Accelerator: "cmdorctrl+b", MenuPath: []string{"Format"}},
		{ID: CmdFormatItalic, Label: "Italic", Category: CategoryFormat, Accelerator: "cmdorctrl+i", MenuPath: []string{"Format"}},
		{ID: CmdFormatUnderline, Label: "Underline", Category: CategoryFormat, Accelerator: "cmdorctrl+u", MenuPath: []string{"Format"}},
		{ID: CmdFormatCue, Label: "Dialogue", Category: CategoryFormat, MenuPath: []string{"Format"}, BeforeSeparator: true},
		{ID: CmdFormatDirection, Label: "Stage Direction", Category: CategoryFormat, MenuPath: []string{"Format"}},
		{ID: CmdFormatAct, Label: "Act Heading", Category: CategoryFormat, MenuPath: []string{"Format"}},
		{ID: CmdFormatScene, Label: "Scene Heading", Category: CategoryFormat, MenuPath: []string{"Format"}},
		{ID: CmdFormatSong, Label: "Song Block", Category: CategoryFormat, MenuPath: []string{"Format"}},
		{ID: CmdFormatPageBreak, Label: "Page Break", Category: CategoryFormat, MenuPath: []string{"Format"}},

		// Help
		{ID: CmdHelpToggle, Label: "View Help", Category: CategoryHelp, Accelerator: "cmdorctrl+shift+/", MenuPath: []string{"Help"}},
		{ID: CmdHelpGitHub, Label: "GitHub", Category: CategoryHelp, MenuPath: []string{"Help"}, BeforeSeparator: true},
		{ID: CmdHelpDocs, Label: "Documentation", Category: CategoryHelp, MenuPath: []string{"Help"}},
		{ID: CmdHelpAbout, Label: "About Downstage Write…", Category: CategoryHelp, MenuPath: []string{"Help"}, BeforeSeparator: true},
	}
}
