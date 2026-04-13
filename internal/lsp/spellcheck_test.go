package lsp

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"go.lsp.dev/protocol"
)

func TestComputeSpellcheckContext_AllowsCharacterWordsAndIgnoresStructuralTokens(t *testing.T) {
	src := `# Play
Subtitle: A Test

## Dramatis Personae

KING LEAR/LEAR - Ruler of Britain
O'BRIEN

## ACT I

### SCENE 1

KING LEAR
Ths line has a typo.

SONG 1: Ballad

SONG END
`

	doc, errs := parser.Parse([]byte(src))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	ctx := ComputeSpellcheckContext(doc, errs)

	assertContains(t, ctx.AllowWords, "KING")
	assertContains(t, ctx.AllowWords, "LEAR")
	assertContains(t, ctx.AllowWords, "O'BRIEN")
	assertContainsRange(t, ctx.IgnoredRanges, 1, 0, 1, 8)    // Subtitle
	assertContainsRange(t, ctx.IgnoredRanges, 12, 0, 12, 9)  // KING LEAR cue
	assertContainsRange(t, ctx.IgnoredRanges, 15, 0, 15, 14) // SONG 1: Ballad
	assertContainsRange(t, ctx.IgnoredRanges, 17, 0, 17, 8)  // SONG END
}

func TestSpellAllowWords_SplitsMultiWordNamesAndDedupes(t *testing.T) {
	words := spellAllowWords([]string{"KING LEAR", "Lear", "O'BRIEN"})

	assertContains(t, words, "KING")
	assertContains(t, words, "LEAR")
	assertContains(t, words, "O'BRIEN")
	if countOccurrences(words, "LEAR") != 1 {
		t.Fatalf("expected LEAR once, got %v", words)
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("expected %q in %v", want, values)
}

func countOccurrences(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func assertContainsRange(t *testing.T, ranges []protocol.Range, startLine, startChar, endLine, endChar uint32) {
	t.Helper()
	for _, r := range ranges {
		if r.Start.Line == startLine &&
			r.Start.Character == startChar &&
			r.End.Line == endLine &&
			r.End.Character == endChar {
			return
		}
	}
	t.Fatalf("expected range [%d:%d]-[%d:%d] in %v", startLine, startChar, endLine, endChar, ranges)
}
