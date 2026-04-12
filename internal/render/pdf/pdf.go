package pdf

import (
	"io"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/render"
)

const manuscriptLineHeight = 5.0 // mm

var _ render.NodeRenderer = (*pdfRenderer)(nil)
var _ dialoguePaginationStrategy = (*pdfRenderer)(nil)

// NewRenderer creates a manuscript-style PDF NodeRenderer.
func NewRenderer(cfg render.Config) render.NodeRenderer {
	return &pdfRenderer{pdfBase: pdfBase{cfg: cfg, lineHeight: manuscriptLineHeight}}
}

type pdfRenderer struct {
	pdfBase
	activeDialogue *bufferedDialogue
}

// --- Lifecycle ---

func (r *pdfRenderer) BeginDocument(doc *ast.Document, w io.Writer) error {
	r.w = w
	tp := render.DocumentTitlePage(doc)
	r.hasTitlePage = tp != nil
	r.hasBody = render.DocumentHasRenderableBody(doc)
	r.titlePageTitle = titlePageTitle(tp)
	r.initPDF(loadBundledFont, defaultFontFamily)
	applyDocumentMetadata(&r.pdfBase, tp)
	r.inlinePlaySections = make(map[*ast.Section]bool)
	for _, section := range render.PlayableTopLevelSections(doc) {
		if render.IsInlinePlaySection(doc, section) {
			r.inlinePlaySections[section] = true
		}
	}
	return nil
}

func (r *pdfRenderer) EndDocument(_ *ast.Document) error {
	return r.pdf.Output(r.w)
}

func titlePageTitle(tp *ast.TitlePage) string {
	if tp == nil {
		return ""
	}
	for _, entry := range tp.Entries {
		if strings.EqualFold(strings.TrimSpace(entry.Key), "title") {
			return strings.TrimSpace(entry.Value)
		}
	}
	return ""
}
