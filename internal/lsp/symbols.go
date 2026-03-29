package lsp

import (
	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/parser"
	"github.com/jscaltreto/downstage/internal/token"
	"go.lsp.dev/protocol"
)

// computeDocumentSymbols builds a hierarchical document outline from the AST.
func computeDocumentSymbols(doc *ast.Document, _ []*parser.ParseError) []protocol.DocumentSymbol {
	if doc == nil {
		return nil
	}

	var symbols []protocol.DocumentSymbol

	for _, n := range doc.Body {
		switch v := n.(type) {
		case *ast.Section:
			symbols = append(symbols, sectionSymbol(v))
		case *ast.DualDialogue:
			symbols = append(symbols, dialogueSymbol(v.Left))
			symbols = append(symbols, dialogueSymbol(v.Right))
		case *ast.Dialogue:
			symbols = append(symbols, dialogueSymbol(v))
		case *ast.Song:
			symbols = append(symbols, songSymbol(v))
		}
	}

	return symbols
}

func sectionSymbol(s *ast.Section) protocol.DocumentSymbol {
	r := toLSPRange(s.Range)
	name := sectionSymbolName(s)
	kind := sectionSymbolKind(s.Kind)

	sym := protocol.DocumentSymbol{
		Name:           name,
		Kind:           kind,
		Range:          r,
		SelectionRange: r,
	}

	// Add nested sections and characters as children.
	seen := make(map[string]bool)
	for _, child := range s.Children {
		switch cv := child.(type) {
		case *ast.Section:
			sym.Children = append(sym.Children, sectionSymbol(cv))
		case *ast.Song:
			sym.Children = append(sym.Children, songSymbol(cv))
		case *ast.DualDialogue:
			for _, dialogue := range []*ast.Dialogue{cv.Left, cv.Right} {
				if dialogue == nil || seen[dialogue.Character] {
					continue
				}
				seen[dialogue.Character] = true
				sym.Children = append(sym.Children, characterSymbol(dialogue.Character, dialogue.NodeRange()))
			}
		default:
			if charName := characterNameFromNode(child); charName != "" && !seen[charName] {
				seen[charName] = true
				sym.Children = append(sym.Children, characterSymbol(charName, child.NodeRange()))
			}
		}
	}

	return sym
}

func sectionSymbolKind(kind ast.SectionKind) protocol.SymbolKind {
	switch kind {
	case ast.SectionAct:
		return protocol.SymbolKindNamespace
	case ast.SectionScene:
		return protocol.SymbolKindClass
	case ast.SectionDramatisPersonae:
		return protocol.SymbolKindStruct
	default:
		return protocol.SymbolKindFile
	}
}

func dialogueSymbol(d *ast.Dialogue) protocol.DocumentSymbol {
	r := toLSPRange(d.Range)
	return protocol.DocumentSymbol{
		Name:           d.Character,
		Kind:           protocol.SymbolKindFunction,
		Range:          r,
		SelectionRange: r,
	}
}

func songSymbol(s *ast.Song) protocol.DocumentSymbol {
	r := toLSPRange(s.Range)
	name := songSymbolName(s)
	return protocol.DocumentSymbol{
		Name:           name,
		Kind:           protocol.SymbolKindFunction,
		Range:          r,
		SelectionRange: r,
	}
}

func characterSymbol(name string, rng token.Range) protocol.DocumentSymbol {
	r := toLSPRange(rng)
	return protocol.DocumentSymbol{
		Name:           name,
		Kind:           protocol.SymbolKindFunction,
		Range:          r,
		SelectionRange: r,
	}
}

func characterNameFromNode(n ast.Node) string {
	switch v := n.(type) {
	case *ast.Dialogue:
		return v.Character
	default:
		return ""
	}
}

func sectionSymbolName(s *ast.Section) string {
	if s == nil {
		return "Section"
	}

	if s.Title != "" {
		return s.Title
	}

	if s.Number != "" {
		switch s.Kind {
		case ast.SectionAct:
			return "Act " + s.Number
		case ast.SectionScene:
			return "Scene " + s.Number
		default:
			return s.Number
		}
	}

	switch s.Kind {
	case ast.SectionAct:
		return "Act"
	case ast.SectionScene:
		return "Scene"
	case ast.SectionDramatisPersonae:
		return "Dramatis Personae"
	default:
		return "Section"
	}
}

func songSymbolName(s *ast.Song) string {
	if s == nil {
		return "Song"
	}

	if s.Title != "" {
		return s.Title + " (song)"
	}

	if s.Number != "" {
		return "Song " + s.Number + " (song)"
	}

	return "Song"
}
