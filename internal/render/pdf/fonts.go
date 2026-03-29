package pdf

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-pdf/fpdf"
)

//go:embed fonts/CourierPrime-Regular.ttf
var courierPrimeRegular []byte

//go:embed fonts/CourierPrime-Bold.ttf
var courierPrimeBold []byte

//go:embed fonts/CourierPrime-Italic.ttf
var courierPrimeItalic []byte

//go:embed fonts/CourierPrime-BoldItalic.ttf
var courierPrimeBoldItalic []byte

//go:embed fonts/LibreBaskerville-Regular.ttf
var libreBaskervilleRegular []byte

//go:embed fonts/LibreBaskerville-Bold.ttf
var libreBaskervilleBold []byte

//go:embed fonts/LibreBaskerville-Italic.ttf
var libreBaskervilleItalic []byte

//go:embed fonts/LibreBaskerville-BoldItalic.ttf
var libreBaskervilleBoldItalic []byte

const defaultFontFamily = "CourierPrime"
const serifFontFamily = "LibreBaskerville"

// loadBundledFont registers the bundled Courier Prime font with fpdf.
func loadBundledFont(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes(defaultFontFamily, "", courierPrimeRegular)
	pdf.AddUTF8FontFromBytes(defaultFontFamily, "B", courierPrimeBold)
	pdf.AddUTF8FontFromBytes(defaultFontFamily, "I", courierPrimeItalic)
	pdf.AddUTF8FontFromBytes(defaultFontFamily, "BI", courierPrimeBoldItalic)
}

// loadSerifFont registers the bundled Libre Baskerville font with fpdf.
func loadSerifFont(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes(serifFontFamily, "", libreBaskervilleRegular)
	pdf.AddUTF8FontFromBytes(serifFontFamily, "B", libreBaskervilleBold)
	pdf.AddUTF8FontFromBytes(serifFontFamily, "I", libreBaskervilleItalic)
	pdf.AddUTF8FontFromBytes(serifFontFamily, "BI", libreBaskervilleBoldItalic)
}

// loadCustomFont loads a user-specified TTF font file and registers all
// available variants. It looks for Bold/Italic/BoldItalic variants using
// common naming conventions alongside the specified file.
func loadCustomFont(pdf *fpdf.Fpdf, family, path string) bool {
	cleanPath, err := validateCustomFontPath(path)
	if err != nil {
		slog.Error("invalid custom font path", "path", path, "error", err)
		return false
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		slog.Error("failed to read custom font", "path", cleanPath, "error", err)
		return false
	}
	pdf.AddUTF8FontFromBytes(family, "", data)
	if pdf.Err() {
		slog.Error("failed to register custom font", "path", cleanPath, "error", pdf.Error())
		pdf.ClearError()
		return false
	}

	// Try to find style variants by common naming patterns
	tryVariant(pdf, family, cleanPath, "B", []string{"-Bold", "bd", "-bold", "B"})
	tryVariant(pdf, family, cleanPath, "I", []string{"-Italic", "i", "-italic", "I", "-Oblique", "-oblique"})
	tryVariant(pdf, family, cleanPath, "BI", []string{"-BoldItalic", "bi", "-bolditalic", "BI", "-BoldOblique", "-boldoblique"})

	return true
}

// tryVariant attempts to load a font style variant by replacing the base
// filename stem with common suffixes.
func tryVariant(pdf *fpdf.Fpdf, family, basePath, style string, suffixes []string) {
	// Strip extension and any existing style suffix to get the base
	ext := ".ttf"
	base := basePath
	if len(base) > 4 && base[len(base)-4:] == ".ttf" {
		base = base[:len(base)-4]
	} else if len(base) > 4 && base[len(base)-4:] == ".TTF" {
		ext = ".TTF"
		base = base[:len(base)-4]
	}

	// Remove known regular-style suffixes from base
	for _, s := range []string{"-Regular", "-regular", "regular"} {
		if len(base) > len(s) && base[len(base)-len(s):] == s {
			base = base[:len(base)-len(s)]
			break
		}
	}

	for _, suffix := range suffixes {
		path := base + suffix + ext
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		pdf.AddUTF8FontFromBytes(family, style, data)
		if pdf.Err() {
			pdf.ClearError()
			continue
		}
		slog.Debug("loaded font variant", "style", style, "path", path)
		return
	}
}

func validateCustomFontPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("font path cannot be empty")
	}

	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return "", fmt.Errorf("font path cannot contain parent traversal")
		}
	}

	sanitized := filepath.Clean(path)
	return sanitized, nil
}
