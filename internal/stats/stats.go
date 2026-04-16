// Package stats computes manuscript statistics from a parsed Downstage
// document. The output is shared core data intended for reuse by the CLI,
// LSP, and editor integrations. Counts are derived from the AST and are
// deterministic for a given input.
package stats

import (
	"sort"
	"strings"
	"unicode"

	"github.com/jscaltreto/downstage/internal/ast"
)

// Stats summarizes a manuscript.
type Stats struct {
	Acts                int              `json:"acts"`
	Scenes              int              `json:"scenes"`
	Songs               int              `json:"songs"`
	TotalWords          int              `json:"totalWords"`
	DialogueWords       int              `json:"dialogueWords"`
	DialogueLines       int              `json:"dialogueLines"`
	Speeches            int              `json:"speeches"`
	StageDirections     int              `json:"stageDirections"`
	StageDirectionWords int              `json:"stageDirectionWords"`
	Characters          []CharacterStats `json:"characters"`
	Runtime             RuntimeEstimate  `json:"runtime"`
}

// CharacterStats holds per-character tallies. Aliases listed in the
// dramatis personae are folded into the primary name.
type CharacterStats struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	Speeches      int      `json:"speeches"`
	DialogueLines int      `json:"dialogueLines"`
	DialogueWords int      `json:"dialogueWords"`
}

// Compute walks the document and returns manuscript statistics. The
// runtime field is filled in using the supplied RuntimeOptions; see
// EstimateRuntime for the heuristic.
func Compute(doc *ast.Document, rt RuntimeOptions) Stats {
	s := Stats{}
	if doc == nil {
		s.Runtime = EstimateRuntime(0, rt)
		return s
	}

	resolver := newAliasResolver(doc)
	perChar := make(map[string]*CharacterStats)

	addCharacter := func(name string) *CharacterStats {
		if existing, ok := perChar[name]; ok {
			return existing
		}
		cs := &CharacterStats{Name: name, Aliases: resolver.aliasesFor(name)}
		perChar[name] = cs
		return cs
	}

	var walk func(nodes []ast.Node)
	walk = func(nodes []ast.Node) {
		for _, node := range nodes {
			switch n := node.(type) {
			case *ast.Section:
				switch n.Kind {
				case ast.SectionAct:
					s.Acts++
				case ast.SectionScene:
					s.Scenes++
				}
				for _, item := range n.OrderedItems() {
					if item.Node != nil {
						walk([]ast.Node{item.Node})
						continue
					}
					if item.Line != nil {
						s.TotalWords += countWords(plainText(item.Line.Content))
					}
				}
			case *ast.Dialogue:
				s.Speeches++
				name := resolver.canonical(n.Character)
				cs := addCharacter(name)
				cs.Speeches++
				for _, line := range n.Lines {
					words := countSpokenWords(line.Content)
					if words == 0 && plainTextTrimmed(line.Content) == "" {
						continue
					}
					s.DialogueLines++
					s.DialogueWords += words
					s.TotalWords += words
					cs.DialogueLines++
					cs.DialogueWords += words
				}
			case *ast.DualDialogue:
				walk([]ast.Node{n.Left, n.Right})
			case *ast.StageDirection:
				s.StageDirections++
				w := countWords(plainText(n.Content))
				s.StageDirectionWords += w
				s.TotalWords += w
			case *ast.Callout:
				w := countWords(plainText(n.Content))
				s.TotalWords += w
			case *ast.Song:
				s.Songs++
				walk(n.Content)
			case *ast.VerseBlock:
				for _, line := range n.Lines {
					s.TotalWords += countWords(plainText(line.Content))
				}
			}
		}
	}

	walk(doc.Body)

	s.Characters = make([]CharacterStats, 0, len(perChar))
	for _, cs := range perChar {
		s.Characters = append(s.Characters, *cs)
	}
	sort.Slice(s.Characters, func(i, j int) bool {
		a, b := s.Characters[i], s.Characters[j]
		if a.Speeches != b.Speeches {
			return a.Speeches > b.Speeches
		}
		if a.DialogueWords != b.DialogueWords {
			return a.DialogueWords > b.DialogueWords
		}
		return a.Name < b.Name
	})

	s.Runtime = EstimateRuntime(s.DialogueWords, rt)
	return s
}

// aliasResolver maps raw cue names (including aliases) to a canonical
// character name using the dramatis personae. Unknown names are returned
// upper-cased so forced cues still tally consistently.
type aliasResolver struct {
	canonicalByKey map[string]string
	aliasesByName  map[string][]string
}

func newAliasResolver(doc *ast.Document) *aliasResolver {
	r := &aliasResolver{
		canonicalByKey: make(map[string]string),
		aliasesByName:  make(map[string][]string),
	}
	if doc == nil {
		return r
	}
	dp := ast.FindDramatisPersonae(doc.Body)
	if dp == nil {
		return r
	}
	for _, ch := range dp.AllCharacters() {
		name := strings.TrimSpace(ch.Name)
		if name == "" {
			continue
		}
		r.canonicalByKey[strings.ToUpper(name)] = name
		for _, alias := range ch.Aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			r.canonicalByKey[strings.ToUpper(alias)] = name
			r.aliasesByName[name] = append(r.aliasesByName[name], alias)
		}
	}
	return r
}

func (r *aliasResolver) canonical(raw string) string {
	key := strings.ToUpper(strings.TrimSpace(raw))
	if name, ok := r.canonicalByKey[key]; ok {
		return name
	}
	return key
}

func (r *aliasResolver) aliasesFor(name string) []string {
	aliases := r.aliasesByName[name]
	if len(aliases) == 0 {
		return nil
	}
	out := make([]string, len(aliases))
	copy(out, aliases)
	return out
}

// countSpokenWords returns the word count for a dialogue line, excluding
// inline stage directions which are performance notes rather than spoken
// text.
func countSpokenWords(inlines []ast.Inline) int {
	var b strings.Builder
	appendSpokenText(&b, inlines)
	return countWords(b.String())
}

func appendSpokenText(b *strings.Builder, inlines []ast.Inline) {
	for _, inline := range inlines {
		switch n := inline.(type) {
		case *ast.TextNode:
			b.WriteString(n.Value)
		case *ast.BoldNode:
			appendSpokenText(b, n.Content)
		case *ast.ItalicNode:
			appendSpokenText(b, n.Content)
		case *ast.BoldItalicNode:
			appendSpokenText(b, n.Content)
		case *ast.UnderlineNode:
			appendSpokenText(b, n.Content)
		case *ast.StrikethroughNode:
			appendSpokenText(b, n.Content)
		case *ast.InlineDirectionNode:
		}
	}
}

func plainText(inlines []ast.Inline) string {
	var b strings.Builder
	appendPlainText(&b, inlines)
	return b.String()
}

func plainTextTrimmed(inlines []ast.Inline) string {
	return strings.TrimSpace(plainText(inlines))
}

func appendPlainText(b *strings.Builder, inlines []ast.Inline) {
	for _, inline := range inlines {
		switch n := inline.(type) {
		case *ast.TextNode:
			b.WriteString(n.Value)
		case *ast.BoldNode:
			appendPlainText(b, n.Content)
		case *ast.ItalicNode:
			appendPlainText(b, n.Content)
		case *ast.BoldItalicNode:
			appendPlainText(b, n.Content)
		case *ast.UnderlineNode:
			appendPlainText(b, n.Content)
		case *ast.StrikethroughNode:
			appendPlainText(b, n.Content)
		case *ast.InlineDirectionNode:
			b.WriteString(" ")
			appendPlainText(b, n.Content)
			b.WriteString(" ")
		}
	}
}

// countWords splits on whitespace and counts runs that contain at least
// one letter or digit. Punctuation-only tokens (e.g. "—") do not count.
func countWords(s string) int {
	n := 0
	inWord := false
	hasAlnum := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if inWord && hasAlnum {
				n++
			}
			inWord = false
			hasAlnum = false
			continue
		}
		inWord = true
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasAlnum = true
		}
	}
	if inWord && hasAlnum {
		n++
	}
	return n
}
