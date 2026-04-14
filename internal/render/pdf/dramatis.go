package pdf

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

// renderDramatisPersonae renders the characters and groups from a
// SectionDramatisPersonae section. Called by both pdfRenderer and
// condensedRenderer from their BeginSection methods.
func renderDramatisPersonae(b *pdfBase, s *ast.Section, charIndent float64) {
	// Heading
	b.setStyle("B")
	b.pdf.Ln(b.lineHeight)
	b.centeredText(strings.ToUpper(render.DramatisPersonaeDisplayTitle(s)))
	b.pdf.Ln(b.lineHeight * 2)
	b.setStyle("")

	// Ungrouped characters
	for _, ch := range s.Characters {
		renderCharacterEntry(b, ch, charIndent)
	}

	// Character groups
	for _, group := range s.Groups {
		b.pdf.Ln(b.lineHeight)
		b.setStyle("B")
		b.centeredText(group.Name)
		b.setStyle("")
		b.pdf.Ln(b.lineHeight)

		for _, ch := range group.Characters {
			renderCharacterEntry(b, ch, charIndent)
		}
	}
}

func renderCharacterEntry(b *pdfBase, ch ast.Character, indent float64) {
	b.ensureSpace(b.lineHeight * 2)
	b.pdf.SetX(b.marginL + indent)

	b.setStyle("B")
	b.pdf.Write(b.lineHeight, render.CharacterDisplayName(ch))
	b.setStyle("")

	if ch.Description != "" {
		b.pdf.Write(b.lineHeight, " \u2014 "+ch.Description)
	}

	b.pdf.Ln(b.lineHeight)
}
