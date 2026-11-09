package telegram

import (
	"news-aggregator/internal/strutil"
	"regexp"
	"strings"
	"unicode/utf8"
)

// DefaultMaxTelegramMessageRunes is the safe character limit for Telegram messages.
// The hard limit in Telegram Bot API is 4096 runes. 4000 leaves a buffer for closing/reopening HTML tags.
const DefaultMaxTelegramMessageRunes = 4000

var tgHTMLTagRegex = regexp.MustCompile(`(?i)<(/)?([a-z0-9_-]+)(?:\s+[^>]*)?>`)

type openTagInfo struct {
	name     string
	fullOpen string
}

// SplitTelegramMessage chunks a text string into slices of at most maxRunes,
// preserving paragraph boundaries, line boundaries, word boundaries, and HTML tag integrity.
func SplitTelegramMessage(text string, maxRunes int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxRunes <= 0 {
		maxRunes = DefaultMaxTelegramMessageRunes
	}

	if utf8.RuneCountInString(text) <= maxRunes {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if utf8.RuneCountInString(remaining) <= maxRunes {
			trimmed := strings.TrimSpace(remaining)
			if trimmed != "" {
				chunks = append(chunks, trimmed)
			}
			break
		}

		// Candidate substring of up to maxRunes
		candidate := strutil.TruncateRunes(remaining, maxRunes)

		// Prevent splitting in the middle of an HTML tag: <tag ...>
		lastOpen := strings.LastIndex(candidate, "<")
		lastClose := strings.LastIndex(candidate, ">")
		if lastOpen > lastClose && lastOpen > 0 {
			candidate = candidate[:lastOpen]
		}

		// Find optimal split point (prefer \n\n, then \n, then ". ", then " ")
		splitIdx := findOptimalSplitIndex(candidate, maxRunes)
		if splitIdx <= 0 {
			splitIdx = len(candidate)
		}

		chunkText := candidate[:splitIdx]
		nextRemaining := strings.TrimLeft(remaining[splitIdx:], "\r\n")

		// Analyze open HTML tags in this chunk and re-balance if needed
		openTags := findUnclosedTags(chunkText)
		if len(openTags) > 0 {
			// Close open tags at end of current chunk
			var closer strings.Builder
			for i := len(openTags) - 1; i >= 0; i-- {
				closer.WriteString("</")
				closer.WriteString(openTags[i].name)
				closer.WriteString(">")
			}
			chunkText += closer.String()

			// Re-open tags at start of next remaining chunk
			var opener strings.Builder
			for _, t := range openTags {
				opener.WriteString(t.fullOpen)
			}
			nextRemaining = opener.String() + nextRemaining
		}

		trimmedChunk := strings.TrimSpace(chunkText)
		if trimmedChunk != "" {
			chunks = append(chunks, trimmedChunk)
		}

		// Safety check against infinite loops
		if len(nextRemaining) >= len(remaining) {
			_, size := utf8.DecodeRuneInString(remaining)
			if size <= 0 {
				break
			}
			remaining = remaining[size:]
		} else {
			remaining = nextRemaining
		}
	}

	return chunks
}

func findOptimalSplitIndex(candidate string, maxRunes int) int {
	minAcceptableRunes := maxRunes / 3 // Don't split too early if not necessary

	// 1. Paragraph boundary: \n\n
	if idx := strings.LastIndex(candidate, "\n\n"); idx > 0 {
		if utf8.RuneCountInString(candidate[:idx]) >= minAcceptableRunes {
			return idx + 2
		}
	}

	// 2. Line boundary: \n
	if idx := strings.LastIndex(candidate, "\n"); idx > 0 {
		if utf8.RuneCountInString(candidate[:idx]) >= minAcceptableRunes {
			return idx + 1
		}
	}

	// 3. Sentence boundary: ". "
	if idx := strings.LastIndex(candidate, ". "); idx > 0 {
		if utf8.RuneCountInString(candidate[:idx]) >= minAcceptableRunes {
			return idx + 2
		}
	}

	// 4. Word boundary: " "
	if idx := strings.LastIndex(candidate, " "); idx > 0 {
		if utf8.RuneCountInString(candidate[:idx]) >= minAcceptableRunes {
			return idx + 1
		}
	}

	// 5. Hard break
	return len(candidate)
}

func findUnclosedTags(s string) []openTagInfo {
	matches := tgHTMLTagRegex.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}

	var stack []openTagInfo
	for _, m := range matches {
		fullMatch := m[0]
		isClosing := m[1] == "/"
		tagName := strings.ToLower(m[2])

		// Ignore self-closing tags like <br/> or <img ... />
		if strings.HasSuffix(fullMatch, "/>") {
			continue
		}

		if isClosing {
			// Pop the matching tag from stack
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].name == tagName {
					stack = append(stack[:i], stack[i+1:]...)
					break
				}
			}
		} else {
			// Push opening tag
			stack = append(stack, openTagInfo{
				name:     tagName,
				fullOpen: fullMatch,
			})
		}
	}

	return stack
}

