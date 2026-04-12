package migrate

import (
	"strings"
)

func UpgradeV1ToV2(source string) (string, bool) {
	lines, hadTrailingNewline := splitLines(source)
	metadataStart, metadataEnd, metadata := extractLeadingMetadata(lines)
	dpStart, dpEnd := findTopLevelDramatisPersonae(lines, metadataEnd)

	firstPlayHeading := -1
	for i := metadataEnd; i < len(lines); i++ {
		if i >= dpStart && i < dpEnd {
			continue
		}
		if isHeading(lines[i], 1) {
			firstPlayHeading = i
			break
		}
	}

	if len(metadata) == 0 && dpStart < 0 {
		return source, false
	}

	title := strings.TrimSpace(metadata["Title"])
	output := make([]string, 0, len(lines)+4)

	if firstPlayHeading >= 0 {
		heading := strings.TrimSpace(lines[firstPlayHeading])
		output = append(output, heading)
	} else {
		if title == "" {
			title = "Untitled Play"
		}
		output = append(output, "# "+title)
	}

	metadataLines := buildMetadataLines(metadata, title)
	if len(metadataLines) > 0 {
		output = append(output, metadataLines...)
	}

	if dpStart >= 0 {
		if len(output) > 0 && output[len(output)-1] != "" {
			output = append(output, "")
		}
		output = append(output, upgradeDramatisPersonae(lines[dpStart:dpEnd])...)
	}

	bodyStart := metadataEnd
	if firstPlayHeading >= 0 {
		bodyStart = firstPlayHeading + 1
	}
	if dpEnd > bodyStart {
		bodyStart = dpEnd
	}
	body := collectBodyLines(lines, metadataStart, metadataEnd, dpStart, dpEnd, firstPlayHeading, bodyStart)
	if len(body) > 0 {
		if len(output) > 0 && output[len(output)-1] != "" && body[0] != "" {
			output = append(output, "")
		}
		output = append(output, body...)
	}

	upgraded := strings.Join(trimLeadingBlankLines(trimTrailingBlankLines(output)), "\n")
	if hadTrailingNewline {
		upgraded += "\n"
	}
	if upgraded == source {
		return source, false
	}
	return upgraded, true
}

func splitLines(source string) ([]string, bool) {
	hadTrailingNewline := strings.HasSuffix(source, "\n")
	lines := strings.Split(source, "\n")
	if hadTrailingNewline && len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	return lines, hadTrailingNewline
}

func extractLeadingMetadata(lines []string) (int, int, map[string]string) {
	metadata := make(map[string]string)
	start := -1
	i := 0
	for i < len(lines) {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}
		if isCommentLine(line) {
			return -1, 0, metadata
		}
		if !isMetadataLine(line) {
			return -1, 0, metadata
		}
		start = i
		break
	}
	if start < 0 {
		return -1, 0, metadata
	}

	end := start
	for end < len(lines) {
		line := lines[end]
		switch {
		case strings.TrimSpace(line) == "":
			end++
		case isMetadataLine(line):
			key, value := splitMetadataLine(line)
			metadata[key] = value
			end++
		case isMetadataContinuation(line):
			end++
		default:
			return start, end, metadata
		}
	}
	return start, end, metadata
}

func buildMetadataLines(metadata map[string]string, title string) []string {
	order := []string{"Subtitle", "Author", "Date", "Draft", "Copyright", "Contact", "Notes"}
	lines := make([]string, 0, len(metadata))
	for _, key := range order {
		if strings.EqualFold(key, "Title") {
			continue
		}
		value := strings.TrimSpace(metadata[key])
		if value == "" {
			continue
		}
		lines = append(lines, key+": "+value)
		delete(metadata, key)
	}
	for key, value := range metadata {
		if strings.EqualFold(key, "Title") || strings.TrimSpace(value) == "" {
			continue
		}
		lines = append(lines, key+": "+strings.TrimSpace(value))
	}
	return lines
}

func findTopLevelDramatisPersonae(lines []string, start int) (int, int) {
	for i := max(start, 0); i < len(lines); i++ {
		if !isHeading(lines[i], 1) {
			continue
		}
		if !isDramatisHeading(lines[i]) {
			continue
		}
		end := i + 1
		for end < len(lines) {
			if isHeading(lines[end], 1) {
				break
			}
			end++
		}
		return i, end
	}
	return -1, -1
}

func upgradeDramatisPersonae(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	out := []string{"## Dramatis Personae"}
	lastEntry := -1
	for _, line := range lines[1:] {
		switch {
		case strings.TrimSpace(line) == "":
			out = append(out, "")
		case isCommentLine(line):
			out = append(out, line)
		case isHeading(line, 2):
			out = append(out, "### "+strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "## ")))
			lastEntry = -1
		case isStandaloneAlias(line):
			if lastEntry >= 0 {
				if merged, ok := mergeAliasIntoEntry(out[lastEntry], line); ok {
					out[lastEntry] = merged
					continue
				}
			}
			out = append(out, strings.Trim(strings.TrimSpace(line), "[]"))
		default:
			out = append(out, normalizeDPEntryLine(line))
			lastEntry = len(out) - 1
		}
	}
	return trimTrailingBlankLines(out)
}

func collectBodyLines(lines []string, metadataStart, metadataEnd, dpStart, dpEnd, playHeading, bodyStart int) []string {
	if playHeading >= 0 {
		return trimLeadingBlankLines(lines[bodyStart:])
	}

	body := make([]string, 0, len(lines))
	for i, line := range lines {
		if metadataStart >= 0 && i >= metadataStart && i < metadataEnd {
			continue
		}
		if dpStart >= 0 && i >= dpStart && i < dpEnd {
			continue
		}
		body = append(body, line)
	}
	return trimLeadingBlankLines(body)
}

func isMetadataLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	idx := strings.Index(trimmed, ":")
	return idx > 0
}

func isMetadataContinuation(line string) bool {
	return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
}

func splitMetadataLine(line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	idx := strings.Index(trimmed, ":")
	return strings.TrimSpace(trimmed[:idx]), strings.TrimSpace(trimmed[idx+1:])
}

func isCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*")
}

func isHeading(line string, level int) bool {
	prefix := strings.Repeat("#", level) + " "
	return strings.HasPrefix(strings.TrimSpace(line), prefix)
}

func isDramatisHeading(line string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#")))
	trimmed = strings.TrimSpace(trimmed)
	return trimmed == "dramatis personae" || trimmed == "cast of characters" || trimmed == "characters"
}

func isStandaloneAlias(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") && strings.Contains(trimmed, "/")
}

func mergeAliasIntoEntry(entryLine string, aliasLine string) (string, bool) {
	aliasSpec := strings.Trim(strings.TrimSpace(aliasLine), "[]")
	aliasBase := strings.TrimSpace(strings.Split(aliasSpec, "/")[0])
	entry := normalizeDPEntryLine(entryLine)
	namePart := entry
	description := ""
	if idx := strings.Index(entry, " - "); idx >= 0 {
		namePart = strings.TrimSpace(entry[:idx])
		description = entry[idx:]
	}
	if !strings.EqualFold(strings.TrimSpace(namePart), aliasBase) {
		return "", false
	}
	return aliasSpec + description, true
}

func normalizeDPEntryLine(line string) string {
	line = strings.ReplaceAll(line, " — ", " - ")
	line = strings.ReplaceAll(line, " – ", " - ")
	return strings.TrimRight(line, " \t")
}

func trimLeadingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	return lines
}

func trimTrailingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
