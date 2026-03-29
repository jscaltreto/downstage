package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStyle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Style
		wantErr bool
	}{
		{name: "standard", input: "standard", want: StyleStandard},
		{name: "condensed", input: "condensed", want: StyleCondensed},
		{name: "unsupported", input: "wide", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStyle(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStyle: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParsePageSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    PageSize
		wantErr bool
	}{
		{name: "letter lowercase", input: "letter", want: PageLetter},
		{name: "letter title case", input: "Letter", want: PageLetter},
		{name: "a4 lowercase", input: "a4", want: PageA4},
		{name: "a4 upper", input: "A4", want: PageA4},
		{name: "unsupported", input: "legal", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePageSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePageSize: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	got := DefaultConfig()

	if got.PageSize != PageLetter {
		t.Fatalf("expected default page size %q, got %q", PageLetter, got.PageSize)
	}
	if got.Style != StyleStandard {
		t.Fatalf("expected default style %q, got %q", StyleStandard, got.Style)
	}
	if got.FontFamily != "Courier" {
		t.Fatalf("expected Courier font family, got %q", got.FontFamily)
	}
	if got.FontSize != 12 {
		t.Fatalf("expected 12 point font, got %v", got.FontSize)
	}
	if got.MarginTop != 72 || got.MarginBottom != 72 || got.MarginLeft != 72 || got.MarginRight != 72 {
		t.Fatalf("unexpected default margins: %#v", got)
	}
	if got.SourceAnchors {
		t.Fatal("expected source anchors to be disabled by default")
	}
}

func TestDefaultConfigValidates(t *testing.T) {
	cfg := DefaultConfig()
	require.NoError(t, cfg.Validate())
}

func TestCondensedMarginsValidate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Style = StyleCondensed
	cfg.MarginTop = 36
	cfg.MarginBottom = 36
	cfg.MarginLeft = 36
	cfg.MarginRight = 36
	require.NoError(t, cfg.Validate())
}

func TestValidateRejectsZeroFontSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FontSize = 0
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FontSize")
}

func TestValidateRejectsNegativeFontSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FontSize = -1
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FontSize")
}

func TestValidateRejectsNegativeMargins(t *testing.T) {
	for _, field := range []string{"Top", "Bottom", "Left", "Right"} {
		t.Run("Margin"+field, func(t *testing.T) {
			cfg := DefaultConfig()
			switch field {
			case "Top":
				cfg.MarginTop = -1
			case "Bottom":
				cfg.MarginBottom = -1
			case "Left":
				cfg.MarginLeft = -1
			case "Right":
				cfg.MarginRight = -1
			}
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Margin"+field)
		})
	}
}

func TestValidateRejectsUnknownPageSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PageSize = "tabloid"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PageSize")
}

func TestValidateRejectsUnknownStyle(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Style = "fancy"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Style")
}

func TestValidateCollectsMultipleErrors(t *testing.T) {
	cfg := Config{
		FontSize:     -1,
		MarginTop:    -1,
		MarginBottom: -1,
		MarginLeft:   -1,
		MarginRight:  -1,
		PageSize:     "bad",
		Style:        "bad",
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FontSize")
	assert.Contains(t, err.Error(), "PageSize")
	assert.Contains(t, err.Error(), "Style")
	assert.Contains(t, err.Error(), "MarginTop")
}

func TestValidateAcceptsZeroMargins(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MarginTop = 0
	cfg.MarginBottom = 0
	cfg.MarginLeft = 0
	cfg.MarginRight = 0
	require.NoError(t, cfg.Validate())
}
