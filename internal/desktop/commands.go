package desktop

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

type Command struct {
	ID              string
	Label           string
	Category        CommandCategory
	Accelerator     string
	MenuPath        []string
	BeforeSeparator bool
	PaletteHidden   bool
	Platforms       []string
}

type CommandMeta struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Category      string `json:"category"`
	Accelerator   string `json:"accelerator,omitempty"`
	PaletteHidden bool   `json:"paletteHidden,omitempty"`
}

const (
	CmdFileNewPlay            = "file.newPlay"
	CmdFileOpen               = "file.open"
	CmdFileSave               = "file.save"
	CmdFileSaveVersion        = "file.saveVersion"
	CmdFileExportPDF          = "file.exportPdf"
	CmdFileExportDs           = "file.exportDs"
	CmdFileSettings           = "file.settings"
	CmdFileSettingsSpellcheck = "file.settings.spellcheck"
	CmdFileQuit               = "file.quit"

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

	CmdLibraryNewFolder     = "library.newFolder"
	CmdLibraryDelete        = "library.delete"
	CmdLibraryRestore       = "library.restoreFile"
	CmdLibraryReviewChanges = "library.reviewChanges"
	CmdLibraryCommitChanges = "library.commitChanges"

	CmdHelpToggle = "help.toggle"
	CmdHelpGitHub = "help.github"
	CmdHelpDocs   = "help.docs"
	CmdHelpAbout  = "help.about"
)

var (
	linuxOnly       = []string{"linux"}
	nonMacPlatforms = []string{"linux", "windows"}
)

// Commands returns the canonical ordered catalog.
func Commands() []Command {
	return []Command{
		{ID: CmdFileNewPlay, Label: "New Play", Category: CategoryFile, Accelerator: "cmdorctrl+n", MenuPath: []string{"File"}},
		{ID: CmdFileOpen, Label: "Open…", Category: CategoryFile, Accelerator: "cmdorctrl+o", MenuPath: []string{"File"}},
		{ID: CmdFileSave, Label: "Save", Category: CategoryFile, Accelerator: "cmdorctrl+s", MenuPath: []string{"File"}, BeforeSeparator: true},
		{ID: CmdFileSaveVersion, Label: "Save Version…", Category: CategoryFile, Accelerator: "cmdorctrl+shift+s", MenuPath: []string{"File"}},
		{ID: CmdFileExportPDF, Label: "PDF…", Category: CategoryFile, Accelerator: "cmdorctrl+e", MenuPath: []string{"File", "Export"}},
		{ID: CmdFileExportDs, Label: "Downstage File…", Category: CategoryFile, MenuPath: []string{"File", "Export"}},
		{ID: CmdFileSettings, Label: "Settings…", Category: CategoryFile, Accelerator: "cmdorctrl+,", MenuPath: []string{"File"}, BeforeSeparator: true},
		{ID: CmdFileSettingsSpellcheck, Label: "Spellcheck Settings", Category: CategoryFile, PaletteHidden: true},
		{ID: CmdFileQuit, Label: "Quit Downstage Write", Category: CategoryFile, Accelerator: "cmdorctrl+q", MenuPath: []string{"File"}, BeforeSeparator: true, Platforms: nonMacPlatforms},

		{ID: CmdEditUndo, Label: "Undo", Category: CategoryEdit, Accelerator: "cmdorctrl+z", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditRedo, Label: "Redo", Category: CategoryEdit, Accelerator: "cmdorctrl+shift+z", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditCut, Label: "Cut", Category: CategoryEdit, Accelerator: "cmdorctrl+x", MenuPath: []string{"Edit"}, BeforeSeparator: true, Platforms: linuxOnly},
		{ID: CmdEditCopy, Label: "Copy", Category: CategoryEdit, Accelerator: "cmdorctrl+c", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditPaste, Label: "Paste", Category: CategoryEdit, Accelerator: "cmdorctrl+v", MenuPath: []string{"Edit"}, Platforms: linuxOnly},
		{ID: CmdEditSelectAll, Label: "Select All", Category: CategoryEdit, Accelerator: "cmdorctrl+a", MenuPath: []string{"Edit"}, BeforeSeparator: true, Platforms: linuxOnly},

		{ID: CmdEditFind, Label: "Find", Category: CategoryEdit, Accelerator: "cmdorctrl+f", MenuPath: []string{"Edit"}, BeforeSeparator: true},
		{ID: CmdEditFindReplace, Label: "Find & Replace", Category: CategoryEdit, Accelerator: "cmdorctrl+optionoralt+f", MenuPath: []string{"Edit"}},
		{ID: CmdEditCopyAll, Label: "Copy Whole Document", Category: CategoryEdit, MenuPath: []string{"Edit"}, BeforeSeparator: true},

		{ID: CmdViewCommandPalette, Label: "Command Palette…", Category: CategoryView, Accelerator: "cmdorctrl+k", MenuPath: []string{"View"}},
		{ID: CmdViewTogglePreview, Label: "Toggle Preview", Category: CategoryView, Accelerator: "cmdorctrl+\\", MenuPath: []string{"View"}, BeforeSeparator: true},
		{ID: CmdViewToggleSidebar, Label: "Toggle Sidebar", Category: CategoryView, Accelerator: "cmdorctrl+shift+b", MenuPath: []string{"View"}},
		{ID: CmdViewToggleIssues, Label: "Issues", Category: CategoryView, MenuPath: []string{"View"}, BeforeSeparator: true},
		{ID: CmdViewToggleOutline, Label: "Outline", Category: CategoryView, MenuPath: []string{"View"}},
		{ID: CmdViewToggleStats, Label: "Stats", Category: CategoryView, MenuPath: []string{"View"}},

		{ID: CmdNavigateNextFile, Label: "Next File", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},
		{ID: CmdNavigatePrevFile, Label: "Previous File", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},
		{ID: CmdNavigateGoToFile, Label: "Go to File…", Category: CategoryNavigate, MenuPath: []string{"Navigate"}},

		{ID: CmdInsertCue, Label: "Dialogue", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertDirection, Label: "Stage Direction", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertAct, Label: "Act Heading", Category: CategoryInsert, MenuPath: []string{"Insert"}, BeforeSeparator: true},
		{ID: CmdInsertScene, Label: "Scene Heading", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertSong, Label: "Song Block", Category: CategoryInsert, MenuPath: []string{"Insert"}},
		{ID: CmdInsertPageBreak, Label: "Page Break", Category: CategoryInsert, MenuPath: []string{"Insert"}, BeforeSeparator: true},

		{ID: CmdFormatBold, Label: "Bold", Category: CategoryFormat, Accelerator: "cmdorctrl+b", MenuPath: []string{"Format"}},
		{ID: CmdFormatItalic, Label: "Italic", Category: CategoryFormat, Accelerator: "cmdorctrl+i", MenuPath: []string{"Format"}},
		{ID: CmdFormatUnderline, Label: "Underline", Category: CategoryFormat, Accelerator: "cmdorctrl+u", MenuPath: []string{"Format"}},
		{ID: CmdFormatStrikethrough, Label: "Strikethrough", Category: CategoryFormat, Accelerator: "cmdorctrl+shift+x", MenuPath: []string{"Format"}},

		{ID: CmdLibraryNewFolder, Label: "New Folder", Category: CategoryFile},
		{ID: CmdLibraryDelete, Label: "Delete File…", Category: CategoryFile, PaletteHidden: true},
		{ID: CmdLibraryRestore, Label: "Restore Deleted File", Category: CategoryFile, PaletteHidden: true},
		{ID: CmdLibraryReviewChanges, Label: "Review Library Changes…", Category: CategoryFile},
		{ID: CmdLibraryCommitChanges, Label: "Commit Library Changes", Category: CategoryFile, PaletteHidden: true},

		{ID: CmdHelpToggle, Label: "View Help", Category: CategoryHelp, Accelerator: "cmdorctrl+shift+/", MenuPath: []string{"Help"}},
		{ID: CmdHelpGitHub, Label: "GitHub", Category: CategoryHelp, MenuPath: []string{"Help"}, BeforeSeparator: true},
		{ID: CmdHelpDocs, Label: "Documentation", Category: CategoryHelp, MenuPath: []string{"Help"}},
		{ID: CmdHelpAbout, Label: "About Downstage Write…", Category: CategoryHelp, MenuPath: []string{"Help"}, BeforeSeparator: true},
	}
}

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
