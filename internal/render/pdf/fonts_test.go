package pdf

import "testing"

func TestValidateCustomFontPathRejectsTraversal(t *testing.T) {
	if _, err := validateCustomFontPath("../fonts/custom.ttf"); err == nil {
		t.Fatal("expected parent traversal path to be rejected")
	}
}

func TestValidateCustomFontPathRejectsWindowsTraversal(t *testing.T) {
	if _, err := validateCustomFontPath("..\\fonts\\custom.ttf"); err == nil {
		t.Fatal("expected windows parent traversal path to be rejected")
	}
}

func TestValidateCustomFontPathCleansValidPath(t *testing.T) {
	got, err := validateCustomFontPath("./fonts/Custom.ttf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fonts/Custom.ttf" {
		t.Fatalf("expected cleaned path, got %q", got)
	}
}

func TestValidateCustomFontPathRejectsEmptyPath(t *testing.T) {
	if _, err := validateCustomFontPath("   "); err == nil {
		t.Fatal("expected empty font path to be rejected")
	}
}
