package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

// CanonicalNameMap builds a name-canonicalization function from a document's
// dramatis personae. Aliases collapse to their primary name; unknown names are
// returned uppercased verbatim. The function is safe to call concurrently.
func CanonicalNameMap(doc *ast.Document) func(string) string {
	if doc == nil {
		return strings.ToUpper
	}
	m := map[string]string{}
	for _, node := range doc.Body {
		collectCharactersInto(node, m)
	}
	return func(name string) string {
		key := strings.ToUpper(strings.TrimSpace(name))
		if v, ok := m[key]; ok {
			return v
		}
		return key
	}
}

func collectCharactersInto(node ast.Node, m map[string]string) {
	switch n := node.(type) {
	case *ast.Section:
		if n.Kind == ast.SectionDramatisPersonae {
			for _, c := range n.AllCharacters() {
				canonical := strings.ToUpper(strings.TrimSpace(c.Name))
				m[canonical] = canonical
				for _, alias := range c.Aliases {
					m[strings.ToUpper(strings.TrimSpace(alias))] = canonical
				}
			}
			return
		}
		for _, child := range n.Children {
			collectCharactersInto(child, m)
		}
	}
}

func fingerprintSectionHeader(s *ast.Section) string {
	h := sha256.New()
	fmt.Fprintf(h, "section|%d|%d|%s|%s\n", int(s.Kind), s.Level, canonText(s.Title), strings.TrimSpace(s.Number))
	if s.Metadata != nil {
		for _, kv := range s.Metadata.Entries {
			fmt.Fprintf(h, "meta|%s=%s\n", canonText(kv.Key), canonText(kv.Value))
		}
	}
	return hexSum(h)
}

func fingerprintSectionLine(line *ast.SectionLine) string {
	h := sha256.New()
	fmt.Fprintf(h, "sectionline|%s", canonText(render.PlainText(line.Content)))
	return hexSum(h)
}

func fingerprintDialogue(d *ast.Dialogue, canonName func(string) string) string {
	h := sha256.New()
	name := d.Character
	if canonName != nil {
		name = canonName(d.Character)
	}
	fmt.Fprintf(h, "dialogue|%s|%s\n", name, canonText(d.Parenthetical))
	for _, line := range d.Lines {
		text := canonText(render.PlainText(line.Content))
		fmt.Fprintf(h, "line|%t|%s\n", line.IsVerse, text)
	}
	return hexSum(h)
}

func fingerprintDualDialogue(d *ast.DualDialogue, canonName func(string) string) string {
	h := sha256.New()
	fmt.Fprintf(h, "dual|%s|%s",
		fingerprintDialogue(d.Left, canonName),
		fingerprintDialogue(d.Right, canonName),
	)
	return hexSum(h)
}

func fingerprintStageDirection(sd *ast.StageDirection) string {
	h := sha256.New()
	fmt.Fprintf(h, "sd|%s", canonText(render.PlainText(sd.Content)))
	return hexSum(h)
}

func fingerprintCallout(c *ast.Callout) string {
	h := sha256.New()
	fmt.Fprintf(h, "callout|%s", canonText(render.PlainText(c.Content)))
	return hexSum(h)
}

func fingerprintSong(s *ast.Song, canonName func(string) string) string {
	h := sha256.New()
	fmt.Fprintf(h, "song|%s|%s\n", canonText(s.Number), canonText(s.Title))
	for _, child := range s.Content {
		fmt.Fprintf(h, "child|%s\n", childFingerprint(child, canonName))
	}
	return hexSum(h)
}

func fingerprintVerseBlock(vb *ast.VerseBlock) string {
	h := sha256.New()
	h.Write([]byte("verseblock"))
	for _, line := range vb.Lines {
		fmt.Fprintf(h, "|%s", canonText(render.PlainText(line.Content)))
	}
	return hexSum(h)
}

// childFingerprint computes a stable fingerprint for any block node, used
// when folding a parent's children into the parent's fingerprint (Song).
func childFingerprint(node ast.Node, canonName func(string) string) string {
	switch n := node.(type) {
	case *ast.Dialogue:
		return fingerprintDialogue(n, canonName)
	case *ast.DualDialogue:
		return fingerprintDualDialogue(n, canonName)
	case *ast.StageDirection:
		return fingerprintStageDirection(n)
	case *ast.Callout:
		return fingerprintCallout(n)
	case *ast.VerseBlock:
		return fingerprintVerseBlock(n)
	case *ast.PageBreak:
		return "pagebreak"
	case *ast.Comment:
		return ""
	}
	return ""
}

// canonText normalizes whitespace for fingerprinting: trim, collapse internal
// runs of whitespace to single spaces. Case is preserved.
func canonText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return b.String()
}

func hexSum(h interface{ Sum([]byte) []byte }) string {
	return hex.EncodeToString(h.Sum(nil))
}
