package lsp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"go.lsp.dev/protocol"
)

// checkMissingDramatisPersonae reports a document with dialogue but no DP.
func checkMissingDramatisPersonae(doc *ast.Document, index *documentIndex) []protocol.Diagnostic {
	if doc == nil || index == nil {
		return nil
	}
	if index.hasDramatisPersonae {
		return nil
	}
	if len(index.dialogues) == 0 {
		return nil
	}

	first := index.dialogues[0].dialogue
	return []protocol.Diagnostic{{
		Range:    toLSPRange(first.NameRange()),
		Severity: protocol.DiagnosticSeverityInformation,
		Code:     diagnosticCodeMissingDramatisPersonae,
		Source:   "downstage",
		Message:  "no Dramatis Personae section: add one to document the cast and enable character checks",
	}}
}

// checkDPDuplicates flags duplicate entries within a single DP scope.
func checkDPDuplicates(index *documentIndex) []protocol.Diagnostic {
	if index == nil {
		return nil
	}

	var diags []protocol.Diagnostic
	scopes := collectScopes(index)
	// Sort the scope traversal by DP line to keep diagnostic order stable
	// across runs.
	sort.SliceStable(scopes, func(i, j int) bool {
		return scopes[i].dp.Range.Start.Line < scopes[j].dp.Range.Start.Line
	})

	for _, scope := range scopes {
		nameFlagged := make(map[int]struct{})
		for key, occurrences := range scope.nameKeyOccurrences {
			if len(occurrences) < 2 {
				continue
			}
			for _, entryIdx := range occurrences[1:] {
				entry := scope.entries[entryIdx]
				nameFlagged[entryIdx] = struct{}{}
				diags = append(diags, protocol.Diagnostic{
					Range:    toLSPRange(entry.character.Range),
					Severity: protocol.DiagnosticSeverityWarning,
					Code:     diagnosticCodeDPDuplicateCharacterName,
					Source:   "downstage",
					Message:  fmt.Sprintf("duplicate character entry %q in Dramatis Personae", displayName(entry.character.Name, key)),
					Data: map[string]string{
						"character": entry.character.Name,
					},
				})
			}
		}

		for key, occurrences := range scope.aliasKeyOccurrences {
			if len(occurrences) < 2 {
				continue
			}
			hasAlias := false
			for _, occ := range occurrences {
				if occ.aliasIndex >= 0 {
					hasAlias = true
					break
				}
			}
			if !hasAlias {
				continue
			}
			for _, occ := range occurrences[1:] {
				if _, ok := nameFlagged[occ.entryIndex]; ok && occ.aliasIndex == -1 {
					continue
				}
				entry := scope.entries[occ.entryIndex]
				label := displayName(entry.character.Name, key)
				if occ.aliasIndex >= 0 && occ.aliasIndex < len(entry.character.Aliases) {
					label = entry.character.Aliases[occ.aliasIndex]
				}
				diags = append(diags, protocol.Diagnostic{
					Range:    toLSPRange(entry.character.Range),
					Severity: protocol.DiagnosticSeverityWarning,
					Code:     diagnosticCodeDPDuplicateAlias,
					Source:   "downstage",
					Message:  fmt.Sprintf("duplicate alias %q collides with another Dramatis Personae entry", label),
					Data: map[string]string{
						"alias": label,
					},
				})
			}
		}
	}

	return diags
}

// checkDPCharacterNoDialogue flags DP entries whose name and aliases never
// appear as a cue within the enclosing play.
func checkDPCharacterNoDialogue(index *documentIndex) []protocol.Diagnostic {
	if index == nil {
		return nil
	}

	var diags []protocol.Diagnostic

	record := func(scope characterScope, used map[string]struct{}) {
		for _, entry := range scope.entries {
			if entry.nameKey == "" && len(entry.aliasKeys) == 0 {
				continue
			}
			if hasAnyKey(used, entry.nameKey, entry.aliasKeys) {
				continue
			}
			diags = append(diags, protocol.Diagnostic{
				Range:    toLSPRange(entry.character.Range),
				Severity: protocol.DiagnosticSeverityInformation,
				Code:     diagnosticCodeDPCharacterNoDialogue,
				Source:   "downstage",
				Message:  fmt.Sprintf("character %q is in Dramatis Personae but never speaks", entry.character.Name),
				Data: map[string]string{
					"character": entry.character.Name,
				},
			})
		}
	}

	for play, scope := range index.characterScopes {
		if scope.dp == nil {
			continue
		}
		record(scope, index.usedCharactersByPlay[play])
	}
	if index.legacyCharacterScope.dp != nil {
		union := make(map[string]struct{})
		for _, bucket := range index.usedCharactersByPlay {
			for k := range bucket {
				union[k] = struct{}{}
			}
		}
		record(index.legacyCharacterScope, union)
	}

	return diags
}

// checkOrphanedCues flags cues that have no dialogue content following them.
func checkOrphanedCues(index *documentIndex) []protocol.Diagnostic {
	if index == nil {
		return nil
	}

	var diags []protocol.Diagnostic
	for _, ref := range index.dialogues {
		if ref.dialogue == nil {
			continue
		}
		if len(ref.dialogue.Lines) > 0 {
			continue
		}
		diags = append(diags, protocol.Diagnostic{
			Range:    toLSPRange(ref.dialogue.NameRange()),
			Severity: protocol.DiagnosticSeverityWarning,
			Code:     diagnosticCodeCueOrphaned,
			Source:   "downstage",
			Message:  fmt.Sprintf("cue %q has no dialogue", ref.dialogue.Character),
			Data: map[string]string{
				"character": ref.dialogue.Character,
			},
		})
	}
	return diags
}

// checkConsecutiveSameCharacterCues flags repeated cues with no break in between.
func checkConsecutiveSameCharacterCues(index *documentIndex) []protocol.Diagnostic {
	if index == nil {
		return nil
	}

	containers := make([]*ast.Section, 0, len(index.containerEvents))
	for container := range index.containerEvents {
		containers = append(containers, container)
	}
	sort.SliceStable(containers, func(i, j int) bool {
		return containers[i].Range.Start.Line < containers[j].Range.Start.Line
	})

	var diags []protocol.Diagnostic
	for _, container := range containers {
		events := index.containerEvents[container]
		lastCueKey := ""
		var lastCueCharacter string
		for _, ev := range events {
			switch ev.kind {
			case containerEventBreak:
				lastCueKey = ""
				lastCueCharacter = ""
			case containerEventCue:
				key := strings.ToUpper(strings.TrimSpace(ev.character))
				if key != "" && key == lastCueKey {
					diags = append(diags, protocol.Diagnostic{
						Range:    toLSPRange(ev.nameRange),
						Severity: protocol.DiagnosticSeverityInformation,
						Code:     diagnosticCodeCueConsecutiveSameCharacter,
						Source:   "downstage",
						Message:  fmt.Sprintf("cue %q repeats without an intervening stage direction or break", ev.character),
						Data: map[string]string{
							"character":         ev.character,
							"previousCharacter": lastCueCharacter,
						},
					})
				}
				lastCueKey = key
				lastCueCharacter = ev.character
			}
		}
	}
	return diags
}

func collectScopes(index *documentIndex) []characterScope {
	scopes := make([]characterScope, 0, len(index.characterScopes)+1)
	for _, scope := range index.characterScopes {
		if scope.dp == nil {
			continue
		}
		scopes = append(scopes, scope)
	}
	if index.legacyCharacterScope.dp != nil {
		scopes = append(scopes, index.legacyCharacterScope)
	}
	return scopes
}

func hasAnyKey(set map[string]struct{}, name string, aliases []string) bool {
	if set == nil {
		return false
	}
	if name != "" {
		if _, ok := set[name]; ok {
			return true
		}
	}
	for _, alias := range aliases {
		if _, ok := set[alias]; ok {
			return true
		}
	}
	return false
}

// displayName chooses a readable label for a duplicate diagnostic: the
// authored name when present, otherwise the uppercase key.
func displayName(name, fallbackKey string) string {
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return trimmed
	}
	return fallbackKey
}
