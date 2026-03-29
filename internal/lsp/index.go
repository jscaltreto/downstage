package lsp

import (
	"sort"
	"strings"

	"github.com/jscaltreto/downstage/internal/ast"
)

type dialogueRef struct {
	dialogue *ast.Dialogue
	scene    *ast.Section
}

type sceneSpeakerCue struct {
	line int
	name string
}

type documentIndex struct {
	acts                   []*ast.Section
	scenes                 []*ast.Section
	sceneActs              map[*ast.Section]*ast.Section
	sceneNumbers           map[*ast.Section]int
	characterCueLines      map[int]struct{}
	documentCharacterNames []string
	knownCharacters        map[string]struct{}
	dialogues              []dialogueRef
	sceneSpeakers          map[*ast.Section][]sceneSpeakerCue
	hasDramatisPersonae    bool
}

func newDocumentIndex(doc *ast.Document) *documentIndex {
	index := &documentIndex{
		characterCueLines: make(map[int]struct{}),
		knownCharacters:   make(map[string]struct{}),
		sceneSpeakers:     make(map[*ast.Section][]sceneSpeakerCue),
		sceneActs:         make(map[*ast.Section]*ast.Section),
		sceneNumbers:      make(map[*ast.Section]int),
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

	if dp := ast.FindDramatisPersonae(doc.Body); dp != nil {
		index.hasDramatisPersonae = true
		for _, ch := range dp.AllCharacters() {
			addDocumentCharacter(ch.Name)
			index.knownCharacters[strings.ToUpper(ch.Name)] = struct{}{}
			for _, alias := range ch.Aliases {
				alias = strings.TrimSpace(alias)
				if alias == "" {
					continue
				}
				index.knownCharacters[strings.ToUpper(alias)] = struct{}{}
			}
		}
	}

	sceneCountsByAct := make(map[*ast.Section]int)
	sceneCountOutsideActs := 0

	var walkNode func(ast.Node, *ast.Section, *ast.Section)
	walkNode = func(node ast.Node, currentAct *ast.Section, currentScene *ast.Section) {
		switch v := node.(type) {
		case *ast.Dialogue:
			index.characterCueLines[v.NameRange().Start.Line] = struct{}{}
			ref := dialogueRef{dialogue: v, scene: currentScene}
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
		case *ast.DualDialogue:
			walkNode(v.Left, currentAct, currentScene)
			walkNode(v.Right, currentAct, currentScene)
		case *ast.Section:
			if v.Kind == ast.SectionAct {
				index.acts = append(index.acts, v)
				currentAct = v
			}
			if v.Kind == ast.SectionScene {
				index.scenes = append(index.scenes, v)
				index.sceneActs[v] = currentAct
				if currentAct != nil {
					sceneCountsByAct[currentAct]++
					index.sceneNumbers[v] = sceneCountsByAct[currentAct]
				} else {
					sceneCountOutsideActs++
					index.sceneNumbers[v] = sceneCountOutsideActs
				}
				currentScene = v
			}
			for _, child := range v.Children {
				walkNode(child, currentAct, currentScene)
			}
		case *ast.Song:
			for _, child := range v.Content {
				walkNode(child, currentAct, currentScene)
			}
		}
	}

	for _, node := range doc.Body {
		walkNode(node, nil, nil)
	}

	sort.Slice(index.acts, func(i, j int) bool {
		return index.acts[i].Range.Start.Line < index.acts[j].Range.Start.Line
	})
	sort.Slice(index.scenes, func(i, j int) bool {
		return index.scenes[i].Range.Start.Line < index.scenes[j].Range.Start.Line
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
	return nearestSectionBeforeLine(idx.acts, line)
}

func (idx *documentIndex) isCharacterCueLine(line int) bool {
	_, ok := idx.characterCueLines[line]
	return ok
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
