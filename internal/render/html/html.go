package html

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
	"github.com/jscaltreto/downstage/internal/token"
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
	inlinePlay     map[*ast.Section]bool
	activePlay     *ast.Section
	inDualDialogue bool
	inParagraph    bool // tracks open <p> in section lines for prose reflow
	sectionStack   []sectionState
}

type sectionState struct {
	closeTag bool
}

func (r *htmlRenderer) sourceAttr(rng token.Range) string {
	if !r.cfg.SourceAnchors {
		return ""
	}
	return fmt.Sprintf(` data-source-line="%d"`, rng.Start.Line+1)
}

// --- Lifecycle ---

func (r *htmlRenderer) BeginDocument(doc *ast.Document, w io.Writer) error {
	r.buf.Reset()
	r.w = w
	r.dirDepth = 0
	tp := render.DocumentTitlePage(doc)
	r.hasTitlePage = tp != nil
	r.titlePageTitle = titlePageTitle(tp)
	r.inlinePlay = make(map[*ast.Section]bool)
	for _, section := range render.PlayableTopLevelSections(doc) {
		if render.IsInlinePlaySection(doc, section) {
			r.inlinePlay[section] = true
		}
	}
	r.activePlay = nil
	r.inDualDialogue = false
	r.inParagraph = false
	r.sectionStack = r.sectionStack[:0]

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

	if tp != nil {
		for _, kv := range tp.Entries {
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
	titleKV, subtitleKV, authorKVs, other := partitionHTMLTitlePageEntries(tp)

	titleText := ""
	if titleKV != nil {
		titleText = titleKV.Value
	}

	r.hasTitlePage = true
	r.titlePageTitle = titleText

	fmt.Fprintf(&r.buf, "<header class=\"downstage-title-page\"%s>\n", r.sourceAttr(tp.NodeRange()))

	if titleKV != nil && titleKV.Value != "" {
		fmt.Fprintf(&r.buf, "<h1%s>%s</h1>\n", r.sourceAttr(titleKV.Range), html.EscapeString(titleKV.Value))
	}
	if subtitleKV != nil && hasKeyValueContent(*subtitleKV) {
		fmt.Fprintf(&r.buf, "<p class=\"subtitle\"%s>", r.sourceAttr(subtitleKV.Range))
		r.renderInlineContent(keyValueInlines(*subtitleKV))
		r.buf.WriteString("</p>\n")
	}
	if len(authorKVs) > 0 {
		fmt.Fprintf(&r.buf, "<p class=\"author\"%s>by</p>\n", r.sourceAttr(authorKVs[0].Range))
		for _, kv := range authorKVs {
			fmt.Fprintf(&r.buf, "<p class=\"author\"%s>%s</p>\n", r.sourceAttr(kv.Range), html.EscapeString(kv.Value))
		}
	}

	if len(other) > 0 {
		r.buf.WriteString("<dl class=\"metadata\">\n")
		for _, kv := range other {
			fmt.Fprintf(&r.buf, "<div%s>", r.sourceAttr(kv.Range))
			fmt.Fprintf(&r.buf, "<dt>%s</dt>", html.EscapeString(kv.Key))
			r.buf.WriteString("<dd>")
			r.renderInlineContent(keyValueInlines(kv))
			r.buf.WriteString("</dd>")
			r.buf.WriteString("</div>\n")
		}
		r.buf.WriteString("</dl>\n")
	}

	r.buf.WriteString("</header>\n")
	return nil
}

// --- Sections ---

func (r *htmlRenderer) BeginSection(s *ast.Section) error {
	r.beginBlock()

	switch s.Kind {
	case ast.SectionAct:
		return r.beginAct(s)
	case ast.SectionScene:
		return r.beginScene(s)
	case ast.SectionDramatisPersonae:
		return r.renderDramatisPersonae(s)
	default: // SectionGeneric
		if render.IsLegacyTopLevelDramatisPersonae(s) {
			r.pushSection(false)
			return nil
		}
		if s.Level == 1 {
			r.activePlay = s
		}
		if r.hasTitlePage && s.Level == 1 && strings.EqualFold(strings.TrimSpace(render.SectionDisplayTitle(s)), r.titlePageTitle) {
			r.pushSection(false)
			return nil
		}
		if s.Level == 1 && r.inlinePlay[s] {
			r.pushSection(true)
			fmt.Fprintf(&r.buf, "<section class=\"downstage-subplay\"%s>\n", r.sourceAttr(s.NodeRange()))
			r.buf.WriteString("<header class=\"downstage-subplay-header\">\n")
			fmt.Fprintf(&r.buf, "<h1%s>%s</h1>\n", r.sourceAttr(s.HeadingRange()), html.EscapeString(render.SectionDisplayTitle(s)))
			r.renderSubplayMetadata(s.Metadata)
			r.buf.WriteString("</header>\n")
			return nil
		}
		if s.Level == 0 {
			r.pushSection(false)
			fmt.Fprintf(&r.buf, "<p class=\"downstage-forced-heading\"><strong>%s</strong></p>\n",
				html.EscapeString(s.Title))
			return nil
		}
		r.pushSection(true)
		fmt.Fprintf(&r.buf, "<section class=\"downstage-section\"%s>\n", r.sourceAttr(s.NodeRange()))
		if s.Title != "" {
			tag := headingTag(s.Level)
			fmt.Fprintf(&r.buf, "<%s%s>%s</%s>\n", tag, r.sourceAttr(s.HeadingRange()), html.EscapeString(strings.ToUpper(s.Title)), tag)
		}
		return nil
	}
}

func (r *htmlRenderer) EndSection(s *ast.Section) error {
	r.closeParagraph()
	state := r.popSection()
	if s.Level == 1 {
		r.activePlay = nil
	}
	if s.Level == 0 && s.Kind == ast.SectionGeneric {
		return nil
	}
	switch s.Kind {
	case ast.SectionDramatisPersonae:
		// already closed in renderDramatisPersonae
	default:
		if state.closeTag {
			r.buf.WriteString("</section>\n")
		}
	}
	return nil
}

func (r *htmlRenderer) BeginSectionLine(sl *ast.SectionLine) error {
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
	if len(sl.Content) > 0 {
		r.buf.WriteString(" ")
	}
	return nil
}

func (r *htmlRenderer) beginAct(s *ast.Section) error {
	if s.Number == "" && r.hasTitlePage {
		r.pushSection(false)
		return nil
	}

	r.pushSection(true)
	fmt.Fprintf(&r.buf, "<section class=\"downstage-act\"%s>\n", r.sourceAttr(s.NodeRange()))

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "ACT " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "ACT " + s.Number
	default:
		heading = s.Title
	}

	fmt.Fprintf(&r.buf, "<h2%s>%s</h2>\n", r.sourceAttr(s.HeadingRange()), html.EscapeString(strings.ToUpper(heading)))
	return nil
}

func (r *htmlRenderer) beginScene(s *ast.Section) error {
	r.pushSection(true)
	fmt.Fprintf(&r.buf, "<section class=\"downstage-scene\"%s>\n", r.sourceAttr(s.NodeRange()))

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "SCENE " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "SCENE " + s.Number
	default:
		heading = s.Title
	}

	fmt.Fprintf(&r.buf, "<h3%s>%s</h3>\n", r.sourceAttr(s.HeadingRange()), html.EscapeString(strings.ToUpper(heading)))
	return nil
}

func (r *htmlRenderer) renderDramatisPersonae(s *ast.Section) error {
	r.pushSection(false)
	className := "downstage-dramatis-personae"
	if r.activePlay != nil && r.inlinePlay[r.activePlay] {
		className += " downstage-dramatis-personae-inline"
	}
	fmt.Fprintf(&r.buf, "<section class=\"%s\"%s>\n", className, r.sourceAttr(s.NodeRange()))
	fmt.Fprintf(&r.buf, "<h2>%s</h2>\n", html.EscapeString(strings.ToUpper(render.DramatisPersonaeDisplayTitle(s))))
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

func (r *htmlRenderer) renderMetadata(className string, tp *ast.TitlePage) {
	if tp == nil || len(tp.Entries) == 0 {
		return
	}

	r.buf.WriteString("<dl class=\"")
	r.buf.WriteString(className)
	r.buf.WriteString("\">\n")
	for _, kv := range tp.Entries {
		if strings.EqualFold(kv.Key, "title") {
			continue
		}
		r.buf.WriteString("<div>")
		fmt.Fprintf(&r.buf, "<dt>%s</dt>", html.EscapeString(kv.Key))
		r.buf.WriteString("<dd>")
		r.renderInlineContent(keyValueInlines(kv))
		r.buf.WriteString("</dd>")
		r.buf.WriteString("</div>\n")
	}
	r.buf.WriteString("</dl>\n")
}

func (r *htmlRenderer) renderSubplayMetadata(tp *ast.TitlePage) {
	_, subtitleKV, authorKVs, other := partitionHTMLTitlePageEntries(tp)
	if subtitleKV != nil && hasKeyValueContent(*subtitleKV) {
		r.buf.WriteString("<p class=\"downstage-subplay-subtitle\">")
		r.renderInlineContent(keyValueInlines(*subtitleKV))
		r.buf.WriteString("</p>\n")
	}
	if len(authorKVs) > 0 {
		r.buf.WriteString("<p class=\"downstage-subplay-author-label\">by</p>\n")
		for _, kv := range authorKVs {
			fmt.Fprintf(&r.buf, "<p class=\"downstage-subplay-author\">%s</p>\n", html.EscapeString(kv.Value))
		}
	}
	if len(other) > 0 {
		r.renderMetadata("downstage-subplay-metadata", &ast.TitlePage{Entries: other})
	}
}

// partitionHTMLTitlePageEntries splits a title page into the slots the HTML
// renderer cares about, preserving source ranges so callers can emit anchors.
// Duplicate Title/Subtitle entries follow last-wins semantics, matching the
// PDF renderer.
func partitionHTMLTitlePageEntries(tp *ast.TitlePage) (title *ast.KeyValue, subtitle *ast.KeyValue, authors []ast.KeyValue, other []ast.KeyValue) {
	if tp == nil {
		return nil, nil, nil, nil
	}
	for i := range tp.Entries {
		kv := tp.Entries[i]
		switch strings.ToLower(strings.TrimSpace(kv.Key)) {
		case "title":
			c := kv
			title = &c
		case "subtitle":
			c := kv
			subtitle = &c
		case "author":
			if strings.TrimSpace(kv.Value) != "" {
				authors = append(authors, kv)
			}
		default:
			other = append(other, kv)
		}
	}
	return title, subtitle, authors, other
}

func (r *htmlRenderer) renderCharacterEntry(ch ast.Character) {
	r.buf.WriteString("<div class=\"character-entry\">")
	fmt.Fprintf(&r.buf, "<dt>%s</dt>", html.EscapeString(render.CharacterDisplayName(ch)))
	if hasCharacterDescription(ch) {
		r.buf.WriteString("<dd>")
		r.renderInlineContent(characterDescriptionInlines(ch))
		r.buf.WriteString("</dd>")
	}
	r.buf.WriteString("</div>\n")
}

// --- Dual Dialogue ---

func (r *htmlRenderer) BeginDualDialogue(dd *ast.DualDialogue) error {
	r.beginBlock()
	r.inDualDialogue = true
	fmt.Fprintf(&r.buf, "<div class=\"downstage-dual-dialogue\"%s>\n", r.sourceAttr(dd.NodeRange()))
	return nil
}

func (r *htmlRenderer) EndDualDialogue(_ *ast.DualDialogue) error {
	r.inDualDialogue = false
	r.buf.WriteString("</div>\n")
	return nil
}

// --- Dialogue ---

func (r *htmlRenderer) BeginDialogue(d *ast.Dialogue) error {
	r.beginBlock()
	fmt.Fprintf(&r.buf, "<div class=\"downstage-dialogue\"%s>\n", r.sourceAttr(d.NodeRange()))
	fmt.Fprintf(&r.buf, "<p class=\"downstage-character\">%s</p>\n", html.EscapeString(strings.ToUpper(d.Character)))
	if d.Parenthetical != "" {
		r.buf.WriteString("<p class=\"downstage-parenthetical\">")
		r.buf.WriteString("(")
		r.renderInlineContent(dialogueParentheticalInlines(d))
		r.buf.WriteString(")</p>\n")
	}
	return nil
}

func (r *htmlRenderer) EndDialogue(_ *ast.Dialogue) error {
	r.buf.WriteString("</div>\n")
	return nil
}

func (r *htmlRenderer) BeginDialogueLine(line *ast.DialogueLine) error {
	if len(line.Content) == 0 {
		r.buf.WriteString("<div class=\"downstage-dialogue-break\"></div>\n")
		return nil
	}
	cls := "downstage-line"
	if line.IsVerse {
		cls += " downstage-verse"
	}
	fmt.Fprintf(&r.buf, "<p class=\"%s\">", cls)
	return nil
}

func (r *htmlRenderer) EndDialogueLine(line *ast.DialogueLine) error {
	if len(line.Content) == 0 {
		return nil
	}
	r.buf.WriteString("</p>\n")
	return nil
}

// --- Stage Direction ---

func (r *htmlRenderer) BeginStageDirection(sd *ast.StageDirection) error {
	r.beginBlock()
	cls := "downstage-stage-direction"
	if sd.Continuation {
		cls += " downstage-continuation"
	}
	fmt.Fprintf(&r.buf, "<p class=\"%s\"%s>", cls, r.sourceAttr(sd.NodeRange()))
	return nil
}

func (r *htmlRenderer) EndStageDirection(_ *ast.StageDirection) error {
	r.buf.WriteString("</p>\n")
	return nil
}

// --- Callout ---

func (r *htmlRenderer) BeginCallout(c *ast.Callout) error {
	r.beginBlock()
	cls := "downstage-callout"
	if c.Continuation {
		cls += " downstage-continuation"
	}
	fmt.Fprintf(&r.buf, "<p class=\"%s\"%s>", cls, r.sourceAttr(c.NodeRange()))
	return nil
}

func (r *htmlRenderer) EndCallout(_ *ast.Callout) error {
	r.buf.WriteString("</p>\n")
	return nil
}

// --- Song ---

func (r *htmlRenderer) BeginSong(song *ast.Song) error {
	r.beginBlock()
	fmt.Fprintf(&r.buf, "<div class=\"downstage-song\"%s>\n", r.sourceAttr(song.NodeRange()))

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

func (r *htmlRenderer) BeginVerseBlock(vb *ast.VerseBlock) error {
	r.beginBlock()
	fmt.Fprintf(&r.buf, "<div class=\"downstage-verse-block\"%s>\n", r.sourceAttr(vb.NodeRange()))
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

func (r *htmlRenderer) RenderPageBreak(pb *ast.PageBreak) error {
	r.beginBlock()
	fmt.Fprintf(&r.buf, "<hr class=\"downstage-page-break\"%s>\n", r.sourceAttr(pb.NodeRange()))
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

func (r *htmlRenderer) renderInlineContent(inlines []ast.Inline) {
	for _, inline := range inlines {
		switch n := inline.(type) {
		case *ast.TextNode:
			r.buf.WriteString(html.EscapeString(n.Value))
		case *ast.BoldNode:
			r.buf.WriteString("<strong>")
			r.renderInlineContent(n.Content)
			r.buf.WriteString("</strong>")
		case *ast.ItalicNode:
			r.buf.WriteString("<em>")
			r.renderInlineContent(n.Content)
			r.buf.WriteString("</em>")
		case *ast.BoldItalicNode:
			r.buf.WriteString("<strong><em>")
			r.renderInlineContent(n.Content)
			r.buf.WriteString("</em></strong>")
		case *ast.UnderlineNode:
			r.buf.WriteString("<u>")
			r.renderInlineContent(n.Content)
			r.buf.WriteString("</u>")
		case *ast.StrikethroughNode:
			r.buf.WriteString("<del>")
			r.renderInlineContent(n.Content)
			r.buf.WriteString("</del>")
		case *ast.InlineDirectionNode:
			r.dirDepth++
			if r.dirDepth == 1 {
				r.buf.WriteString("<span class=\"downstage-inline-direction\">(")
			}
			r.renderInlineContent(n.Content)
			r.dirDepth--
			if r.dirDepth == 0 {
				r.buf.WriteString(")</span>")
			}
		}
	}
}

// keyValueInlines returns the inline representation of a KeyValue's value,
// falling back to a single TextNode when the parser didn't populate
// ValueInlines (e.g. tests that build KeyValues by hand).
func keyValueInlines(kv ast.KeyValue) []ast.Inline {
	if len(kv.ValueInlines) > 0 {
		return kv.ValueInlines
	}
	if kv.Value == "" {
		return nil
	}
	return []ast.Inline{&ast.TextNode{Value: kv.Value}}
}

// hasKeyValueContent reports whether a KeyValue has any renderable value,
// checking both the legacy string and the inline form.
func hasKeyValueContent(kv ast.KeyValue) bool {
	if strings.TrimSpace(kv.Value) != "" {
		return true
	}
	return strings.TrimSpace(render.PlainText(kv.ValueInlines)) != ""
}

// characterDescriptionInlines is the same fallback for Character descriptions.
func characterDescriptionInlines(ch ast.Character) []ast.Inline {
	if len(ch.DescriptionInlines) > 0 {
		return ch.DescriptionInlines
	}
	if ch.Description == "" {
		return nil
	}
	return []ast.Inline{&ast.TextNode{Value: ch.Description}}
}

// hasCharacterDescription is the gate for rendering a description.
func hasCharacterDescription(ch ast.Character) bool {
	if strings.TrimSpace(ch.Description) != "" {
		return true
	}
	return strings.TrimSpace(render.PlainText(ch.DescriptionInlines)) != ""
}

func dialogueParentheticalInlines(d *ast.Dialogue) []ast.Inline {
	if len(d.ParentheticalInlines()) > 0 {
		return d.ParentheticalInlines()
	}
	paren := strings.TrimSpace(d.Parenthetical)
	if strings.HasPrefix(paren, "(") && strings.HasSuffix(paren, ")") {
		paren = strings.TrimSuffix(strings.TrimPrefix(paren, "("), ")")
	}
	return []ast.Inline{&ast.TextNode{Value: paren}}
}

// --- Helpers ---

func titlePageTitle(tp *ast.TitlePage) string {
	if tp == nil {
		return ""
	}
	var title string
	for _, entry := range tp.Entries {
		if strings.EqualFold(strings.TrimSpace(entry.Key), "title") {
			title = strings.TrimSpace(entry.Value)
		}
	}
	return title
}

func (r *htmlRenderer) closeParagraph() {
	if r.inParagraph {
		r.buf.WriteString("</p>\n")
		r.inParagraph = false
	}
}

func (r *htmlRenderer) beginBlock() {
	r.closeParagraph()
}

func (r *htmlRenderer) pushSection(closeTag bool) {
	r.sectionStack = append(r.sectionStack, sectionState{closeTag: closeTag})
}

func (r *htmlRenderer) popSection() sectionState {
	if len(r.sectionStack) == 0 {
		return sectionState{}
	}
	last := r.sectionStack[len(r.sectionStack)-1]
	r.sectionStack = r.sectionStack[:len(r.sectionStack)-1]
	return last
}

func headingTag(level int) string {
	if level >= 1 && level <= 6 {
		return fmt.Sprintf("h%d", level)
	}
	return "h2"
}
