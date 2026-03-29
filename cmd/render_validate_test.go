package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

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
