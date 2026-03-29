package html

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

var _ render.NodeRenderer = (*htmlRenderer)(nil)

// NewRenderer creates an HTML NodeRenderer that produces a self-contained
// HTML document with embedded CSS. The style (standard or condensed) is
// selected from cfg.Style.
func NewRenderer(cfg render.Config) render.NodeRenderer {
	return &htmlRenderer{cfg: cfg}
}

type htmlRenderer struct {
	cfg            render.Config
	w              io.Writer
	buf            strings.Builder
	dirDepth       int
	hasTitlePage   bool
	titlePageTitle string
	inDualDialogue bool
	skipSection    bool
	inParagraph    bool // tracks open <p> in section lines for prose reflow
}

// --- Lifecycle ---

func (r *htmlRenderer) BeginDocument(doc *ast.Document, w io.Writer) error {
	r.w = w
	r.hasTitlePage = doc.TitlePage != nil
	r.titlePageTitle = titlePageTitle(doc.TitlePage)

	title := r.titlePageTitle
	if title == "" {
		title = "Untitled"
	}

	css := standardCSS
	if r.cfg.Style == render.StyleCondensed {
		css = condensedCSS
	}

	r.buf.WriteString("<!DOCTYPE html>\n")
	r.buf.WriteString("<html lang=\"en\">\n<head>\n")
	r.buf.WriteString("<meta charset=\"utf-8\">\n")
	r.buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&r.buf, "<title>%s</title>\n", html.EscapeString(title))

	if doc.TitlePage != nil {
		for _, kv := range doc.TitlePage.Entries {
			if strings.EqualFold(kv.Key, "author") {
				fmt.Fprintf(&r.buf, "<meta name=\"author\" content=\"%s\">\n", html.EscapeString(kv.Value))
			}
		}
	}

	r.buf.WriteString("<meta name=\"generator\" content=\"Downstage\">\n")
	r.buf.WriteString("<style>\n")
	r.buf.WriteString(css)
	r.buf.WriteString("\n</style>\n")
	r.buf.WriteString("</head>\n<body>\n")
	r.buf.WriteString("<div class=\"downstage-document\">\n")
	return nil
}

func (r *htmlRenderer) EndDocument(_ *ast.Document) error {
	r.buf.WriteString("</div>\n</body>\n</html>\n")
	_, err := io.WriteString(r.w, r.buf.String())
	return err
}

// --- Front matter ---

func (r *htmlRenderer) RenderTitlePage(tp *ast.TitlePage) error {
	var title, subtitle, author string
	var other []ast.KeyValue

	for _, kv := range tp.Entries {
		switch strings.ToLower(kv.Key) {
		case "title":
			title = kv.Value
		case "subtitle":
			subtitle = kv.Value
		case "author":
			author = kv.Value
		default:
			other = append(other, kv)
		}
	}

	r.buf.WriteString("<header class=\"downstage-title-page\">\n")

	if title != "" {
		fmt.Fprintf(&r.buf, "<h1>%s</h1>\n", html.EscapeString(title))
	}
	if subtitle != "" {
		fmt.Fprintf(&r.buf, "<p class=\"subtitle\">%s</p>\n", html.EscapeString(subtitle))
	}
	if author != "" {
		r.buf.WriteString("<p class=\"author\">by</p>\n")
		fmt.Fprintf(&r.buf, "<p class=\"author\">%s</p>\n", html.EscapeString(author))
	}

	if len(other) > 0 {
		r.buf.WriteString("<dl class=\"metadata\">\n")
		for _, kv := range other {
			r.buf.WriteString("<div>")
			fmt.Fprintf(&r.buf, "<dt>%s</dt>", html.EscapeString(kv.Key))
			fmt.Fprintf(&r.buf, "<dd>%s</dd>", html.EscapeString(kv.Value))
			r.buf.WriteString("</div>\n")
		}
		r.buf.WriteString("</dl>\n")
	}

	r.buf.WriteString("</header>\n")
	return nil
}

// --- Sections ---

func (r *htmlRenderer) BeginSection(s *ast.Section) error {
	r.skipSection = false

	switch s.Kind {
	case ast.SectionAct:
		return r.beginAct(s)
	case ast.SectionScene:
		return r.beginScene(s)
	case ast.SectionDramatisPersonae:
		return r.renderDramatisPersonae(s)
	default: // SectionGeneric
		if r.hasTitlePage && s.Level == 1 && strings.EqualFold(strings.TrimSpace(s.Title), r.titlePageTitle) {
			r.skipSection = true
			return nil
		}
		if s.Level == 0 {
			fmt.Fprintf(&r.buf, "<p class=\"downstage-forced-heading\"><strong>%s</strong></p>\n",
				html.EscapeString(s.Title))
			return nil
		}
		r.buf.WriteString("<section class=\"downstage-section\">\n")
		if s.Title != "" {
			tag := headingTag(s.Level)
			fmt.Fprintf(&r.buf, "<%s>%s</%s>\n", tag, html.EscapeString(strings.ToUpper(s.Title)), tag)
		}
		return nil
	}
}

func (r *htmlRenderer) EndSection(s *ast.Section) error {
	r.closeParagraph()
	if r.skipSection {
		r.skipSection = false
		return nil
	}
	if s.Level == 0 && s.Kind == ast.SectionGeneric {
		return nil
	}
	switch s.Kind {
	case ast.SectionDramatisPersonae:
		// already closed in renderDramatisPersonae
	default:
		r.buf.WriteString("</section>\n")
	}
	return nil
}

func (r *htmlRenderer) BeginSectionLine(sl *ast.SectionLine) error {
	if r.skipSection {
		return nil
	}
	if len(sl.Content) == 0 {
		r.closeParagraph()
		r.buf.WriteString("<div class=\"downstage-section-break\"></div>\n")
		return nil
	}
	if !r.inParagraph {
		r.buf.WriteString("<p>")
		r.inParagraph = true
	}
	return nil
}

func (r *htmlRenderer) EndSectionLine(sl *ast.SectionLine) error {
	if r.skipSection {
		return nil
	}
	if len(sl.Content) > 0 {
		r.buf.WriteString(" ")
	}
	return nil
}

func (r *htmlRenderer) beginAct(s *ast.Section) error {
	if s.Number == "" && r.hasTitlePage {
		r.skipSection = true
		return nil
	}

	r.buf.WriteString("<section class=\"downstage-act\">\n")

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "ACT " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "ACT " + s.Number
	default:
		heading = s.Title
	}

	fmt.Fprintf(&r.buf, "<h2>%s</h2>\n", html.EscapeString(strings.ToUpper(heading)))
	return nil
}

func (r *htmlRenderer) beginScene(s *ast.Section) error {
	r.buf.WriteString("<section class=\"downstage-scene\">\n")

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "SCENE " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "SCENE " + s.Number
	default:
		heading = s.Title
	}

	fmt.Fprintf(&r.buf, "<h3>%s</h3>\n", html.EscapeString(strings.ToUpper(heading)))
	return nil
}

func (r *htmlRenderer) renderDramatisPersonae(s *ast.Section) error {
	r.buf.WriteString("<section class=\"downstage-dramatis-personae\">\n")
	r.buf.WriteString("<h2>DRAMATIS PERSONAE</h2>\n")
	r.buf.WriteString("<dl>\n")

	for _, ch := range s.Characters {
		r.renderCharacterEntry(ch)
	}

	for _, group := range s.Groups {
		r.buf.WriteString("</dl>\n")
		fmt.Fprintf(&r.buf, "<p class=\"downstage-character-group-name\">%s</p>\n",
			html.EscapeString(group.Name))
		r.buf.WriteString("<dl>\n")
		for _, ch := range group.Characters {
			r.renderCharacterEntry(ch)
		}
	}

	r.buf.WriteString("</dl>\n")
	r.buf.WriteString("</section>\n")
	return nil
}

func (r *htmlRenderer) renderCharacterEntry(ch ast.Character) {
	r.buf.WriteString("<div class=\"character-entry\">")
	fmt.Fprintf(&r.buf, "<dt>%s</dt>", html.EscapeString(ch.Name))
	if ch.Description != "" {
		fmt.Fprintf(&r.buf, "<dd>%s</dd>", html.EscapeString(ch.Description))
	}
	r.buf.WriteString("</div>\n")
}

// --- Dual Dialogue ---

func (r *htmlRenderer) BeginDualDialogue(_ *ast.DualDialogue) error {
	r.inDualDialogue = true
	r.buf.WriteString("<div class=\"downstage-dual-dialogue\">\n")
	return nil
}

func (r *htmlRenderer) EndDualDialogue(_ *ast.DualDialogue) error {
	r.inDualDialogue = false
	r.buf.WriteString("</div>\n")
	return nil
}

// --- Dialogue ---

func (r *htmlRenderer) BeginDialogue(d *ast.Dialogue) error {
	r.buf.WriteString("<div class=\"downstage-dialogue\">\n")
	fmt.Fprintf(&r.buf, "<p class=\"downstage-character\">%s", html.EscapeString(strings.ToUpper(d.Character)))
	if d.Parenthetical != "" {
		paren := d.Parenthetical
		if len(paren) == 0 || paren[0] != '(' {
			paren = "(" + paren + ")"
		}
		fmt.Fprintf(&r.buf, " <span class=\"downstage-parenthetical\">%s</span>",
			html.EscapeString(paren))
	}
	r.buf.WriteString("</p>\n")
	return nil
}

func (r *htmlRenderer) EndDialogue(_ *ast.Dialogue) error {
	r.buf.WriteString("</div>\n")
	return nil
}

func (r *htmlRenderer) BeginDialogueLine(line *ast.DialogueLine) error {
	cls := "downstage-line"
	if line.IsVerse {
		cls += " downstage-verse"
	}
	fmt.Fprintf(&r.buf, "<p class=\"%s\">", cls)
	return nil
}

func (r *htmlRenderer) EndDialogueLine(_ *ast.DialogueLine) error {
	r.buf.WriteString("</p>\n")
	return nil
}

// --- Stage Direction ---

func (r *htmlRenderer) BeginStageDirection(_ *ast.StageDirection) error {
	r.buf.WriteString("<p class=\"downstage-stage-direction\">")
	return nil
}

func (r *htmlRenderer) EndStageDirection(_ *ast.StageDirection) error {
	r.buf.WriteString("</p>\n")
	return nil
}

// --- Song ---

func (r *htmlRenderer) BeginSong(song *ast.Song) error {
	r.buf.WriteString("<div class=\"downstage-song\">\n")

	header := "SONG"
	if song.Number != "" {
		header = fmt.Sprintf("SONG %s", song.Number)
	}
	if song.Title != "" {
		header += ": " + song.Title
	}

	fmt.Fprintf(&r.buf, "<h4>%s</h4>\n", html.EscapeString(header))
	return nil
}

func (r *htmlRenderer) EndSong(_ *ast.Song) error {
	r.buf.WriteString("<p class=\"downstage-song-end\">SONG END</p>\n")
	r.buf.WriteString("</div>\n")
	return nil
}

// --- Verse Block ---

func (r *htmlRenderer) BeginVerseBlock(_ *ast.VerseBlock) error {
	r.buf.WriteString("<div class=\"downstage-verse-block\">\n")
	return nil
}

func (r *htmlRenderer) EndVerseBlock(_ *ast.VerseBlock) error {
	r.buf.WriteString("</div>\n")
	return nil
}

func (r *htmlRenderer) BeginVerseLine(_ *ast.VerseLine) error {
	r.buf.WriteString("<p class=\"downstage-verse-line\">")
	return nil
}

func (r *htmlRenderer) EndVerseLine(_ *ast.VerseLine) error {
	r.buf.WriteString("</p>\n")
	return nil
}

// --- Leaves ---

func (r *htmlRenderer) RenderPageBreak(_ *ast.PageBreak) error {
	r.buf.WriteString("<hr class=\"downstage-page-break\">\n")
	return nil
}

func (r *htmlRenderer) RenderComment(_ *ast.Comment) error {
	return nil
}

// --- Inline ---

func (r *htmlRenderer) RenderText(t *ast.TextNode) error {
	r.buf.WriteString(html.EscapeString(t.Value))
	return nil
}

func (r *htmlRenderer) BeginBold(_ *ast.BoldNode) error {
	r.buf.WriteString("<strong>")
	return nil
}

func (r *htmlRenderer) EndBold(_ *ast.BoldNode) error {
	r.buf.WriteString("</strong>")
	return nil
}

func (r *htmlRenderer) BeginItalic(_ *ast.ItalicNode) error {
	r.buf.WriteString("<em>")
	return nil
}

func (r *htmlRenderer) EndItalic(_ *ast.ItalicNode) error {
	r.buf.WriteString("</em>")
	return nil
}

func (r *htmlRenderer) BeginBoldItalic(_ *ast.BoldItalicNode) error {
	r.buf.WriteString("<strong><em>")
	return nil
}

func (r *htmlRenderer) EndBoldItalic(_ *ast.BoldItalicNode) error {
	r.buf.WriteString("</em></strong>")
	return nil
}

func (r *htmlRenderer) BeginUnderline(_ *ast.UnderlineNode) error {
	r.buf.WriteString("<u>")
	return nil
}

func (r *htmlRenderer) EndUnderline(_ *ast.UnderlineNode) error {
	r.buf.WriteString("</u>")
	return nil
}

func (r *htmlRenderer) BeginStrikethrough(_ *ast.StrikethroughNode) error {
	r.buf.WriteString("<del>")
	return nil
}

func (r *htmlRenderer) EndStrikethrough(_ *ast.StrikethroughNode) error {
	r.buf.WriteString("</del>")
	return nil
}

func (r *htmlRenderer) BeginInlineDirection(_ *ast.InlineDirectionNode) error {
	r.dirDepth++
	if r.dirDepth == 1 {
		r.buf.WriteString("<span class=\"downstage-inline-direction\">(")
	}
	return nil
}

func (r *htmlRenderer) EndInlineDirection(_ *ast.InlineDirectionNode) error {
	r.dirDepth--
	if r.dirDepth == 0 {
		r.buf.WriteString(")</span>")
	}
	return nil
}

// --- Helpers ---

func titlePageTitle(tp *ast.TitlePage) string {
	if tp == nil {
		return ""
	}
	for _, entry := range tp.Entries {
		if strings.EqualFold(strings.TrimSpace(entry.Key), "title") {
			return strings.TrimSpace(entry.Value)
		}
	}
	return ""
}

func (r *htmlRenderer) closeParagraph() {
	if r.inParagraph {
		r.buf.WriteString("</p>\n")
		r.inParagraph = false
	}
}

func headingTag(level int) string {
	if level >= 1 && level <= 6 {
		return fmt.Sprintf("h%d", level)
	}
	return "h2"
}
