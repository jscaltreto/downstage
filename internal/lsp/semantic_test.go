package lsp

import (
	"reflect"
	"testing"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/token"
)

func TestDeltaEncode_Empty(t *testing.T) {
	result := deltaEncode(nil)
	if result != nil {
		t.Errorf("expected nil for empty tokens, got %v", result)
	}
}

func TestDeltaEncode_SingleToken(t *testing.T) {
	tokens := []rawToken{
		{line: 3, startChar: 5, length: 10, tokenType: tokenTypeNamespace},
	}
	result := deltaEncode(tokens)
	expected := []uint32{3, 5, 10, 0, 0}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDeltaEncode_MultipleTokens(t *testing.T) {
	tokens := []rawToken{
		{line: 0, startChar: 0, length: 5, tokenType: tokenTypeNamespace},
		{line: 0, startChar: 10, length: 3, tokenType: tokenTypeType},
		{line: 2, startChar: 4, length: 7, tokenType: tokenTypeComment},
	}
	result := deltaEncode(tokens)
	expected := []uint32{
		0, 0, 5, 0, 0, // first token
		0, 10, 3, 1, 0, // same line, delta char = 10
		2, 4, 7, 2, 0, // new line, absolute char
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestSortTokens(t *testing.T) {
	tokens := []rawToken{
		{line: 2, startChar: 0},
		{line: 0, startChar: 5},
		{line: 0, startChar: 0},
		{line: 1, startChar: 3},
	}
	sortTokens(tokens)

	expected := []rawToken{
		{line: 0, startChar: 0},
		{line: 0, startChar: 5},
		{line: 1, startChar: 3},
		{line: 2, startChar: 0},
	}
	if !reflect.DeepEqual(tokens, expected) {
		t.Errorf("sort failed: got %v", tokens)
	}
}

func TestComputeSemanticTokens_Nil(t *testing.T) {
	result := computeSemanticTokens(nil, nil)
	if result != nil {
		t.Errorf("expected nil for nil doc, got %v", result)
	}
}

func TestComputeSemanticTokens_WithDialogue(t *testing.T) {
	dialogue := &ast.Dialogue{
		Character: "HAMLET",
		Range: token.Range{
			Start: token.Position{Line: 5, Column: 0},
			End:   token.Position{Line: 7, Column: 0},
		},
	}
	dialogue.SetNameRange(token.Range{
		Start: token.Position{Line: 5, Column: 0},
		End:   token.Position{Line: 5, Column: 6},
	})
	doc := &ast.Document{Body: []ast.Node{dialogue}}

	tokens := computeSemanticTokens(doc, nil)
	// Should produce one token for the character name.
	if len(tokens) != 5 {
		t.Fatalf("expected 5 values (1 token), got %d", len(tokens))
	}
	// deltaLine=5, deltaStartChar=0, length=6 ("HAMLET"), tokenType=1 (type), mods=0
	expected := []uint32{5, 0, 6, tokenTypeType, 0}
	if !reflect.DeepEqual(tokens, expected) {
		t.Errorf("expected %v, got %v", expected, tokens)
	}
}

func TestComputeSemanticTokens_UsesUTF16Columns(t *testing.T) {
	dialogue := &ast.Dialogue{
		Character: "A🙂",
		Range: token.Range{
			Start: token.Position{Line: 3, Column: 0},
			End:   token.Position{Line: 3, Column: 0},
		},
	}
	dialogue.SetNameRange(token.Range{
		Start: token.Position{Line: 3, Column: 0},
		End:   token.Position{Line: 3, Column: 3},
	})

	doc := &ast.Document{Body: []ast.Node{dialogue}}
	tokens := computeSemanticTokens(doc, nil)
	expected := []uint32{3, 0, 3, tokenTypeType, 0}
	if !reflect.DeepEqual(tokens, expected) {
		t.Errorf("expected %v, got %v", expected, tokens)
	}
}

func TestExtractTokens_DualDialogueExcludesCaret(t *testing.T) {
	doc, errs := parser.Parse([]byte(`HAMLET
Hello.

OPHELIA ^
Hi.`))
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	dual, ok := doc.Body[0].(*ast.DualDialogue)
	if !ok {
		t.Fatalf("expected dual dialogue, got %T", doc.Body[0])
	}

	tokens := extractTokens(dual.Right)
	if len(tokens) != 1 {
		t.Fatalf("expected one semantic token, got %d", len(tokens))
	}
	if tokens[0].length != len("OPHELIA") {
		t.Fatalf("expected token length %d, got %d", len("OPHELIA"), tokens[0].length)
	}
}
