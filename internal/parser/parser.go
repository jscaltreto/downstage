package parser

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/lexer"
	"github.com/jscaltreto/downstage/internal/token"
)

const (
	maxTitlePageEntries         = 256
	maxDialogueLines            = 2048
	maxInlineDelimiterLookahead = 8192
)

// Parse lexes input and produces an AST document along with any parse errors.
func Parse(input []byte) (*ast.Document, []*ParseError) {
	tokens := lexer.Lex(input)
	p := &parser{
		tokens: tokens,
		pos:    0,
		errors: make([]*ParseError, 0),
	}
	return p.parseDocument(), p.errors
}

type parser struct {
	tokens      []token.Token
	pos         int
	errors      []*ParseError
	pendingDual bool // set when the last parsed element was a DualDialogueChar dialogue
}

func (p *parser) peek() token.Token {
	if p.pos >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() token.Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *parser) at(t token.Type) bool {
	return p.peek().Type == t
}

func (p *parser) atAny(types ...token.Type) bool {
	cur := p.peek().Type
	for _, t := range types {
		if cur == t {
			return true
		}
	}
	return false
}

func (p *parser) skipBlanks() {
	for p.at(token.Blank) {
		p.advance()
	}
}

func (p *parser) skipBlanksAndComments() {
	for {
		switch {
		case p.at(token.Blank):
			p.advance()
		case p.at(token.LineComment):
			p.advance()
		case p.at(token.BlockCommentStart):
			p.parseBlockComment() // consumes start, body text, and end tokens
		default:
			return
		}
	}
}

func (p *parser) addError(msg string, r token.Range) {
	p.errors = append(p.errors, &ParseError{Message: msg, Range: r})
}

func (p *parser) addCodedError(code, msg string, r token.Range) {
	p.errors = append(p.errors, &ParseError{Message: msg, Range: r, Code: code})
}

func (p *parser) skipToNextBlank() {
	for !p.at(token.Blank) && !p.at(token.EOF) {
		p.advance()
	}
}

func (p *parser) parseDocument() *ast.Document {
	doc := &ast.Document{}

	if p.at(token.TitleKey) || p.at(token.TitleValue) {
		legacy := p.parseTitlePage()
		if legacy != nil {
			p.addError("document-level metadata is a V1 pattern; add a top-level # heading and move metadata under it", legacy.Range)
		}
	}

	p.skipBlanksAndComments()

	// Body
	doc.Body = p.parseBody()

	// Set document range
	if len(p.tokens) > 0 {
		doc.Range = token.Range{
			Start: p.tokens[0].Range.Start,
			End:   p.tokens[len(p.tokens)-1].Range.End,
		}
	}

	return doc
}

func (p *parser) parseTitlePage() *ast.TitlePage {
	tp := &ast.TitlePage{}
	startRange := p.peek().Range

	for p.at(token.TitleKey) || p.at(token.TitleValue) {
		if p.at(token.TitleKey) {
			if len(tp.Entries) >= maxTitlePageEntries {
				p.addError("title page exceeds maximum entry count", p.peek().Range)
				p.skipTitlePageOverflow()
				break
			}

			keyTok := p.advance()
			kv := ast.KeyValue{
				Key:   keyTok.Literal,
				Range: keyTok.Range,
			}
			if p.at(token.TitleValue) {
				valTok := p.advance()
				kv.Value = valTok.Literal
				kv.Range.End = valTok.Range.End
			}
			tp.Entries = append(tp.Entries, kv)
		} else {
			// Continuation value without key; append to last entry
			valTok := p.advance()
			if len(tp.Entries) > 0 {
				last := &tp.Entries[len(tp.Entries)-1]
				if last.Value != "" {
					last.Value += "\n"
				}
				last.Value += valTok.Literal
				last.Range.End = valTok.Range.End
			}
		}
	}

	tp.Range = startRange
	if len(tp.Entries) > 0 {
		tp.Range.End = tp.Entries[len(tp.Entries)-1].Range.End
	}

	return tp
}

func (p *parser) parseBody() []ast.Node {
	var nodes []ast.Node
	prevContinuation := token.EOF

	for !p.at(token.EOF) {
		hadBlanks := p.at(token.Blank)
		p.skipBlanks()
		if p.at(token.EOF) {
			break
		}

		switch p.peek().Type {
		case token.HeadingH1:
			node := p.parseSection(1)
			if node != nil {
				nodes = append(nodes, node)
			}
			prevContinuation = token.EOF
		case token.LineComment:
			nodes = append(nodes, p.parseLineComment())
		case token.BlockCommentStart:
			nodes = append(nodes, p.parseBlockComment())
		case token.PageBreak:
			tok := p.advance()
			nodes = append(nodes, &ast.PageBreak{Range: tok.Range})
			prevContinuation = token.EOF
		default:
			// Body-only document: accept dialogue, stage directions, songs,
			// etc. directly. Metadata is still required to live under a `#`
			// heading (V1 document-level metadata is flagged separately).
			elem := p.parseBodyElement()
			if elem != nil {
				prevContinuation = markContinuation(elem, prevContinuation, hadBlanks)
				nodes = append(nodes, elem)
				nodes = p.makeDualDialogue(nodes)
			}
		}
	}

	return nodes
}

// markContinuation sets Continuation on adjacent callouts and stage directions.
func markContinuation(node ast.Node, prev token.Type, hadBlanks bool) token.Type {
	switch n := node.(type) {
	case *ast.StageDirection:
		if prev == token.StageDirection && !hadBlanks {
			n.Continuation = true
		}
		return token.StageDirection
	case *ast.Callout:
		if prev == token.Callout && !hadBlanks {
			n.Continuation = true
		}
		return token.Callout
	default:
		return token.EOF
	}
}

// makeDualDialogue checks if the parser just produced a dual-dialogue right-side
// dialogue. If so, it pairs the last two nodes in the slice into a DualDialogue.
// Returns the updated slice.
func (p *parser) makeDualDialogue(nodes []ast.Node) []ast.Node {
	if !p.pendingDual {
		return nodes
	}
	p.pendingDual = false

	n := len(nodes)
	if n < 2 {
		return nodes
	}

	right, ok := nodes[n-1].(*ast.Dialogue)
	if !ok {
		return nodes
	}
	left, ok := nodes[n-2].(*ast.Dialogue)
	if !ok {
		return nodes
	}

	dual := &ast.DualDialogue{
		Left:  left,
		Right: right,
		Range: token.Range{
			Start: left.Range.Start,
			End:   right.Range.End,
		},
	}
	nodes[n-2] = dual
	return nodes[:n-1]
}

// maybeDualDialogueSection wraps the last two children of a Section into a
// DualDialogue if the parser just processed a DualDialogueChar token.
func (p *parser) maybeDualDialogueSection(section *ast.Section) {
	if !p.pendingDual {
		return
	}
	p.pendingDual = false

	n := len(section.Children)
	if n < 2 {
		return
	}

	right, ok := section.Children[n-1].(*ast.Dialogue)
	if !ok {
		return
	}
	left, ok := section.Children[n-2].(*ast.Dialogue)
	if !ok {
		return
	}

	dual := &ast.DualDialogue{
		Left:  left,
		Right: right,
		Range: token.Range{
			Start: left.Range.Start,
			End:   right.Range.End,
		},
	}
	section.WrapLastTwoChildren(dual)
}

func (p *parser) parseBodyElement() ast.Node {
	switch p.peek().Type {
	case token.HeadingH1:
		return p.parseSection(1)

	case token.HeadingH2:
		return p.parseSection(2)

	case token.HeadingH3:
		return p.parseSection(3)

	case token.CharacterName, token.ForcedCharacter:
		p.pendingDual = false
		return p.parseDialogue()

	case token.DualDialogueChar:
		p.pendingDual = true
		return p.parseDialogue()

	case token.StageDirection:
		return p.parseStageDirection()

	case token.Callout:
		return p.parseCallout()

	case token.SongStart:
		return p.parseSong()

	case token.LineComment:
		return p.parseLineComment()

	case token.BlockCommentStart:
		return p.parseBlockComment()

	case token.PageBreak:
		tok := p.advance()
		return &ast.PageBreak{Range: tok.Range}

	case token.ForcedHeading:
		tok := p.advance()
		return &ast.Section{
			Kind:  ast.SectionGeneric,
			Level: 0, // inline heading, no page break
			Title: strings.TrimPrefix(tok.Literal, "."),
			Range: tok.Range,
		}

	case token.Verse:
		return p.parseVerseBlock()

	case token.Text:
		tok := p.advance()
		return &ast.StageDirection{
			Content: parseInlineContent(tok.Literal, tok.Range),
			Range:   tok.Range,
		}

	case token.CharacterAlias:
		tok := p.advance()
		return &ast.StageDirection{
			Content: parseInlineContent(tok.Literal, tok.Range),
			Range:   tok.Range,
		}

	default:
		// Unexpected token; emit error and skip
		tok := p.advance()
		p.addError("unexpected token: "+tok.Type.String(), tok.Range)
		p.skipToNextBlank()
		return nil
	}
}

// headingTokenForLevel returns the token type for a given heading level.
func headingTokenForLevel(level int) token.Type {
	switch level {
	case 1:
		return token.HeadingH1
	case 2:
		return token.HeadingH2
	case 3:
		return token.HeadingH3
	default:
		return token.HeadingH1
	}
}

// hasSubHeading scans ahead (without consuming) to see if there are any
// headings below the given level before the next same-or-higher heading.
// This determines whether a generic section contains structural content
// (acts/scenes) or is pure prose.
func (p *parser) hasSubHeading(level int) bool {
	saved := p.pos
	defer func() { p.pos = saved }()

	for !p.at(token.EOF) && !p.atHeadingAtOrAboveLevel(level) {
		if level < 3 && p.at(headingTokenForLevel(level+1)) {
			return true
		}
		if level < 2 && p.at(headingTokenForLevel(level+2)) {
			return true
		}
		p.advance()
	}
	return false
}

func (p *parser) hasStructuralBodyContent(level int) bool {
	saved := p.pos
	defer func() { p.pos = saved }()

	for !p.at(token.EOF) && !p.atHeadingAtOrAboveLevel(level) {
		if p.atAny(token.Blank, token.LineComment, token.BlockCommentStart) {
			p.advance()
			continue
		}
		if p.atAny(
			token.CharacterName,
			token.ForcedCharacter,
			token.DualDialogueChar,
			token.StageDirection,
			token.Callout,
			token.SongStart,
			token.Verse,
			token.PageBreak,
			token.ForcedHeading,
		) {
			return true
		}
		if level < 3 && p.at(headingTokenForLevel(level+1)) {
			return true
		}
		if level < 2 && p.at(headingTokenForLevel(level+2)) {
			return true
		}
		p.advance()
	}
	return false
}

// atHeadingLevel returns true if the current token is a heading at or above the given level.
func (p *parser) atHeadingAtOrAboveLevel(level int) bool {
	switch level {
	case 1:
		return p.at(token.HeadingH1)
	case 2:
		return p.at(token.HeadingH1) || p.at(token.HeadingH2)
	case 3:
		return p.at(token.HeadingH1) || p.at(token.HeadingH2) || p.at(token.HeadingH3)
	default:
		return false
	}
}

func inferSectionKind(title string, level int, insideAct bool) ast.SectionKind {
	upper := strings.ToUpper(strings.TrimSpace(title))

	if level == 1 {
		return ast.SectionGeneric
	}

	normalized := strings.ToLower(strings.TrimSpace(title))
	switch normalized {
	case "dramatis personae", "cast of characters", "characters":
		return ast.SectionDramatisPersonae
	}
	if isSectionKeyword(upper, "ACT") {
		return ast.SectionAct
	}
	if isSectionKeyword(upper, "SCENE") {
		return ast.SectionScene
	}
	if insideAct && level >= 2 {
		return ast.SectionScene
	}
	return ast.SectionGeneric
}

// parseActTitle parses "ACT I: The Beginning" into number and title.
func parseActTitle(raw string) (number, title string) {
	upper := strings.ToUpper(raw)
	rest := strings.TrimSpace(strings.TrimPrefix(upper, "ACT"))
	if idx := strings.Index(rest, ":"); idx >= 0 {
		number = strings.TrimSpace(rest[:idx])
		// Extract original-case title after colon
		origRest := strings.TrimSpace(raw[len("ACT"):])
		if colonIdx := strings.Index(origRest, ":"); colonIdx >= 0 {
			title = strings.TrimSpace(origRest[colonIdx+1:])
		}
	} else {
		number = rest
	}
	return
}

// parseSceneTitle parses "SCENE 1: The Garden" into number and title.
func parseSceneTitle(raw string) (number, title string) {
	upper := strings.ToUpper(raw)
	rest := strings.TrimSpace(strings.TrimPrefix(upper, "SCENE"))
	if idx := strings.Index(rest, ":"); idx >= 0 {
		number = strings.TrimSpace(rest[:idx])
		origRest := strings.TrimSpace(raw[len("SCENE"):])
		if colonIdx := strings.Index(origRest, ":"); colonIdx >= 0 {
			title = strings.TrimSpace(origRest[colonIdx+1:])
		}
	} else {
		number = rest
	}
	return
}

// parseSection parses a unified Section node from a heading token at the given level.
// When insideAct is true, ## headings default to SectionScene.
func (p *parser) parseSection(level int) *ast.Section {
	return p.parseSectionInContext(level, false)
}

func (p *parser) parseSectionInContext(level int, insideAct bool) *ast.Section {
	headingTok := p.advance()
	kind := inferSectionKind(headingTok.Literal, level, insideAct)
	if level == 1 {
		normalized := strings.ToLower(strings.TrimSpace(headingTok.Literal))
		switch normalized {
		case "dramatis personae", "cast of characters", "characters":
			p.addError("top-level Dramatis Personae is a V1 pattern; move it under the owning # section as ## Dramatis Personae", headingTok.Range)
		}
	}

	section := &ast.Section{
		Kind:  kind,
		Level: level,
		Range: headingTok.Range,
	}
	section.SetHeadingRange(headingTok.Range)

	switch kind {
	case ast.SectionAct:
		section.Number, section.Title = parseActTitle(headingTok.Literal)
	case ast.SectionScene:
		upper := strings.ToUpper(headingTok.Literal)
		if strings.HasPrefix(upper, "SCENE") {
			section.Number, section.Title = parseSceneTitle(headingTok.Literal)
		} else {
			section.Title = headingTok.Literal
		}
	case ast.SectionDramatisPersonae:
		section.Title = headingTok.Literal
	case ast.SectionGeneric:
		section.Title = headingTok.Literal
	}

	p.skipBlanks()
	if level == 1 {
		section.Metadata = p.parseSectionMetadata()
		p.skipBlanks()
	}

	switch kind {
	case ast.SectionDramatisPersonae:
		p.parseDPContent(section)
	case ast.SectionAct:
		p.parseActContent(section)
	case ast.SectionScene:
		p.parseSceneContent(section, level)
	case ast.SectionGeneric:
		p.parseGenericContent(section, level)
	}

	// Set end range from last consumed token
	if p.pos > 0 {
		section.Range.End = p.tokens[p.pos-1].Range.End
	}

	return section
}

// parseDPContent fills Characters and Groups for a Dramatis Personae section.
// Stops at the next heading at or above the section level, or EOF.
func (p *parser) parseDPContent(section *ast.Section) {
	var currentGroup *ast.CharacterGroup
	groupLevel := section.Level + 1

	for !p.at(token.EOF) && !p.atHeadingAtOrAboveLevel(section.Level) {
		switch p.peek().Type {
		case headingTokenForLevel(groupLevel):
			// Sub-heading becomes a character group
			tok := p.advance()
			if currentGroup != nil {
				section.Groups = append(section.Groups, *currentGroup)
			}
			currentGroup = &ast.CharacterGroup{
				Name:  tok.Literal,
				Range: tok.Range,
			}
			p.skipBlanks()

		case token.CharacterName:
			// ALL-CAPS name, possibly with description on same line
			tok := p.advance()
			if hasUnsupportedDPDash(tok.Literal) {
				p.addCodedError(ErrCodeDPUnicodeDash, "character descriptions in Dramatis Personae must use ASCII ` - `", tok.Range)
			}
			ch := parseCharacterEntry(tok)
			if currentGroup != nil {
				currentGroup.Characters = append(currentGroup.Characters, ch)
				currentGroup.Range.End = tok.Range.End
			} else {
				section.Characters = append(section.Characters, ch)
			}
			section.Range.End = tok.Range.End

		case token.CharacterAlias:
			tok := p.advance()
			p.addCodedError(ErrCodeDPStandaloneAlias, "standalone character alias syntax is not supported; use NAME/ALIAS inline", tok.Range)

		case token.Text:
			tok := p.advance()
			if hasUnsupportedDPDash(tok.Literal) {
				p.addCodedError(ErrCodeDPUnicodeDash, "character descriptions in Dramatis Personae must use ASCII ` - `", tok.Range)
			}
			ch := parseCharacterEntry(tok)
			if currentGroup != nil {
				currentGroup.Characters = append(currentGroup.Characters, ch)
				currentGroup.Range.End = tok.Range.End
			} else {
				section.Characters = append(section.Characters, ch)
			}
			section.Range.End = tok.Range.End

		case token.Blank:
			p.advance()

		case token.LineComment:
			p.advance()

		case token.BlockCommentStart:
			p.parseBlockComment()

		default:
			// Drop tokens we can't interpret as a DP entry (e.g. stray verse
			// lines, unexpected headings at the wrong level). Advancing keeps
			// us making forward progress; the surrounding surfaces will flag
			// anything semantically important elsewhere.
			p.advance()
		}
	}

	if currentGroup != nil {
		section.Groups = append(section.Groups, *currentGroup)
	}
}

func (p *parser) parseSectionMetadata() *ast.TitlePage {
	saved := p.pos
	// Comments and blanks between the heading and the metadata block are
	// transparent — consume them to find the first metadata line. If no
	// metadata is present we restore position so the caller sees them as
	// regular section content.
	p.skipBlanksAndComments()
	if !p.atAny(token.Text, token.Verse) {
		p.pos = saved
		return nil
	}

	tp := &ast.TitlePage{}
	start := token.Range{}

	for !p.at(token.EOF) {
		if p.at(token.Blank) || p.atAny(token.HeadingH1, token.HeadingH2, token.HeadingH3, token.PageBreak) {
			break
		}
		if !p.atAny(token.Text, token.Verse) {
			break
		}

		raw := p.peek().Literal
		if key, value, ok := parseMetadataEntry(raw); ok {
			tok := p.advance()
			if len(tp.Entries) == 0 {
				start = tok.Range
			}
			tp.Entries = append(tp.Entries, ast.KeyValue{
				Key:   key,
				Value: value,
				Range: tok.Range,
			})
			continue
		}

		if value, ok := parseMetadataContinuation(raw); ok {
			if len(tp.Entries) == 0 {
				p.pos = saved
				return nil
			}
			tok := p.advance()
			last := &tp.Entries[len(tp.Entries)-1]
			if last.Value != "" {
				last.Value += "\n"
			}
			last.Value += value
			last.Range.End = tok.Range.End
			continue
		}

		break
	}

	if len(tp.Entries) == 0 {
		p.pos = saved
		return nil
	}

	tp.Range = start
	tp.Range.End = tp.Entries[len(tp.Entries)-1].Range.End
	return tp
}

func parseMetadataEntry(raw string) (key, value string, ok bool) {
	idx := strings.Index(raw, ":")
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(raw[:idx])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(raw[idx+1:]), true
}

func parseMetadataContinuation(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	if raw[0] != ' ' && raw[0] != '\t' {
		return "", false
	}
	return strings.TrimSpace(raw), true
}

// parseActContent fills Children for an act section with scenes and content.
// Stops at next H1 or EOF.
func (p *parser) parseActContent(section *ast.Section) {
	prevContinuation := token.EOF
	for !p.at(token.EOF) && !p.at(token.HeadingH1) {
		if p.at(token.HeadingH2) {
			// Check if it's a new ACT (breaks out)
			nextLit := strings.ToUpper(p.peek().Literal)
			if strings.HasPrefix(nextLit, "ACT") {
				break
			}
			// Sub-section inside act: parse as level 2, with insideAct=true
			child := p.parseSectionInContext(2, true)
			section.AppendChild(child)
			prevContinuation = token.EOF
			continue
		}

		if p.at(token.HeadingH3) {
			child := p.parseSectionInContext(3, true)
			section.AppendChild(child)
			prevContinuation = token.EOF
			continue
		}

		hadBlanks := p.at(token.Blank)
		p.skipBlanks()
		if p.at(token.EOF) || p.at(token.HeadingH1) {
			break
		}
		if p.at(token.HeadingH2) || p.at(token.HeadingH3) {
			continue
		}

		elem := p.parseBodyElement()
		if elem != nil {
			prevContinuation = markContinuation(elem, prevContinuation, hadBlanks)
			section.AppendChild(elem)
			p.maybeDualDialogueSection(section)
		}
	}
}

// parseSceneContent fills Children for a scene section.
// Stops at a heading at or above the given level, or EOF.
func (p *parser) parseSceneContent(section *ast.Section, level int) {
	prevContinuation := token.EOF
	for !p.at(token.EOF) && !p.atHeadingAtOrAboveLevel(level) {
		hadBlanks := p.at(token.Blank)
		p.skipBlanks()
		if p.at(token.EOF) || p.atHeadingAtOrAboveLevel(level) {
			break
		}

		// Nested headings below this level become children
		if level < 3 && p.at(headingTokenForLevel(level+1)) {
			child := p.parseSectionInContext(level+1, true)
			section.AppendChild(child)
			prevContinuation = token.EOF
			continue
		}

		elem := p.parseBodyElement()
		if elem != nil {
			prevContinuation = markContinuation(elem, prevContinuation, hadBlanks)
			section.AppendChild(elem)
			p.maybeDualDialogueSection(section)
		}
	}
}

// parseGenericContent fills Children and Lines for a generic section.
// Sub-headings become nested Children. Structural content (dialogue, stage
// directions, songs, etc.) also becomes Children. Plain text becomes Lines.
// Stops at a heading at or above the given level, or EOF.
func (p *parser) parseGenericContent(section *ast.Section, level int) {
	// Scan ahead to determine if this section contains structural sub-headings.
	// If so, text/stage directions are structural content (Children).
	// If not, they're prose lines (Lines) with paragraph reflow.
	hasStructuralContent := p.hasStructuralBodyContent(level)
	prevContinuation := token.EOF

	for !p.at(token.EOF) && !p.atHeadingAtOrAboveLevel(level) {
		// Sub-headings become child sections
		if level < 3 && p.at(headingTokenForLevel(level+1)) {
			child := p.parseSectionInContext(level+1, false)
			section.AppendChild(child)
			prevContinuation = token.EOF
			continue
		}
		if level < 2 && p.at(headingTokenForLevel(level+2)) {
			child := p.parseSectionInContext(level+2, false)
			section.AppendChild(child)
			prevContinuation = token.EOF
			continue
		}

		// Text becomes prose in leaf generic sections. Demoted ALL-CAPS lines
		// are promoted to stage directions so they still render as italic text.
		if p.at(token.Text) && !hasStructuralContent {
			if lexer.IsCharacterName(strings.TrimSpace(p.peek().Literal)) {
				tok := p.advance()
				section.AppendChild(&ast.StageDirection{
					Content: parseInlineContent(tok.Literal, tok.Range),
					Range:   tok.Range,
				})
				prevContinuation = token.EOF
				continue
			}
			tok := p.advance()
			line := ast.SectionLine{
				Content: parseInlineContent(tok.Literal, tok.Range),
				Range:   tok.Range,
			}
			section.AppendLine(line)
			prevContinuation = token.EOF
			continue
		}

		// Structural content goes into Children
		if p.atAny(token.CharacterName, token.ForcedCharacter, token.DualDialogueChar,
			token.StageDirection, token.Callout, token.SongStart, token.Verse, token.PageBreak,
			token.ForcedHeading, token.Text, token.CharacterAlias) {
			hadBlanks := false // blanks already consumed by the Blank handler below
			elem := p.parseBodyElement()
			if elem != nil {
				prevContinuation = markContinuation(elem, prevContinuation, hadBlanks)
				section.AppendChild(elem)
				p.maybeDualDialogueSection(section)
			}
			continue
		}

		// Comments go into Children
		if p.at(token.LineComment) {
			section.AppendChild(p.parseLineComment())
			continue
		}
		if p.at(token.BlockCommentStart) {
			section.AppendChild(p.parseBlockComment())
			continue
		}

		if p.at(token.Blank) {
			tok := p.advance()
			if hasStructuralContent {
				_ = tok
				prevContinuation = token.EOF
				continue
			}
			// Preserve blank lines as empty section lines (paragraph breaks)
			section.AppendLine(ast.SectionLine{Range: tok.Range})
			prevContinuation = token.EOF
			continue
		}

		// Any other token becomes a text line with inline formatting
		tok := p.advance()
		line := ast.SectionLine{
			Content: parseInlineContent(tok.Literal, tok.Range),
			Range:   tok.Range,
		}
		section.AppendLine(line)
		prevContinuation = token.EOF
	}

	// Trim trailing blank lines
	section.TrimTrailingBlankLines()
}

func parseCharacterEntry(tok token.Token) ast.Character {
	ch := ast.Character{Range: tok.Range}
	namePart := tok.Literal
	if idx := strings.Index(tok.Literal, " - "); idx >= 0 {
		namePart = strings.TrimSpace(tok.Literal[:idx])
		ch.Description = strings.TrimSpace(tok.Literal[idx+3:])
	}
	ch.Name, ch.Aliases = parseAliasSpec(strings.TrimSpace(namePart))
	return ch
}

func hasUnsupportedDPDash(raw string) bool {
	return strings.Contains(raw, " — ") || strings.Contains(raw, " – ")
}

func (p *parser) parseDialogue() *ast.Dialogue {
	nameTok := p.advance()
	dlg := &ast.Dialogue{
		Character: nameTok.Literal,
		Range:     nameTok.Range,
	}
	dlg.SetNameRange(nameTok.Range)

	// Strip @ prefix for forced characters (including dual dialogue forced characters)
	if nameTok.Type == token.ForcedCharacter ||
		(nameTok.Type == token.DualDialogueChar && strings.HasPrefix(nameTok.Literal, "@")) {
		dlg.Character = strings.TrimPrefix(nameTok.Literal, "@")
		dlg.SetNameRange(shiftRangeStart(nameTok.Range, 1, 1))
		dlg.Forced = true
	}

	// Check for parenthetical right after character name: (text)
	if p.at(token.Text) {
		lit := strings.TrimSpace(p.peek().Literal)
		if strings.HasPrefix(lit, "(") && strings.HasSuffix(lit, ")") {
			pTok := p.advance()
			dlg.Parenthetical = strings.TrimSpace(pTok.Literal)
			inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(dlg.Parenthetical, "("), ")"))
			if inner != "" {
				innerRange := sliceInlineRange(dlg.Parenthetical, pTok.Range, 1, len(dlg.Parenthetical)-1)
				dlg.SetParentheticalInlines(parseInlineContent(inner, innerRange))
			}
			dlg.SetParentheticalRange(pTok.Range)
		}
	}

	// Collect dialogue lines until structural break
loop:
	for !p.at(token.EOF) {
		if p.at(token.Blank) {
			// Stop if the blank leads into another cue or structural break.
			saved := p.pos
			p.skipBlanks()
			if p.atAny(token.CharacterName, token.ForcedCharacter, token.DualDialogueChar,
				token.HeadingH1, token.HeadingH2, token.HeadingH3, token.SongStart,
				token.SongEnd, token.PageBreak, token.EOF, token.StageDirection,
				token.BlockCommentStart) {
				p.pos = saved // restore; let caller handle the blank
				break
			}
			// Also break on blank followed by blank (paragraph break)
			if p.at(token.Blank) {
				p.pos = saved
				break
			}
			// Single blank line continues dialogue.
			blankRange := p.tokens[saved].Range
			p.pos = saved
			p.skipBlanks()
			if len(dlg.Lines) >= maxDialogueLines {
				p.addError("dialogue exceeds maximum line count", p.peek().Range)
				p.skipDialogueContent()
				break loop
			}
			dlg.Lines = append(dlg.Lines, ast.DialogueLine{Range: blankRange})
		}

		switch p.peek().Type {
		case token.Text:
			if len(dlg.Lines) >= maxDialogueLines {
				p.addError("dialogue exceeds maximum line count", p.peek().Range)
				p.skipDialogueContent()
				break loop
			}
			tok := p.advance()
			line := ast.DialogueLine{
				Content: parseInlineContent(tok.Literal, tok.Range),
				Range:   tok.Range,
			}
			dlg.Lines = append(dlg.Lines, line)

		case token.Verse:
			if len(dlg.Lines) >= maxDialogueLines {
				p.addError("dialogue exceeds maximum line count", p.peek().Range)
				p.skipDialogueContent()
				break loop
			}
			tok := p.advance()
			trimmed := strings.TrimLeft(tok.Literal, " ")
			leadingSpaces := len(tok.Literal) - len(trimmed)
			line := ast.DialogueLine{
				Content: parseInlineContent(trimmed, shiftRangeStart(tok.Range, leadingSpaces, leadingSpaces)),
				IsVerse: true,
				Range:   tok.Range,
			}
			dlg.Lines = append(dlg.Lines, line)

		case token.StageDirection:
			if len(dlg.Lines) >= maxDialogueLines {
				p.addError("dialogue exceeds maximum line count", p.peek().Range)
				p.skipDialogueContent()
				break loop
			}
			tok := p.advance()
			line := ast.DialogueLine{
				Content: []ast.Inline{
					&ast.InlineDirectionNode{
						Content: parseInlineContent(tok.Literal, tok.Range),
						Range:   tok.Range,
					},
				},
				Range: tok.Range,
			}
			dlg.Lines = append(dlg.Lines, line)

		case token.LineComment:
			// Skip comments within dialogue.
			p.advance()

		case token.BlockCommentStart:
			// Block comments are transparent within dialogue.
			p.parseBlockComment()

		default:
			break loop
		}
	}

	if len(dlg.Lines) > 0 {
		dlg.Range.End = dlg.Lines[len(dlg.Lines)-1].Range.End
	}
	return dlg
}

func (p *parser) parseStageDirection() *ast.StageDirection {
	tok := p.advance()
	return &ast.StageDirection{
		Content: parseInlineContent(tok.Literal, tok.Range),
		Range:   tok.Range,
	}
}

func (p *parser) parseCallout() *ast.Callout {
	tok := p.advance()
	return &ast.Callout{
		Content: parseInlineContent(tok.Literal, tok.Range),
		Range:   tok.Range,
	}
}

func (p *parser) parseSong() *ast.Song {
	startTok := p.advance()
	song := &ast.Song{Range: startTok.Range}

	// Parse song number and title from formats:
	//   SONG
	//   SONG: Title
	//   SONG 1: Title
	//   SONG 1 Title
	rest := strings.TrimPrefix(startTok.Literal, "SONG")
	rest = strings.TrimSpace(rest)
	if rest != "" {
		if strings.HasPrefix(rest, ":") {
			// "SONG: Title" — no number, just title
			song.Title = strings.TrimSpace(rest[1:])
		} else if idx := strings.Index(rest, ":"); idx >= 0 {
			// "SONG 1: Title" — number before colon, title after
			song.Number = strings.TrimSpace(rest[:idx])
			song.Title = strings.TrimSpace(rest[idx+1:])
		} else {
			// "SONG 1 Title" or "SONG 1" — first word is number, rest is title
			parts := strings.SplitN(rest, " ", 2)
			song.Number = parts[0]
			if len(parts) > 1 {
				song.Title = parts[1]
			}
		}
	}

	p.skipBlanks()

	// Collect content until SONG END
	prevContinuation := token.EOF
	for !p.at(token.SongEnd) && !p.at(token.EOF) {
		hadBlanks := p.at(token.Blank)
		p.skipBlanks()
		if p.at(token.SongEnd) || p.at(token.EOF) {
			break
		}

		elem := p.parseBodyElement()
		if elem != nil {
			prevContinuation = markContinuation(elem, prevContinuation, hadBlanks)
			song.Content = append(song.Content, elem)
			song.Content = p.makeDualDialogue(song.Content)
		}
	}

	if p.at(token.SongEnd) {
		endTok := p.advance()
		song.SetEndRange(endTok.Range)
		song.Range.End = endTok.Range.End
	} else {
		p.addError("unterminated SONG block", startTok.Range)
	}

	return song
}

func (p *parser) parseVerseBlock() *ast.VerseBlock {
	vb := &ast.VerseBlock{Range: p.peek().Range}

	for p.at(token.Verse) {
		tok := p.advance()
		trimmed := strings.TrimLeft(tok.Literal, " ")
		leadingSpaces := len(tok.Literal) - len(trimmed)
		vb.Lines = append(vb.Lines, ast.VerseLine{
			Content: parseInlineContent(trimmed, shiftRangeStart(tok.Range, leadingSpaces, leadingSpaces)),
			Range:   tok.Range,
		})
	}

	if len(vb.Lines) > 0 {
		vb.Range.End = vb.Lines[len(vb.Lines)-1].Range.End
	}
	return vb
}

func (p *parser) parseLineComment() *ast.Comment {
	tok := p.advance()
	text := strings.TrimPrefix(tok.Literal, "//")
	text = strings.TrimSpace(text)
	return &ast.Comment{
		Text:  text,
		Block: false,
		Range: tok.Range,
	}
}

func (p *parser) parseBlockComment() *ast.Comment {
	startTok := p.advance() // BlockCommentStart
	var textParts []string

	content := strings.TrimPrefix(startTok.Literal, "/*")
	content = strings.TrimSpace(content)
	if strings.HasSuffix(content, "*/") {
		content = strings.TrimSuffix(content, "*/")
		content = strings.TrimSpace(content)
	}
	if content != "" {
		textParts = append(textParts, content)
	}

	endRange := startTok.Range

	for !p.at(token.BlockCommentEnd) && !p.at(token.EOF) {
		tok := p.advance()
		textParts = append(textParts, tok.Literal)
	}

	if p.at(token.BlockCommentEnd) {
		endTok := p.advance()
		endContent := strings.TrimSuffix(endTok.Literal, "*/")
		endContent = strings.TrimSpace(endContent)
		if endContent != "" {
			textParts = append(textParts, endContent)
		}
		endRange = endTok.Range
	} else {
		p.addError("unterminated block comment", startTok.Range)
	}

	return &ast.Comment{
		Text:  strings.Join(textParts, "\n"),
		Block: true,
		Range: token.Range{Start: startTok.Range.Start, End: endRange.End},
	}
}

// parseInlineContent converts a string into inline AST nodes,
// handling bold, italic, underline, strikethrough, and inline directions.
func parseInlineContent(s string, r token.Range) []Inline {
	return parseInlines(s, r)
}

// Inline is a convenience alias used only in the return type of parseInlineContent.
type Inline = ast.Inline

func parseInlines(s string, r token.Range) []ast.Inline {
	var result []ast.Inline
	i := 0

	for i < len(s) {
		switch {
		// Bold italic: ***text***
		case i+2 < len(s) && s[i] == '*' && s[i+1] == '*' && s[i+2] == '*':
			end := findInlineDelimiter(s, i+3, "***")
			if end >= 0 {
				inner := s[i+3 : i+3+end]
				nodeRange := sliceInlineRange(s, r, i, i+3+end+3)
				result = append(result, &ast.BoldItalicNode{
					Content: []ast.Inline{&ast.TextNode{Value: inner, Range: sliceInlineRange(s, r, i+3, i+3+end)}},
					Range:   nodeRange,
				})
				i = i + 3 + end + 3
				continue
			}
			result = appendText(result, "*", r)
			i++

		// Bold: **text**
		case i+1 < len(s) && s[i] == '*' && s[i+1] == '*':
			end := findInlineDelimiter(s, i+2, "**")
			if end >= 0 {
				inner := s[i+2 : i+2+end]
				nodeRange := sliceInlineRange(s, r, i, i+2+end+2)
				result = append(result, &ast.BoldNode{
					Content: []ast.Inline{&ast.TextNode{Value: inner, Range: sliceInlineRange(s, r, i+2, i+2+end)}},
					Range:   nodeRange,
				})
				i = i + 2 + end + 2
				continue
			}
			result = appendText(result, "*", r)
			i++

		// Italic: *text*
		case s[i] == '*':
			end := findInlineDelimiter(s, i+1, "*")
			if end >= 0 {
				inner := s[i+1 : i+1+end]
				nodeRange := sliceInlineRange(s, r, i, i+1+end+1)
				result = append(result, &ast.ItalicNode{
					Content: []ast.Inline{&ast.TextNode{Value: inner, Range: sliceInlineRange(s, r, i+1, i+1+end)}},
					Range:   nodeRange,
				})
				i = i + 1 + end + 1
				continue
			}
			result = appendText(result, "*", r)
			i++

		// Underline: _text_
		case s[i] == '_':
			end := findInlineDelimiter(s, i+1, "_")
			if end >= 0 {
				inner := s[i+1 : i+1+end]
				nodeRange := sliceInlineRange(s, r, i, i+1+end+1)
				result = append(result, &ast.UnderlineNode{
					Content: []ast.Inline{&ast.TextNode{Value: inner, Range: sliceInlineRange(s, r, i+1, i+1+end)}},
					Range:   nodeRange,
				})
				i = i + 1 + end + 1
				continue
			}
			result = appendText(result, "_", r)
			i++

		// Strikethrough: ~text~
		case s[i] == '~':
			end := findInlineDelimiter(s, i+1, "~")
			if end >= 0 {
				inner := s[i+1 : i+1+end]
				nodeRange := sliceInlineRange(s, r, i, i+1+end+1)
				result = append(result, &ast.StrikethroughNode{
					Content: []ast.Inline{&ast.TextNode{Value: inner, Range: sliceInlineRange(s, r, i+1, i+1+end)}},
					Range:   nodeRange,
				})
				i = i + 1 + end + 1
				continue
			}
			result = appendText(result, "~", r)
			i++

		// Inline direction: (text)
		case s[i] == '(':
			end := findInlineDelimiter(s, i+1, ")")
			if end >= 0 {
				inner := s[i+1 : i+1+end]
				nodeRange := sliceInlineRange(s, r, i, i+1+end+1)
				result = append(result, &ast.InlineDirectionNode{
					Content: parseInlineContent(inner, sliceInlineRange(s, r, i+1, i+1+end)),
					Range:   nodeRange,
				})
				i = i + 1 + end + 1
				continue
			}
			result = appendText(result, "(", r)
			i++

		default:
			// Collect plain text until next special character
			j := i + 1
			for j < len(s) && s[j] != '*' && s[j] != '_' && s[j] != '~' && s[j] != '(' {
				j++
			}
			result = appendText(result, s[i:j], r)
			i = j
		}
	}

	return result
}

func isSectionKeyword(title, keyword string) bool {
	if title == keyword {
		return true
	}
	if strings.HasPrefix(title, keyword+" ") || strings.HasPrefix(title, keyword+":") {
		return true
	}
	return false
}

func parseAliasSpec(raw string) (string, []string) {
	parts := strings.Split(raw, "/")
	if len(parts) == 0 {
		return "", nil
	}

	name := strings.TrimSpace(parts[0])
	var aliases []string
	for _, part := range parts[1:] {
		alias := strings.TrimSpace(part)
		if alias == "" {
			continue
		}
		aliases = appendUniqueAliases(aliases, alias)
	}
	return name, aliases
}

func (p *parser) skipTitlePageOverflow() {
	for p.at(token.TitleKey) || p.at(token.TitleValue) {
		p.advance()
	}
}

func (p *parser) skipDialogueContent() {
	for !p.at(token.EOF) {
		if p.at(token.Blank) {
			saved := p.pos
			p.skipBlanks()
			if p.atAny(token.CharacterName, token.ForcedCharacter, token.DualDialogueChar,
				token.HeadingH1, token.HeadingH2, token.HeadingH3, token.SongStart,
				token.SongEnd, token.PageBreak, token.EOF, token.StageDirection, token.Callout) || p.at(token.Blank) {
				p.pos = saved
				return
			}
			p.pos = saved
			p.skipBlanks()
		}

		switch p.peek().Type {
		case token.Text, token.Verse, token.StageDirection, token.LineComment:
			p.advance()
		default:
			return
		}
	}
}

func findInlineDelimiter(s string, start int, delimiter string) int {
	if start >= len(s) {
		return -1
	}

	end := start + maxInlineDelimiterLookahead
	if end > len(s) {
		end = len(s)
	}

	return strings.Index(s[start:end], delimiter)
}

func appendUniqueAliases(existing []string, aliases ...string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, alias := range existing {
		seen[strings.ToUpper(alias)] = struct{}{}
	}
	for _, alias := range aliases {
		key := strings.ToUpper(strings.TrimSpace(alias))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, strings.TrimSpace(alias))
	}
	return existing
}

func shiftRangeStart(r token.Range, columnDelta, offsetDelta int) token.Range {
	r.Start.Column += columnDelta
	r.Start.Offset += offsetDelta
	return r
}

func sliceInlineRange(source string, base token.Range, startByte, endByte int) token.Range {
	startColumnDelta := utf16Len(source[:startByte])
	endColumnDelta := utf16Len(source[:endByte])

	return token.Range{
		Start: token.Position{
			Line:   base.Start.Line,
			Column: base.Start.Column + startColumnDelta,
			Offset: base.Start.Offset + startByte,
		},
		End: token.Position{
			Line:   base.End.Line,
			Column: base.Start.Column + endColumnDelta,
			Offset: base.Start.Offset + endByte,
		},
	}
}

// appendText appends text to the last TextNode if possible, or creates a new one.
func appendText(nodes []ast.Inline, text string, r token.Range) []ast.Inline {
	if len(nodes) > 0 {
		if tn, ok := nodes[len(nodes)-1].(*ast.TextNode); ok {
			tn.Value += text
			tn.Range.End = r.End
			return nodes
		}
	}
	return append(nodes, &ast.TextNode{Value: text, Range: r})
}

func utf16Len(s string) int {
	return token.UTF16Len(s)
}
