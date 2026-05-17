package revisions

import (
	"github.com/jscaltreto/downstage/internal/ast"
)

type SynthOptions struct {
	// TrailingPlaceholders adds a REMOVED page after the indexed region.
	TrailingPlaceholders map[int]string
}

// SynthesizeDocument constructs an AST document that, when walked by the
// existing renderer, emits exactly the revision content for the given
// regions in order, with explicit PageBreaks between regions so each region
// starts on its own page.
//
// The synthetic document has no TitlePage, no Dramatis Personae, and no
// outline structure. Per-page top-margin annotations and per-page footer
// labels are delivered separately via render.Config callbacks.
//
// The returned regionAnchors slice has one entry per region pointing at the
// first AST node of that region — the orchestrator looks these up in a
// pagemap after rendering to compute how many internal pages each region
// produced.
func SynthesizeDocument(regions []Region, opts SynthOptions) (*ast.Document, []ast.Node) {
	doc := &ast.Document{}
	var anchors []ast.Node

	for i, r := range regions {
		if i > 0 {
			doc.Body = append(doc.Body, &ast.PageBreak{})
		}
		anchor := firstAnchor(r.V2Nodes)
		anchors = append(anchors, anchor)

		if len(r.V2Nodes) == 0 {
			placeholder := &ast.Section{
				Level: 0,
				Kind:  ast.SectionGeneric,
				Title: r.MarginNote,
			}
			doc.Body = append(doc.Body, placeholder)
			if anchor == nil {
				anchors[i] = placeholder
			}
			continue
		}
		doc.Body = append(doc.Body, r.V2Nodes...)

		if note, ok := opts.TrailingPlaceholders[i]; ok && note != "" {
			doc.Body = append(doc.Body,
				&ast.PageBreak{},
				&ast.Section{
					Level: 0,
					Kind:  ast.SectionGeneric,
					Title: note,
				},
			)
		}
	}
	return doc, anchors
}

func firstAnchor(nodes []ast.Node) ast.Node {
	for _, n := range nodes {
		if n != nil {
			return n
		}
	}
	return nil
}
