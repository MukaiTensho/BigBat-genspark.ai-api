package openai

import "unicode/utf8"

func ApproxTokenCount(text string) int {
	if text == "" {
		return 0
	}
	runes := utf8.RuneCountInString(text)
	if runes <= 0 {
		return 0
	}
	return (runes / 4) + 1
}
