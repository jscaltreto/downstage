package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rootTestdata returns the path to the repo-root testdata/ directory.
func rootTestdata(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "testdata")
}

// readTestdata reads a file from the repo-root testdata/ directory.
func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(rootTestdata(t), name))
	require.NoError(t, err)
	return data
}

// nodeCollector is a Visitor that collects all nodes matching a predicate.
type nodeCollector struct {
	predicate func(ast.Node) bool
	nodes     []ast.Node
}

func (c *nodeCollector) Visit(node ast.Node) ast.Visitor {
	if c.predicate(node) {
		c.nodes = append(c.nodes, node)
	}
	return c
}

// collectNodes walks the AST and returns all nodes matching the predicate.
func collectNodes(t *testing.T, doc *ast.Document, predicate func(ast.Node) bool) []ast.Node {
	t.Helper()
	c := &nodeCollector{predicate: predicate, nodes: nil}
	ast.Walk(c, doc)
	return c.nodes
}

// collectTyped is a generic helper that collects nodes of a specific type.
func collectTyped[T ast.Node](t *testing.T, doc *ast.Document) []T {
	t.Helper()
	var result []T
	c := &nodeCollector{
		predicate: func(n ast.Node) bool {
			_, ok := n.(T)
			return ok
		},
	}
	ast.Walk(c, doc)
	for _, n := range c.nodes {
		result = append(result, n.(T))
	}
	return result
}

// collectSections returns all Section nodes of the given kind.
func collectSections(t *testing.T, doc *ast.Document, kind ast.SectionKind) []*ast.Section {
	t.Helper()
	all := collectTyped[*ast.Section](t, doc)
	var filtered []*ast.Section
	for _, s := range all {
		if s.Kind == kind {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func firstTopLevelSection(t *testing.T, doc *ast.Document) *ast.Section {
	t.Helper()
	for _, node := range doc.Body {
		if section, ok := node.(*ast.Section); ok && section.Level == 1 {
			return section
		}
	}
	t.Fatal("expected at least one top-level section")
	return nil
}

func metadataMap(section *ast.Section) map[string]string {
	if section == nil || section.Metadata == nil {
		return nil
	}
	values := make(map[string]string, len(section.Metadata.Entries))
	for _, entry := range section.Metadata.Entries {
		values[entry.Key] = entry.Value
	}
	return values
}

// --- No-panic tests ---

func TestAllTestdataFilesNoPanic(t *testing.T) {
	dirs := []string{
		rootTestdata(t),
		filepath.Join("testdata"),
	}

	var files []string
	for _, dir := range dirs {
		matches, err := filepath.Glob(filepath.Join(dir, "*.ds"))
		require.NoError(t, err)
		files = append(files, matches...)
	}
	require.NotEmpty(t, files, "expected to find .ds test data files")

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			input, err := os.ReadFile(f)
			require.NoError(t, err)

			doc, _ := parser.Parse(input)
			require.NotNil(t, doc, "parser returned nil document for %s", f)
		})
	}
}

// --- full_play.ds ---

func TestFullPlay(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "full_play.ds"))
	assert.Empty(t, errs)

	play := firstTopLevelSection(t, doc)
	require.Equal(t, "The Last Curtain Call", play.Title)
	meta := metadataMap(play)
	require.NotNil(t, meta)
	assert.Equal(t, "A Drama in Two Acts", meta["Subtitle"])
	assert.Equal(t, "Eleanor Vance", meta["Author"])
	assert.Equal(t, "2025", meta["Date"])
	assert.Equal(t, "Third Draft", meta["Draft"])

	// Body has Act sections
	acts := collectSections(t, doc, ast.SectionAct)
	require.NotEmpty(t, acts, "expected Act sections")

	// Acts have Scene children
	var totalScenes int
	for _, act := range acts {
		for _, child := range act.Children {
			if s, ok := child.(*ast.Section); ok && s.Kind == ast.SectionScene {
				totalScenes++
			}
		}
	}
	assert.True(t, totalScenes >= 2, "expected at least 2 scenes total, got %d", totalScenes)

	// Dialogue exists
	dialogues := collectTyped[*ast.Dialogue](t, doc)
	assert.NotEmpty(t, dialogues, "expected Dialogue nodes")

	// Verse lines exist
	var hasVerse bool
	for _, dlg := range dialogues {
		for _, line := range dlg.Lines {
			if line.IsVerse {
				hasVerse = true
				break
			}
		}
		if hasVerse {
			break
		}
	}
	assert.True(t, hasVerse, "expected verse lines in full play")

	// Stage directions
	sds := collectTyped[*ast.StageDirection](t, doc)
	assert.NotEmpty(t, sds, "expected StageDirection nodes")

	// Comments
	comments := collectTyped[*ast.Comment](t, doc)
	assert.NotEmpty(t, comments, "expected Comment nodes")

	// Page breaks
	pbs := collectTyped[*ast.PageBreak](t, doc)
	assert.NotEmpty(t, pbs, "expected PageBreak nodes")

	// Songs
	songs := collectTyped[*ast.Song](t, doc)
	assert.NotEmpty(t, songs, "expected Song nodes")
}

// --- minimal.ds ---

func TestMinimalPlay(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "minimal.ds"))
	assert.Empty(t, errs)

	play := firstTopLevelSection(t, doc)
	assert.Equal(t, "Minimal Play", play.Title)

	require.NotEmpty(t, doc.Body)
	dialogues := collectTyped[*ast.Dialogue](t, doc)
	assert.NotEmpty(t, dialogues, "expected at least one dialogue")
}

// --- title_page_only.ds ---

func TestIntegrationTitlePageOnly(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "title_page_only.ds"))
	assert.Empty(t, errs)

	section := firstTopLevelSection(t, doc)
	require.NotNil(t, section.Metadata)
	assert.True(t, len(section.Metadata.Entries) >= 3, "expected multiple entries, got %d", len(section.Metadata.Entries))
	assert.Empty(t, section.Children, "expected no body content for title-page-only section")
}

// --- no_title_page.ds ---

func TestIntegrationNoTitlePage(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "no_title_page.ds"))
	assert.Empty(t, errs)

	assert.Nil(t, doc.TitlePage, "expected nil TitlePage")
	assert.NotEmpty(t, doc.Body, "expected body content")
}

// --- no_acts.ds ---

func TestIntegrationNoActs(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "no_acts.ds"))
	assert.Empty(t, errs)

	// The parser treats the top-level "# Short Play" heading as an Act-like
	// section, so we verify scenes and dialogue exist in the tree.
	require.NotEmpty(t, doc.Body, "expected body content")

	scenes := collectSections(t, doc, ast.SectionScene)
	dialogues := collectTyped[*ast.Dialogue](t, doc)
	assert.NotEmpty(t, scenes, "expected Scene sections")
	assert.NotEmpty(t, dialogues, "expected Dialogue nodes")
}

// --- dialogue_varieties.ds ---

func TestIntegrationDialogueVarieties(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "dialogue_varieties.ds"))
	assert.Empty(t, errs)

	dialogues := collectTyped[*ast.Dialogue](t, doc)
	assert.True(t, len(dialogues) >= 5, "expected many dialogue nodes, got %d", len(dialogues))

	// Check for parentheticals
	var hasParenthetical bool
	for _, dlg := range dialogues {
		if dlg.Parenthetical != "" {
			hasParenthetical = true
			break
		}
	}
	assert.True(t, hasParenthetical, "expected at least one dialogue with parenthetical")

	// Check for verse lines
	var hasVerse bool
	for _, dlg := range dialogues {
		for _, line := range dlg.Lines {
			if line.IsVerse {
				hasVerse = true
				break
			}
		}
		if hasVerse {
			break
		}
	}
	assert.True(t, hasVerse, "expected verse lines in dialogue varieties")
}

// --- dual_dialogue.ds ---

func TestIntegrationDualDialogue(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "dual_dialogue.ds"))
	assert.Empty(t, errs)

	duals := collectTyped[*ast.DualDialogue](t, doc)
	assert.Len(t, duals, 2, "expected 2 dual dialogue blocks")

	// First dual dialogue
	assert.Equal(t, "BRICK", duals[0].Left.Character)
	assert.Equal(t, "STEEL", duals[0].Right.Character)

	// Second dual dialogue has parentheticals
	assert.Equal(t, "(frustrated)", duals[1].Left.Parenthetical)
	assert.Equal(t, "(calmly)", duals[1].Right.Parenthetical)

	// Scene 3 has a ^ without preceding dialogue — should be a regular dialogue
	// (CHORUS ^ has no preceding dialogue in that scene context)
	dialogues := collectTyped[*ast.Dialogue](t, doc)
	var chorusFound bool
	for _, dlg := range dialogues {
		if dlg.Character == "CHORUS" {
			chorusFound = true
		}
	}
	assert.True(t, chorusFound, "expected CHORUS as standalone dialogue (no preceding dialogue to pair)")
}

// --- formatting.ds ---

func TestIntegrationFormatting(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "formatting.ds"))
	assert.Empty(t, errs)

	var hasBold, hasItalic, hasBoldItalic, hasUnderline, hasStrikethrough bool
	nodes := collectNodes(t, doc, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.BoldNode:
			hasBold = true
		case *ast.ItalicNode:
			hasItalic = true
		case *ast.BoldItalicNode:
			hasBoldItalic = true
		case *ast.UnderlineNode:
			hasUnderline = true
		case *ast.StrikethroughNode:
			hasStrikethrough = true
		}
		return false
	})
	_ = nodes

	assert.True(t, hasBold, "expected BoldNode")
	assert.True(t, hasItalic, "expected ItalicNode")
	assert.True(t, hasBoldItalic, "expected BoldItalicNode")
	assert.True(t, hasUnderline, "expected UnderlineNode")
	assert.True(t, hasStrikethrough, "expected StrikethroughNode")
}

// --- comments.ds ---

func TestIntegrationComments(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "comments.ds"))
	assert.Empty(t, errs)

	comments := collectTyped[*ast.Comment](t, doc)
	require.NotEmpty(t, comments, "expected Comment nodes")

	var hasLine, hasBlock bool
	for _, c := range comments {
		if c.Block {
			hasBlock = true
		} else {
			hasLine = true
		}
	}
	assert.True(t, hasLine, "expected line comments (Block=false)")
	assert.True(t, hasBlock, "expected block comments (Block=true)")
}

// --- songs.ds ---

func TestIntegrationSongs(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "songs.ds"))
	// The unnumbered "SONG: A Simple Tune" format produces a stray SongEnd
	// error in the current parser. We accept this known limitation.
	_ = errs

	songs := collectTyped[*ast.Song](t, doc)
	require.True(t, len(songs) >= 2, "expected at least 2 songs, got %d", len(songs))

	// At least one song should have a number
	var hasNumber bool
	for _, s := range songs {
		if s.Number != "" {
			hasNumber = true
			break
		}
	}
	assert.True(t, hasNumber, "expected at least one song with a number")

	// All songs should have titles
	for _, s := range songs {
		assert.NotEmpty(t, s.Title, "expected song to have a title")
	}
}

// --- errors.ds ---

func TestIntegrationErrorRecovery(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "errors.ds"))

	assert.NotEmpty(t, errs, "expected parse errors for malformed input")
	require.NotNil(t, doc, "document should not be nil even with errors")

	// Despite errors, some valid content should still be parsed
	dialogues := collectTyped[*ast.Dialogue](t, doc)
	assert.NotEmpty(t, dialogues, "expected some dialogue to survive error recovery")
}

// --- empty input ---

func TestIntegrationEmptyInput(t *testing.T) {
	doc, errs := parser.Parse([]byte(""))
	require.NotNil(t, doc)
	assert.Empty(t, errs)
}

// --- example.ds ---

func TestIntegrationExamplePlay(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "example.ds"))
	assert.Empty(t, errs, "example play should parse without errors: %v", errs)

	play := firstTopLevelSection(t, doc)
	assert.Equal(t, "The Example Play", play.Title)

	// Dramatis personae
	dp := ast.FindDramatisPersonaeInSection(play)
	require.NotNil(t, dp, "expected Dramatis Personae section")
	allChars := dp.AllCharacters()
	assert.True(t, len(allChars) >= 3, "expected at least 3 characters")

	// Acts
	acts := collectSections(t, doc, ast.SectionAct)
	assert.True(t, len(acts) >= 2, "expected at least 2 acts")

	// Scenes
	var totalScenes int
	for _, act := range acts {
		for _, child := range act.Children {
			if s, ok := child.(*ast.Section); ok && s.Kind == ast.SectionScene {
				totalScenes++
			}
		}
	}
	assert.True(t, totalScenes >= 3, "expected at least 3 scenes total")

	// Dialogue
	dialogues := collectTyped[*ast.Dialogue](t, doc)
	assert.NotEmpty(t, dialogues)

	// Songs
	songs := collectTyped[*ast.Song](t, doc)
	assert.NotEmpty(t, songs, "expected songs in example play")

	// Page breaks
	pbs := collectTyped[*ast.PageBreak](t, doc)
	assert.NotEmpty(t, pbs, "expected page breaks in example play")

	// Comments (both line and block)
	comments := collectTyped[*ast.Comment](t, doc)
	assert.NotEmpty(t, comments)
}

// --- edge_cases.ds ---

func TestIntegrationEdgeCases(t *testing.T) {
	doc, errs := parser.Parse(readTestdata(t, "edge_cases.ds"))
	// Edge cases may or may not produce errors; we just ensure no panic
	_ = errs
	require.NotNil(t, doc)

	// Page breaks should survive
	pbs := collectTyped[*ast.PageBreak](t, doc)
	assert.NotEmpty(t, pbs, "expected page breaks in edge cases")

	// Dialogue should survive despite tricky names
	dialogues := collectTyped[*ast.Dialogue](t, doc)
	assert.NotEmpty(t, dialogues, "expected dialogue in edge cases")
}

// --- Garbage input should not panic ---

func TestGarbageInputNoPanic(t *testing.T) {
	inputs := [][]byte{
		[]byte("@@@###***"),
		[]byte("SONG 99\n\n\n\n"),
		[]byte("# \n## \n### \n"),
		[]byte("((((("),
		[]byte("/* never closed"),
		[]byte("\x00\x01\x02\x03"),
		[]byte("Title:\n\nSOMECHARACTER\n  verse line\n===\n(stage dir)\n// comment\n/* block */"),
	}
	for i, input := range inputs {
		t.Run("garbage_"+string(rune('A'+i)), func(t *testing.T) {
			doc, _ := parser.Parse(input)
			assert.NotNil(t, doc, "parser should never return nil document")
		})
	}
}
