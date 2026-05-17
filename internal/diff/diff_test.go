package diff

import (
	"testing"

	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseOrFail is a small helper to keep test fixtures readable.
func parseOrFail(t *testing.T, src string) []Block {
	t.Helper()
	doc, errs := parser.Parse([]byte(src))
	require.Empty(t, errs, "parse errors: %v", errs)
	return FlattenedBlocks(doc, CanonicalNameMap(doc))
}

func TestFlattenedBlocks_OrderedKindsMatchRendererWalk(t *testing.T) {
	src := "# Test\n\n" +
		"## ACT I\n\n" +
		"### SCENE 1\n\n" +
		"> Lights up.\n\n" +
		"HAMLET\nTo be or not to be.\n\n" +
		">> Important note.\n\n" +
		"===\n\n" +
		"### SCENE 2\n\n" +
		"HAMLET\nThat is the question.\n"

	blocks := parseOrFail(t, src)
	require.NotEmpty(t, blocks)

	// We expect at least: title-section header, act header, scene-1 header,
	// stage direction, dialogue, callout, page break, scene-2 header,
	// dialogue. Allow flex on exact count to tolerate parser nuances.
	var kinds []BlockKind
	for _, b := range blocks {
		kinds = append(kinds, b.Kind)
	}

	// Section headers always come first within their nesting.
	assert.Equal(t, BlockSectionHeader, kinds[0], "first block should be the play title section header")

	// Find expected kinds appear in order.
	mustContain := []BlockKind{
		BlockSectionHeader, // play title
		BlockSectionHeader, // act
		BlockSectionHeader, // scene 1
		BlockStageDirection,
		BlockDialogue,
		BlockCallout,
		BlockPageBreak,
		BlockSectionHeader, // scene 2
		BlockDialogue,
	}
	pos := 0
	for _, want := range mustContain {
		found := false
		for ; pos < len(kinds); pos++ {
			if kinds[pos] == want {
				found = true
				pos++
				break
			}
		}
		assert.True(t, found, "expected to find %v at or after current position", want)
	}
}

func TestDiff_EmptyEqual(t *testing.T) {
	hunks := Diff(nil, nil)
	assert.Empty(t, hunks)
}

func TestDiff_AllEqual(t *testing.T) {
	src := "# Same\n\nHAMLET\nLine.\n"
	v1 := parseOrFail(t, src)
	v2 := parseOrFail(t, src)
	hunks := Diff(v1, v2)
	require.Len(t, hunks, 1)
	assert.Equal(t, HunkEqual, hunks[0].Kind)
	assert.Equal(t, len(v1), hunks[0].V1End)
	assert.Equal(t, len(v2), hunks[0].V2End)
}

func TestDiff_PureInsert(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nLine A.\n")
	v2 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nLine A.\n\nHAMLET\nLine B.\n")
	hunks := Diff(v1, v2)

	var insert *Hunk
	for i := range hunks {
		if hunks[i].Kind == HunkInsert {
			insert = &hunks[i]
		}
	}
	require.NotNil(t, insert, "expected an Insert hunk: %+v", hunks)
	assert.Equal(t, insert.V1Start, insert.V1End, "insert v1 span must be empty")
	assert.Greater(t, insert.V2End, insert.V2Start, "insert v2 span must be non-empty")
}

func TestDiff_PureDelete(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nLine A.\n\nHAMLET\nLine B.\n")
	v2 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nLine A.\n")
	hunks := Diff(v1, v2)

	var del *Hunk
	for i := range hunks {
		if hunks[i].Kind == HunkDelete {
			del = &hunks[i]
		}
	}
	require.NotNil(t, del, "expected a Delete hunk: %+v", hunks)
	assert.Greater(t, del.V1End, del.V1Start)
	assert.Equal(t, del.V2Start, del.V2End)
}

func TestDiff_ModifyOneDialogue(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nOriginal line.\n")
	v2 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nUpdated line.\n")
	hunks := Diff(v1, v2)

	var modify *Hunk
	for i := range hunks {
		if hunks[i].Kind == HunkModify {
			modify = &hunks[i]
		}
	}
	require.NotNil(t, modify, "expected a Modify hunk: %+v", hunks)
	assert.Equal(t, 1, modify.V1End-modify.V1Start)
	assert.Equal(t, 1, modify.V2End-modify.V2Start)
}

func TestDiff_CharacterAliasIsNotAChange(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## Dramatis Personae\n\nHAMLET/HAM - Prince of Denmark\n\n## ACT I\n\nHAMLET\nTo be or not to be.\n")
	v2 := parseOrFail(t, "# Play\n\n## Dramatis Personae\n\nHAMLET/HAM - Prince of Denmark\n\n## ACT I\n\nHAM\nTo be or not to be.\n")

	hunks := Diff(v1, v2)

	// Verify no Modify or Insert/Delete hunks; only Equal.
	for _, h := range hunks {
		assert.Equal(t, HunkEqual, h.Kind, "alias change should not produce non-Equal hunks: %+v", hunks)
	}
}

func TestDiff_CommentOnlyEditIsNoOp(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nLine.\n")
	v2 := parseOrFail(t, "# Play\n\n// note\n\n## ACT I\n\n// another\n\nHAMLET\nLine.\n")
	hunks := Diff(v1, v2)
	for _, h := range hunks {
		assert.Equal(t, HunkEqual, h.Kind, "comment-only edit should not produce non-Equal hunks: %+v", hunks)
	}
}

func TestDiff_WhitespaceOnlyEditIsNoOp(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nLine of dialogue.\n")
	v2 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nLine of    dialogue.\n")
	hunks := Diff(v1, v2)
	for _, h := range hunks {
		assert.Equal(t, HunkEqual, h.Kind, "whitespace-only edit should not produce non-Equal hunks: %+v", hunks)
	}
}

func TestDiff_Reorder_DetectedAsModification(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nFirst.\n\nHAMLET\nSecond.\n")
	v2 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nSecond.\n\nHAMLET\nFirst.\n")
	hunks := Diff(v1, v2)

	hasNonEqual := false
	for _, h := range hunks {
		if h.Kind != HunkEqual {
			hasNonEqual = true
		}
	}
	assert.True(t, hasNonEqual, "reorder should surface as a non-Equal hunk: %+v", hunks)
}

func TestDiff_MyersFallback_NoUniqueFingerprints(t *testing.T) {
	// Build streams where every fingerprint repeats — Patience finds no
	// anchors and the algorithm must fall through to Myers.
	v1 := parseOrFail(t, "# Play\n\nHAMLET\nA.\n\nHAMLET\nA.\n\nHAMLET\nA.\n")
	v2 := parseOrFail(t, "# Play\n\nHAMLET\nA.\n\nHAMLET\nA.\n\nHAMLET\nA.\n\nHAMLET\nA.\n")
	hunks := Diff(v1, v2)

	// Expect the diff still terminates and produces an Insert covering the
	// extra dialogue in v2.
	totalV2 := 0
	for _, h := range hunks {
		totalV2 += h.V2End - h.V2Start
	}
	assert.Equal(t, len(v2), totalV2, "diff must cover every v2 block: %+v", hunks)
}

func TestDiff_FullRewrite_AllChanged(t *testing.T) {
	v1 := parseOrFail(t, "# Play\n\n## ACT I\n\nHAMLET\nOld line.\n")
	v2 := parseOrFail(t, "# Play 2\n\n## ACT I\n\nOPHELIA\nBrand new line.\n")
	hunks := Diff(v1, v2)

	totalV1, totalV2 := 0, 0
	for _, h := range hunks {
		totalV1 += h.V1End - h.V1Start
		totalV2 += h.V2End - h.V2Start
	}
	assert.Equal(t, len(v1), totalV1)
	assert.Equal(t, len(v2), totalV2)
}

func TestCanonicalNameMap_KnownAndUnknown(t *testing.T) {
	doc, errs := parser.Parse([]byte("# P\n\n## Dramatis Personae\n\nHAMLET/HAM - Prince\n"))
	require.Empty(t, errs)
	canon := CanonicalNameMap(doc)
	assert.Equal(t, "HAMLET", canon("hamlet"))
	assert.Equal(t, "HAMLET", canon("HAM"))
	assert.Equal(t, "OPHELIA", canon("ophelia"), "unknown name should round-trip uppercased")
}

func TestCanonText_NormalizesWhitespace(t *testing.T) {
	assert.Equal(t, "to be or not to be", canonText("   to be   or  not\tto\nbe   "))
	assert.Equal(t, "", canonText("   \t\n"))
}
