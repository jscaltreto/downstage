package render

import "testing"

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
