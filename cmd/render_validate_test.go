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
	if err := os.WriteFile(input, []byte("ALICE\nHello."), 0o644); err != nil {
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

func TestRunRenderStdoutRejectsPDF(t *testing.T) {
	resetRenderFlags()
	renderStdout = true
	renderFormat = "pdf"

	err := runRender(&cobra.Command{}, []string{"test.ds"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--stdout is not supported for pdf output")
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
