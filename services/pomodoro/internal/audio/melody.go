package audio

import "github.com/InsideGallery/pomodoro/services/pomodoro/internal/timer"

// Melody is a selectable looping sound: the embedded file name and its UI label.
type Melody struct {
	File, Label string
}

// FocusLoop is the file looped during focus.
const FocusLoop = "tick.mp3"

// Melodies lists the selectable melodies in UI order.
var Melodies = []Melody{
	{"tick.mp3", "Tick"},
	{"pause.mp3", "Pause"},
	{"guitar.mp3", "Guitar"},
	{"piano.mp3", "Piano"},
}

// MelodyFor returns the melody with the given file, or Tick when unknown.
func MelodyFor(file string) Melody {
	for _, m := range Melodies {
		if m.File == file {
			return m
		}
	}

	return Melodies[0]
}

// NextMelody steps from file by step (+1 or -1), wrapping around the list.
func NextMelody(file string, step int) Melody {
	idx := 0

	for i, m := range Melodies {
		if m.File == MelodyFor(file).File {
			idx = i

			break
		}
	}

	n := len(Melodies)

	return Melodies[((idx+step)%n+n)%n]
}

// loopFor returns the file to loop for the state, or "" for silence.
func loopFor(st timer.State, muted, loopEnabled bool, breakMelody string) string {
	if muted || !loopEnabled {
		return ""
	}

	switch st {
	case timer.StateFocus:
		return FocusLoop
	case timer.StateBreak, timer.StateLongBreak:
		return MelodyFor(breakMelody).File
	case timer.StateIdle, timer.StatePaused:
		return ""
	}

	return ""
}
