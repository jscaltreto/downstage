package render

import (
	"errors"
	"fmt"
	"strings"
)

// Style represents a rendering style variant.
type Style string

const (
	StyleStandard  Style = "standard"
	StyleCondensed Style = "condensed"
)

// ParseStyle converts a string to a Style.
func ParseStyle(s string) (Style, error) {
	switch Style(s) {
	case StyleStandard, StyleCondensed:
		return Style(s), nil
	default:
		return "", fmt.Errorf("unsupported style: %q", s)
	}
}

// PageSize represents the paper size.
type PageSize string

const (
	PageLetter PageSize = "letter"
	PageA4     PageSize = "a4"
)

// Dimensions describes a physical or logical page size in millimeters.
type Dimensions struct {
	WidthMM  float64
	HeightMM float64
}

// ParsePageSize converts a string to a PageSize. Accepts both "a4" and "A4"
// spellings but normalizes to the canonical lowercase constant.
func ParsePageSize(s string) (PageSize, error) {
	switch {
	case strings.EqualFold(s, string(PageLetter)):
		return PageLetter, nil
	case strings.EqualFold(s, string(PageA4)):
		return PageA4, nil
	default:
		return "", fmt.Errorf("unsupported page size: %q", s)
	}
}

// SheetDimensions returns the physical sheet size in millimeters for standard
// PDF rendering.
func (p PageSize) SheetDimensions() (Dimensions, error) {
	switch p {
	case PageLetter:
		return Dimensions{WidthMM: 215.9, HeightMM: 279.4}, nil
	case PageA4:
		return Dimensions{WidthMM: 210, HeightMM: 297}, nil
	default:
		return Dimensions{}, fmt.Errorf("unsupported page size: %q", p)
	}
}

// CondensedPageDimensions returns the acting-edition logical page size derived
// from the selected physical sheet: half-letter for Letter, A5 for A4.
func (p PageSize) CondensedPageDimensions() (Dimensions, error) {
	switch p {
	case PageLetter:
		return Dimensions{WidthMM: 139.7, HeightMM: 215.9}, nil
	case PageA4:
		return Dimensions{WidthMM: 148, HeightMM: 210}, nil
	default:
		return Dimensions{}, fmt.Errorf("unsupported page size: %q", p)
	}
}

// Config holds rendering configuration.
type Config struct {
	PageSize      PageSize
	Style         Style
	FontFamily    string
	FontPath      string // path to a custom TTF font file (optional)
	FontSize      float64
	MarginTop     float64 // points (72 points = 1 inch)
	MarginBottom  float64
	MarginLeft    float64
	MarginRight   float64
	SourceAnchors bool // emit data-source-line attributes on block elements
}

// Validate checks that Config values are within acceptable ranges.
func (c Config) Validate() error {
	var errs []error
	if c.FontSize <= 0 {
		errs = append(errs, fmt.Errorf("FontSize must be > 0, got %g", c.FontSize))
	}
	if c.MarginTop < 0 {
		errs = append(errs, fmt.Errorf("MarginTop must be >= 0, got %g", c.MarginTop))
	}
	if c.MarginBottom < 0 {
		errs = append(errs, fmt.Errorf("MarginBottom must be >= 0, got %g", c.MarginBottom))
	}
	if c.MarginLeft < 0 {
		errs = append(errs, fmt.Errorf("MarginLeft must be >= 0, got %g", c.MarginLeft))
	}
	if c.MarginRight < 0 {
		errs = append(errs, fmt.Errorf("MarginRight must be >= 0, got %g", c.MarginRight))
	}
	switch c.PageSize {
	case PageLetter, PageA4:
	default:
		errs = append(errs, fmt.Errorf("unsupported PageSize: %q", c.PageSize))
	}
	switch c.Style {
	case StyleStandard, StyleCondensed:
	default:
		errs = append(errs, fmt.Errorf("unsupported Style: %q", c.Style))
	}
	return errors.Join(errs...)
}

// DefaultConfig returns a Config with standard play manuscript settings.
func DefaultConfig() Config {
	return Config{
		PageSize:     PageLetter,
		Style:        StyleStandard,
		FontFamily:   "Courier",
		FontSize:     12,
		MarginTop:    72,
		MarginBottom: 72,
		MarginLeft:   72,
		MarginRight:  72,
	}
}
