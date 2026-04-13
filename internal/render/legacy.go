package render

import (
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

func IsLegacyTopLevelDramatisPersonae(section *ast.Section) bool {
	if section == nil || section.Level != 1 || section.Kind != ast.SectionGeneric {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(section.Title)) {
	case "dramatis personae", "cast of characters", "characters":
		return true
	default:
		return false
	}
}
