package lsp

import (
	"sort"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
	"github.com/jscaltreto/downstage/internal/token"
)

type dialogueRef struct {
	dialogue *ast.Dialogue
	scene    *ast.Section
	play     *ast.Section
}

type sceneSpeakerCue struct {
	line int
	name string
}

// containerEventKind classifies entries in the per-container event stream
// used by the cue-consecutive-same-character diagnostic.
type containerEventKind int

const (
	containerEventCue containerEventKind = iota
	// containerEventBreak signals any structural element that severs the
	// "two cues for the same character, back-to-back" pattern: a standalone
	// stage direction, callout, song, page break, verse block, or nested
	// section heading encountered within the container.
	containerEventBreak
)

type containerEvent struct {
	kind      containerEventKind
	line      int
	character string
	nameRange token.Range
}

type documentIndex struct {
	acts                   []*ast.Section
	scenes                 []*ast.Section
	topLevelSections       []*ast.Section
	actNumbers             map[*ast.Section]int
	sceneActs              map[*ast.Section]*ast.Section
	sceneNumbers           map[*ast.Section]int
	actPlays               map[*ast.Section]*ast.Section
	actsByPlay             map[*ast.Section][]*ast.Section
	characterCueLines      map[int]struct{}
	documentCharacterNames []string
	knownCharacters        map[string]struct{}
	characterScopes        map[*ast.Section]characterScope
	legacyCharacterScope   characterScope
	dialogues              []dialogueRef
	sceneSpeakers          map[*ast.Section][]sceneSpeakerCue
	// usedCharactersByPlay maps a top-level play section to the set of
	// uppercase character keys that appear as cues within it. Populated
	// during the dialogue walk and consulted by the
	// dp-character-no-dialogue check. Forced cues count as usage.
	usedCharactersByPlay map[*ast.Section]map[string]struct{}
	// containerEvents is an ordered stream of cue + break events keyed by
	// the nearest enclosing container section (scene, else act, else
	// top-level play). Drives the cue-consecutive-same-character check.
	containerEvents     map[*ast.Section][]containerEvent
	hasDramatisPersonae bool
}

func newDocumentIndex(doc *ast.Document) *documentIndex {
	index := &documentIndex{
		characterCueLines:    make(map[int]struct{}),
		knownCharacters:      make(map[string]struct{}),
		characterScopes:      make(map[*ast.Section]characterScope),
		actNumbers:           make(map[*ast.Section]int),
		sceneSpeakers:        make(map[*ast.Section][]sceneSpeakerCue),
		sceneActs:            make(map[*ast.Section]*ast.Section),
		sceneNumbers:         make(map[*ast.Section]int),
		actPlays:             make(map[*ast.Section]*ast.Section),
		actsByPlay:           make(map[*ast.Section][]*ast.Section),
		usedCharactersByPlay: make(map[*ast.Section]map[string]struct{}),
		containerEvents:      make(map[*ast.Section][]containerEvent),
	}
	if doc == nil {
		return index
	}

	seenNames := make(map[string]struct{})
	addDocumentCharacter := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToUpper(name)
		if _, ok := seenNames[key]; ok {
			return
		}
		seenNames[key] = struct{}{}
		index.documentCharacterNames = append(index.documentCharacterNames, name)
	}

	for _, node := range doc.Body {
		section, ok := node.(*ast.Section)
		if !ok || section.Level != 1 {
			continue
		}
		scope := newCharacterScope(ast.FindDramatisPersonaeInSection(section))
		if scope.dp != nil {
			index.characterScopes[section] = scope
			index.hasDramatisPersonae = true
			for _, name := range scope.names {
				addDocumentCharacter(name)
			}
			for key := range scope.known {
				index.knownCharacters[key] = struct{}{}
			}
		}
	}

	// When no top-level section owns a Dramatis Personae the document is
	// either V1-shaped (doc-level DP) or DP-free. In that case fall back to a
	// document-wide scope. In compilations where at least one play has a DP,
	// plays without one intentionally get no scope — that keeps scoping rules
	// self-contained per play rather than leaking names across the collection.
	if len(index.characterScopes) == 0 {
		index.legacyCharacterScope = newCharacterScope(ast.FindDramatisPersonae(doc.Body))
	}
	if index.legacyCharacterScope.dp != nil {
		index.hasDramatisPersonae = true
		for _, name := range index.legacyCharacterScope.names {
			addDocumentCharacter(name)
		}
		for key := range index.legacyCharacterScope.known {
			index.knownCharacters[key] = struct{}{}
		}
	}

	actCountsByPlay := make(map[*ast.Section]int)
	sceneCountsByAct := make(map[*ast.Section]int)
	sceneCountsByPlay := make(map[*ast.Section]int)

	// innerContainer picks the most specific container for cue-event
	// bucketing: scene > act > play. Events in the containerEvents stream
	// reset at each container boundary by virtue of being keyed per section.
	innerContainer := func(play, act, scene *ast.Section) *ast.Section {
		switch {
		case scene != nil:
			return scene
		case act != nil:
			return act
		default:
			return play
		}
	}

	// addUsedCharacter records a cue's character name under its enclosing
	// play. A nil play (dialogue outside any H1) buckets under the nil key,
	// which the legacy-scope branch of the no-dialogue check consults.
	addUsedCharacter := func(play *ast.Section, name string) {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return
		}
		key := strings.ToUpper(trimmed)
		set, ok := index.usedCharactersByPlay[play]
		if !ok {
			set = make(map[string]struct{})
			index.usedCharactersByPlay[play] = set
		}
		set[key] = struct{}{}
	}

	recordBreak := func(container *ast.Section, line int) {
		if container == nil {
			return
		}
		index.containerEvents[container] = append(index.containerEvents[container], containerEvent{
			kind: containerEventBreak,
			line: line,
		})
	}

	recordCue := func(container *ast.Section, dlg *ast.Dialogue) {
		if container == nil {
			return
		}
		nameRange := dlg.NameRange()
		index.containerEvents[container] = append(index.containerEvents[container], containerEvent{
			kind:      containerEventCue,
			line:      nameRange.Start.Line,
			character: dlg.Character,
			nameRange: nameRange,
		})
	}

	var walkNode func(ast.Node, *ast.Section, *ast.Section, *ast.Section)
	walkNode = func(node ast.Node, currentTopLevel *ast.Section, currentAct *ast.Section, currentScene *ast.Section) {
		switch v := node.(type) {
		case *ast.Dialogue:
			index.characterCueLines[v.NameRange().Start.Line] = struct{}{}
			ref := dialogueRef{dialogue: v, scene: currentScene, play: currentTopLevel}
			index.dialogues = append(index.dialogues, ref)
			if len(v.Lines) > 0 {
				addDocumentCharacter(v.Character)
			}
			if currentScene != nil && len(v.Lines) > 0 {
				index.sceneSpeakers[currentScene] = append(index.sceneSpeakers[currentScene], sceneSpeakerCue{
					line: v.NameRange().Start.Line,
					name: v.Character,
				})
			}
			// Track the cue for dp-character-no-dialogue (forced cues count,
			// per the DP "is the character used in this play" semantic).
			addUsedCharacter(currentTopLevel, v.Character)
			for _, part := range splitConjunctionCue(v.Character) {
				addUsedCharacter(currentTopLevel, part)
			}
			recordCue(innerContainer(currentTopLevel, currentAct, currentScene), v)
		case *ast.DualDialogue:
			// A DualDialogue breaks the "same character back-to-back" chain
			// on either side: the author marked simultaneous speech, which
			// is structurally distinct from a bare consecutive cue.
			container := innerContainer(currentTopLevel, currentAct, currentScene)
			recordBreak(container, v.Range.Start.Line)
			walkNode(v.Left, currentTopLevel, currentAct, currentScene)
			walkNode(v.Right, currentTopLevel, currentAct, currentScene)
			recordBreak(container, v.Range.End.Line)
		case *ast.StageDirection:
			recordBreak(innerContainer(currentTopLevel, currentAct, currentScene), v.Range.Start.Line)
		case *ast.Callout:
			recordBreak(innerContainer(currentTopLevel, currentAct, currentScene), v.Range.Start.Line)
		case *ast.PageBreak:
			recordBreak(innerContainer(currentTopLevel, currentAct, currentScene), v.Range.Start.Line)
		case *ast.VerseBlock:
			recordBreak(innerContainer(currentTopLevel, currentAct, currentScene), v.Range.Start.Line)
		case *ast.Section:
			if v.Level == 1 {
				currentTopLevel = v
				currentAct = nil
				currentScene = nil
				index.topLevelSections = append(index.topLevelSections, v)
			}
			if v.Kind == ast.SectionAct {
				index.acts = append(index.acts, v)
				actCountsByPlay[currentTopLevel]++
				index.actNumbers[v] = actCountsByPlay[currentTopLevel]
				index.actPlays[v] = currentTopLevel
				index.actsByPlay[currentTopLevel] = append(index.actsByPlay[currentTopLevel], v)
				currentAct = v
				currentScene = nil
			}
			if v.Kind == ast.SectionScene {
				index.scenes = append(index.scenes, v)
				index.sceneActs[v] = currentAct
				if currentAct != nil {
					sceneCountsByAct[currentAct]++
					index.sceneNumbers[v] = sceneCountsByAct[currentAct]
				} else {
					sceneCountsByPlay[currentTopLevel]++
					index.sceneNumbers[v] = sceneCountsByPlay[currentTopLevel]
				}
				currentScene = v
			}
			for _, child := range v.Children {
				walkNode(child, currentTopLevel, currentAct, currentScene)
			}
		case *ast.Song:
			// A Song is itself a structural container distinct from dialogue
			// prose, so treat the surrounding boundary as a break but still
			// descend to track cues inside.
			container := innerContainer(currentTopLevel, currentAct, currentScene)
			recordBreak(container, v.Range.Start.Line)
			for _, child := range v.Content {
				walkNode(child, currentTopLevel, currentAct, currentScene)
			}
			recordBreak(container, v.Range.End.Line)
		}
	}

	for _, node := range doc.Body {
		walkNode(node, nil, nil, nil)
	}

	sort.Slice(index.acts, func(i, j int) bool {
		return index.acts[i].Range.Start.Line < index.acts[j].Range.Start.Line
	})
	sort.Slice(index.scenes, func(i, j int) bool {
		return index.scenes[i].Range.Start.Line < index.scenes[j].Range.Start.Line
	})
	sort.Slice(index.topLevelSections, func(i, j int) bool {
		return index.topLevelSections[i].Range.Start.Line < index.topLevelSections[j].Range.Start.Line
	})
	for scene, cues := range index.sceneSpeakers {
		sort.Slice(cues, func(i, j int) bool {
			return cues[i].line < cues[j].line
		})
		index.sceneSpeakers[scene] = cues
	}

	return index
}

func (idx *documentIndex) sceneForLine(line int) *ast.Section {
	return nearestSectionBeforeLine(idx.scenes, line)
}

func (idx *documentIndex) actForLine(line int) *ast.Section {
	play := nearestSectionBeforeLine(idx.topLevelSections, line)
	if acts := idx.actsByPlay[play]; len(acts) > 0 {
		return nearestSectionBeforeLine(acts, line)
	}
	return nil
}

func (idx *documentIndex) isCharacterCueLine(line int) bool {
	_, ok := idx.characterCueLines[line]
	return ok
}

func (idx *documentIndex) characterScopeForSection(section *ast.Section) characterScope {
	if section != nil {
		if scope, ok := idx.characterScopes[section]; ok && scope.dp != nil {
			return scope
		}
	}
	return idx.legacyCharacterScope
}

func (idx *documentIndex) characterScopeForLine(doc *ast.Document, line int) characterScope {
	return idx.characterScopeForSection(topLevelSectionForLine(doc, line))
}

func (idx *documentIndex) sceneSpeakersBeforeLine(scene *ast.Section, line int) []string {
	cues := idx.sceneSpeakers[scene]
	if len(cues) == 0 {
		return nil
	}

	limit := sort.Search(len(cues), func(i int) bool {
		return cues[i].line >= line
	})
	speakers := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		speakers = append(speakers, cues[i].name)
	}
	return speakers
}

func nearestSectionBeforeLine(sections []*ast.Section, line int) *ast.Section {
	if len(sections) == 0 {
		return nil
	}

	idx := sort.Search(len(sections), func(i int) bool {
		return sections[i].Range.Start.Line > line
	})
	if idx == 0 {
		return nil
	}
	return sections[idx-1]
}
