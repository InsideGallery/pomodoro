package fsystems

import (
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/InsideGallery/pomodoro/pkg/ui"
)

func wrapText(text string, face *textv2.GoTextFace, maxW float64) []string {
	var lines []string

	for _, paragraph := range splitNewlines(text) {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}

		words := splitWords(paragraph)
		current := ""

		for _, word := range words {
			test := current
			if test != "" {
				test += " "
			}

			test += word

			w, _ := ui.MeasureText(test, face)
			if w > maxW && current != "" {
				lines = append(lines, current)
				current = word
			} else {
				current = test
			}
		}

		if current != "" {
			lines = append(lines, current)
		}
	}

	return lines
}

func splitNewlines(s string) []string {
	var lines []string

	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}

	lines = append(lines, s[start:])

	return lines
}

func splitWords(s string) []string {
	var words []string

	start := -1
	for i, ch := range s {
		if ch == ' ' {
			if start >= 0 {
				words = append(words, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}

	if start >= 0 {
		words = append(words, s[start:])
	}

	return words
}
