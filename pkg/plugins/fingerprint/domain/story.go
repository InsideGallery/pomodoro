package domain

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"github.com/InsideGallery/pomodoro/pkg/platform"
)

// StoriesFile is the top-level JSON structure for stories.json.
type StoriesFile struct {
	City  string      `json:"city"`
	Cases []CaseStory `json:"cases"`
}

// CaseStory holds the narrative for a case location.
type CaseStory struct {
	Name     string   `json:"name"`
	Intro    string   `json:"intro"`
	Evidence []string `json:"evidence"` // 20 evidence locations per case
	Solved   []string `json:"solved"`   // solved description templates with {name}
}

// Stories holds all loaded stories. Populated by LoadStories.
var Stories *StoriesFile //nolint:gochecknoglobals // loaded once

// LoadStories reads stories.json via platform asset loader.
func LoadStories(_ string) error {
	data, err := platform.ReadAsset("external/fingerprint/stories.json")
	if err != nil {
		return fmt.Errorf("read stories: %w", err)
	}

	var sf StoriesFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("parse stories: %w", err)
	}

	Stories = &sf

	return nil
}

// GetCaseStory returns the story for a case index.
func GetCaseStory(caseIdx int) *CaseStory {
	if Stories == nil || caseIdx < 0 || caseIdx >= len(Stories.Cases) {
		return &CaseStory{Name: "UNKNOWN", Intro: "A crime was committed.", Evidence: []string{"at the scene"}, Solved: []string{"{name} was involved."}}
	}

	return &Stories.Cases[caseIdx]
}

// EvidenceForPuzzle returns the evidence location for a specific puzzle within a case.
func EvidenceForPuzzle(caseIdx, puzzleIdx int) string {
	story := GetCaseStory(caseIdx)
	if len(story.Evidence) == 0 {
		return "at the scene"
	}

	return story.Evidence[puzzleIdx%len(story.Evidence)]
}

// UnsolvedDescription returns the text shown before a puzzle is solved.
func UnsolvedDescription(caseIdx, puzzleIdx int) string {
	story := GetCaseStory(caseIdx)
	evidence := EvidenceForPuzzle(caseIdx, puzzleIdx)

	return fmt.Sprintf("%s\n\nFingerprint found %s.\nAnalysis required.", story.Intro, evidence)
}

// SolvedDescription returns the text shown after identifying the fingerprint owner.
func SolvedDescription(caseIdx, puzzleIdx int, personName string) string {
	story := GetCaseStory(caseIdx)
	evidence := EvidenceForPuzzle(caseIdx, puzzleIdx)

	// Pick a solved template
	if len(story.Solved) > 0 {
		rng := rand.New(rand.NewPCG(uint64(caseIdx*100+puzzleIdx), 0)) //nolint:gosec // deterministic
		tmpl := story.Solved[rng.IntN(len(story.Solved))]

		// Replace {name} placeholder
		result := ""
		for i := 0; i < len(tmpl); {
			if i+6 <= len(tmpl) && tmpl[i:i+6] == "{name}" {
				result += personName
				i += 6
			} else {
				result += string(tmpl[i])
				i++
			}
		}

		return fmt.Sprintf("Fingerprint found %s.\n\n%s", evidence, result)
	}

	return fmt.Sprintf("Fingerprint found %s.\n\n%s was identified.", evidence, personName)
}

// NoMatchDescription returns the text shown when fingerprint didn't match anyone.
func NoMatchDescription(caseIdx, puzzleIdx int) string {
	evidence := EvidenceForPuzzle(caseIdx, puzzleIdx)

	return fmt.Sprintf("Fingerprint found %s\ndoes not match any records.\nCase remains open.", evidence)
}

// CaseCount returns the number of available cases from stories.
func CaseCount() int {
	if Stories == nil {
		return 10
	}

	return len(Stories.Cases)
}

// CaseNameFromStory returns the case name from stories.json.
func CaseNameFromStory(caseIdx int) string {
	story := GetCaseStory(caseIdx)

	return story.Name
}

// Character ties a person name to their avatar file.
type Character struct {
	Name   string
	Avatar string
}

// Characters are the game's suspects — each has a unique avatar.
var Characters = []Character{ //nolint:gochecknoglobals // constant
	{Name: "Rob Malfoy", Avatar: "m.{Rob Malfoy}.jpg"},
	{Name: "Steve Gilber", Avatar: "m.{Steve Gilber}.jpg"},
	{Name: "Elizabet Queen", Avatar: "w.{Elizabet Queen}.jpg"},
	{Name: "May Forty", Avatar: "w.{May Forty}.jpg"},
}

// UnknownAvatar is the filename for unsolved puzzles.
const UnknownAvatar = "unkown.jpg"

// CharacterForRecord returns the character assigned to a fingerprint record.
func CharacterForRecord(rec *FingerprintRecord) Character {
	idx := (rec.ID - 1) % len(Characters)

	return Characters[idx]
}

// AvatarForRecord returns the avatar filename for a record.
func AvatarForRecord(rec *FingerprintRecord) string {
	return CharacterForRecord(rec).Avatar
}
