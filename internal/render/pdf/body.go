package pdf

import (
	"fmt"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

// --- Section ---

func (r *pdfRenderer) BeginSection(s *ast.Section) error {
	switch s.Kind {
	case ast.SectionAct:
		return r.beginAct(s)
	case ast.SectionScene:
		return r.beginScene(s)
	case ast.SectionDramatisPersonae:
		renderDramatisPersonae(&r.pdfBase, s, r.bodyW*0.15)
		return nil
	default: // SectionGeneric
		// Skip play title heading if title page already rendered it
		if r.hasTitlePage && s.Level == 1 && strings.EqualFold(strings.TrimSpace(s.Title), r.titlePageTitle) {
			return nil
		}
		if s.Level == 0 {
			// Forced heading: inline, no page break
			r.ensureSpace(r.lineHeight * 3)
			r.pdf.Ln(r.lineHeight)
			r.setStyle("B")
			r.centeredText(s.Title)
			r.setStyle("")
			r.pdf.Ln(r.lineHeight)
			return nil
		}
		r.pdf.AddPage()
		if s.Title != "" {
			r.setStyle("B")
			r.centeredText(strings.ToUpper(s.Title))
			r.setStyle("")
			r.pdf.Ln(r.lineHeight * 2)
		}
		return nil
	}
}

func (r *pdfRenderer) EndSection(_ *ast.Section) error {
	return nil
}

func (r *pdfRenderer) BeginSectionLine(sl *ast.SectionLine) error {
	if len(sl.Content) == 0 {
		// Blank line = paragraph break
		r.pdf.Ln(r.lineHeight * 2)
	}
	return nil
}

func (r *pdfRenderer) EndSectionLine(sl *ast.SectionLine) error {
	if len(sl.Content) > 0 {
		// Single line break = space (prose reflow)
		r.pdf.Write(r.lineHeight, " ")
	}
	return nil
}

func (r *pdfRenderer) beginAct(s *ast.Section) error {
	// Numbered acts start on a new page. Title-only headings
	// (the play title parsed as an Act) are skipped when a title
	// page already rendered them.
	if s.Number != "" {
		r.pdf.AddPage()
	} else {
		if r.hasTitlePage {
			return nil
		}
		r.ensureSpace(r.lineHeight * 4)
		r.pdf.Ln(r.lineHeight * 2)
	}

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "ACT " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "ACT " + s.Number
	default:
		heading = s.Title
	}

	r.setStyle("BU")
	r.centeredText(strings.ToUpper(heading))
	r.setStyle("")
	r.pdf.Ln(r.lineHeight * 2)
	return nil
}

func (r *pdfRenderer) beginScene(s *ast.Section) error {
	r.ensureSpace(r.lineHeight * 3)
	r.pdf.Ln(r.lineHeight)

	var heading string
	switch {
	case s.Number != "" && s.Title != "":
		heading = "SCENE " + s.Number + ": " + s.Title
	case s.Number != "":
		heading = "SCENE " + s.Number
	default:
		heading = s.Title
	}

	r.setStyle("B")
	r.centeredText(strings.ToUpper(heading))
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Dialogue ---

func (r *pdfRenderer) BeginDialogue(d *ast.Dialogue) error {
	r.ensureSpace(r.lineHeight * 3)
	r.pdf.Ln(r.lineHeight)

	// Character name — centered, uppercase, bold
	r.setStyle("B")
	r.centeredText(strings.ToUpper(d.Character))
	r.setStyle("")

	// Parenthetical — centered, italic
	if d.Parenthetical != "" {
		r.setStyle("I")
		paren := d.Parenthetical
		if len(paren) == 0 || paren[0] != '(' {
			paren = "(" + paren + ")"
		}
		r.centeredText(paren)
		r.setStyle("")
	}

	// Set dialogue column margins
	dialogueMargin := r.bodyW * 0.15
	dialogueX := r.marginL + dialogueMargin
	r.pdf.SetLeftMargin(dialogueX)
	r.pdf.SetRightMargin(r.marginR + dialogueMargin)
	return nil
}

func (r *pdfRenderer) EndDialogue(_ *ast.Dialogue) error {
	r.pdf.SetLeftMargin(r.marginL)
	r.pdf.SetRightMargin(r.marginR)
	return nil
}

func (r *pdfRenderer) BeginDialogueLine(line *ast.DialogueLine) error {
	dialogueMargin := r.bodyW * 0.15
	dialogueX := r.marginL + dialogueMargin
	if line.IsVerse {
		r.pdf.SetX(dialogueX + 10)
	} else {
		r.pdf.SetX(dialogueX)
	}
	return nil
}

func (r *pdfRenderer) EndDialogueLine(_ *ast.DialogueLine) error {
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Stage Direction ---

func (r *pdfRenderer) BeginStageDirection(_ *ast.StageDirection) error {
	r.ensureSpace(r.lineHeight * 2)
	r.pdf.Ln(r.lineHeight / 2)
	r.setStyle("I")
	r.pdf.SetX(r.marginL)
	return nil
}

func (r *pdfRenderer) EndStageDirection(_ *ast.StageDirection) error {
	r.pdf.Ln(r.lineHeight)
	r.setStyle("")
	r.pdf.Ln(r.lineHeight / 2)
	return nil
}

// --- Song ---

func (r *pdfRenderer) BeginSong(song *ast.Song) error {
	r.ensureSpace(r.lineHeight * 3)
	r.pdf.Ln(r.lineHeight)

	header := "SONG"
	if song.Number != "" {
		header = fmt.Sprintf("SONG %s", song.Number)
	}
	if song.Title != "" {
		header += ": " + song.Title
	}

	r.setStyle("B")
	r.centeredText(header)
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

func (r *pdfRenderer) EndSong(_ *ast.Song) error {
	r.pdf.Ln(r.lineHeight / 2)
	r.setStyle("B")
	r.centeredText("SONG END")
	r.setStyle("")
	r.pdf.Ln(r.lineHeight)
	return nil
}

// --- Verse Block ---

func (r *pdfRenderer) BeginVerseBlock(_ *ast.VerseBlock) error {
	r.ensureSpace(r.lineHeight * 2)
	return nil
}

func (r *pdfRenderer) EndVerseBlock(_ *ast.VerseBlock) error {
	return nil
}

func (r *pdfRenderer) BeginVerseLine(_ *ast.VerseLine) error {
	verseX := r.marginL + r.bodyW*0.15 + 10
	r.pdf.SetX(verseX)
	return nil
}

func (r *pdfRenderer) EndVerseLine(_ *ast.VerseLine) error {
	r.pdf.Ln(r.lineHeight)
	return nil
}
