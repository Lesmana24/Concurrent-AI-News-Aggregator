package strutil

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// TruncateRunes truncates string s to at most maxRunes Unicode code points (runes)
// without allocating a slice or breaking multi-byte UTF-8 character sequences.
// If utf8.RuneCountInString(s) <= maxRunes or maxRunes <= 0, appropriate strings are returned.
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == maxRunes {
			return s[:i]
		}
		count++
	}
	return s
}

// TruncateRunesWithEllipsis truncates string s so that the resulting string,
// including the ellipsis suffix, contains at most maxRunes code points.
// If s already fits within maxRunes, it is returned intact.
func TruncateRunesWithEllipsis(s string, maxRunes int, ellipsis string) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	ellipsisRunes := utf8.RuneCountInString(ellipsis)
	if maxRunes <= ellipsisRunes {
		return TruncateRunes(ellipsis, maxRunes)
	}
	targetRunes := maxRunes - ellipsisRunes
	return TruncateRunes(s, targetRunes) + ellipsis
}

// TruncateWords truncates clean text to at most maxRunes. If the truncated text
// exceeds minWordBreakRunes and contains spaces, it cleanly snaps back to the
// last space boundary so words are not cut in half, appending the ellipsis.
func TruncateWords(text string, maxRunes int, minWordBreakRunes int, ellipsis string) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if utf8.RuneCountInString(clean) <= maxRunes {
		return clean
	}

	truncated := TruncateRunes(clean, maxRunes)
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 && utf8.RuneCountInString(truncated[:lastSpace]) >= minWordBreakRunes {
		truncated = truncated[:lastSpace]
	}
	return truncated + ellipsis
}

// CapitalizeFirst capitalizes the first Unicode character of a string in a rune-safe manner.
func CapitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

