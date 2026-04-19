package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func resetRenderFlags() {
	renderFormat = "pdf"
	renderOutput = ""
	renderPageSize = "letter"
	renderStyle = "standard"
	renderLayout = "single"
	renderGutter = "0.125in"
	renderFont = ""
	renderStdin = false
	renderStdout = false
	renderSourceName = "<stdin>"
	renderSourceAnchors = false
}

func TestRunRenderFailsOnParseErrors(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	renderFormat = "pdf"
	renderOutput = t.TempDir() + "/out.pdf"
	renderPageSize = "letter"
	renderStyle = "standard"
	renderFont = ""

	input := t.TempDir() + "/broken.ds"
	if err := os.WriteFile(input, []byte("SONG\nALICE\nHello."), 0o644); err != nil {
		t.Fatal(err)
	}

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil {
		t.Fatal("expected parse failure")
	}
}

func TestRunValidateReturnsError(t *testing.T) {
	input := t.TempDir() + "/broken.ds"
	if err := os.WriteFile(input, []byte("SONG\nALICE\nHello."), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&out)

	err := runValidate(command, []string{input})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(out.String(), "error:") {
		t.Fatalf("expected diagnostics, got %q", out.String())
	}
}

func TestRunRenderRejectsHTMLOnlyPDFFlags(t *testing.T) {
	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Play\n\nALICE\nHello."), 0o644); err != nil {
		t.Fatal(err)
	}

	renderFormat = "html"
	renderOutput = t.TempDir() + "/out.html"
	renderPageSize = "a4"
	renderStyle = "standard"
	renderFont = ""

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "--page-size is only supported for pdf output") {
		t.Fatalf("expected html page-size rejection, got %v", err)
	}

	renderPageSize = "letter"
	renderFont = "/tmp/custom.ttf"

	err = runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "--font is only supported for pdf output") {
		t.Fatalf("expected html font rejection, got %v", err)
	}
}

func TestRunRenderRejectsStandard2Up(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Play\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	renderStyle = "standard"
	renderLayout = "2up"
	renderOutput = t.TempDir() + "/out.pdf"

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "only supported for style") {
		t.Fatalf("expected condensed-only rejection, got %v", err)
	}
}

func TestRunRenderRejectsStandardBooklet(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Play\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	renderStyle = "standard"
	renderLayout = "booklet"
	renderOutput = t.TempDir() + "/out.pdf"

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "only supported for style") {
		t.Fatalf("expected condensed-only rejection, got %v", err)
	}
}

func TestRunRenderRejectsGutterOnNonBooklet(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Play\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	renderStyle = "condensed"
	renderLayout = "2up"
	renderGutter = "3mm"
	renderOutput = t.TempDir() + "/out.pdf"

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "--gutter is only valid") {
		t.Fatalf("expected gutter/2up rejection, got %v", err)
	}
}

func TestRunRenderRejectsLayoutForHTML(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Play\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	renderFormat = "html"
	renderLayout = "2up"
	renderOutput = t.TempDir() + "/out.html"

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "--pdf-layout is only supported for pdf output") {
		t.Fatalf("expected html layout rejection, got %v", err)
	}
}

func TestRunRenderRejectsInvalidLayout(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Play\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	renderLayout = "4up"

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "unsupported pdf layout") {
		t.Fatalf("expected layout parse error, got %v", err)
	}
}

func TestRunRenderRejectsInvalidGutter(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Play\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	renderStyle = "condensed"
	renderLayout = "booklet"
	renderGutter = "3cm"
	renderOutput = t.TempDir() + "/out.pdf"

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "--gutter") {
		t.Fatalf("expected gutter parse error, got %v", err)
	}
}

func TestRunRenderSupportsCondensed2Up(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	body := "# Test\n\n## ACT I\n\n### SCENE 1\n\nALICE\nLine 1.\n\nBOB\nLine 2.\n\n"
	if err := os.WriteFile(input, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	renderStyle = "condensed"
	renderLayout = "2up"
	renderStdout = true

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	renderErr := runRender(&cobra.Command{}, []string{input})

	w.Close()
	os.Stdout = origStdout

	if renderErr != nil {
		t.Fatalf("expected condensed 2up to succeed, got: %v", renderErr)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-") {
		t.Fatalf("expected PDF magic bytes, got %q", buf.String()[:20])
	}
}

func TestRunRenderSupportsCondensedBooklet(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	body := "# Test\n\n## ACT I\n\n### SCENE 1\n\nALICE\nLine 1.\n\nBOB\nLine 2.\n\n"
	if err := os.WriteFile(input, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	renderStyle = "condensed"
	renderLayout = "booklet"
	renderGutter = "3mm"
	renderStdout = true

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	renderErr := runRender(&cobra.Command{}, []string{input})

	w.Close()
	os.Stdout = origStdout

	if renderErr != nil {
		t.Fatalf("expected condensed booklet to succeed, got: %v", renderErr)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-") {
		t.Fatalf("expected PDF magic bytes, got %q", buf.String()[:20])
	}
}

func TestRunRenderRejectsUnsupportedPageSize(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("ALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	renderPageSize = "legal"
	renderOutput = t.TempDir() + "/out.pdf"

	err := runRender(&cobra.Command{}, []string{input})
	if err == nil || !strings.Contains(err.Error(), "unsupported page size") {
		t.Fatalf("expected unsupported page size error, got %v", err)
	}
}

func TestRunRenderSupportsA4CondensedPDF(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Test\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	renderStyle = "condensed"
	renderPageSize = "a4"
	renderStdout = true

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	renderErr := runRender(&cobra.Command{}, []string{input})

	w.Close()
	os.Stdout = origStdout

	if renderErr != nil {
		t.Fatalf("expected A4 condensed PDF render to succeed, got: %v", renderErr)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-") {
		t.Fatalf("expected PDF magic bytes, got %q", buf.String()[:20])
	}
}

func TestRunRenderStdoutAcceptsPDF(t *testing.T) {
	resetRenderFlags()

	input := t.TempDir() + "/play.ds"
	if err := os.WriteFile(input, []byte("# Test\n\nALICE\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	renderStdout = true
	renderFormat = "pdf"

	// Capture stdout
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	renderErr := runRender(&cobra.Command{}, []string{input})

	w.Close()
	os.Stdout = origStdout

	if renderErr != nil {
		t.Fatalf("expected PDF stdout to succeed, got: %v", renderErr)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-") {
		t.Fatalf("expected PDF magic bytes, got %q", buf.String()[:20])
	}
}

func TestRunRenderStdinRejectsFileArgs(t *testing.T) {
	resetRenderFlags()
	renderStdin = true
	renderStdout = true

	err := runRender(&cobra.Command{}, []string{"test.ds"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--stdin does not accept file arguments")
}

func TestRunRenderRequiresOneArgWithoutStdin(t *testing.T) {
	resetRenderFlags()

	err := runRender(&cobra.Command{}, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

func TestRunRenderStdinRequiresExplicitOutputTarget(t *testing.T) {
	resetRenderFlags()
	renderStdin = true
	renderFormat = "html"

	err := runRender(&cobra.Command{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--stdin requires --stdout or --output")
}
