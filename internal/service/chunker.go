package service

import (
	"news-aggregator/internal/strutil"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var indonesianStopwords = map[string]bool{
	"ada": true, "adalah": true, "akan": true, "apa": true, "apakah": true,
	"atau": true, "bagaimana": true, "bisa": true, "dalam": true, "dan": true,
	"dapat": true, "dari": true, "di": true, "dia": true, "dimana": true,
	"ini": true, "itu": true, "jika": true, "juga": true, "kapan": true,
	"ke": true, "karena": true, "oleh": true, "pada": true, "saya": true,
	"sebagai": true, "secara": true, "sedang": true, "seperti": true, "siapa": true,
	"sudah": true, "telah": true, "tentang": true, "tersebut": true, "tidak": true,
	"untuk": true, "yang": true,
}

var wordRegexp = regexp.MustCompile(`[\p{L}\p{N}]+`)

type ScoredChunk struct {
	Index int
	Text  string
	Score int
}

// SmartSelectChunks splits long content into paragraph chunks, scores each against question keywords,
// and selects the most relevant chunks up to maxChars budget.
func SmartSelectChunks(content string, question string, maxChars int) string {
	content = strings.TrimSpace(content)
	if utf8.RuneCountInString(content) <= maxChars {
		return content
	}

	chunks := splitIntoParagraphChunks(content, 600)
	if len(chunks) <= 1 {
		if utf8.RuneCountInString(content) > maxChars {
			return strutil.TruncateRunes(content, maxChars) + "..."
		}
		return content
	}

	keywords := extractKeywords(question)
	if len(keywords) == 0 {
		// Fallback to leading paragraphs if no distinctive keywords found
		return selectLeadingChunks(chunks, maxChars)
	}

	var scored []ScoredChunk
	for idx, chunk := range chunks {
		score := scoreChunk(chunk, keywords)
		scored = append(scored, ScoredChunk{
			Index: idx,
			Text:  chunk,
			Score: score,
		})
	}

	// Always give a baseline score bonus to chunk 0 (article lead/summary)
	scored[0].Score += 1

	// Sort primarily by Score desc, secondarily by Index asc
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].Index < scored[j].Index
	})

	// Select top chunks within budget
	var selected []ScoredChunk
	totalChars := 0
	for _, sc := range scored {
		chunkLen := utf8.RuneCountInString(sc.Text)
		if totalChars+chunkLen > maxChars && len(selected) > 0 {
			continue
		}
		selected = append(selected, sc)
		totalChars += chunkLen
		if totalChars >= maxChars {
			break
		}
	}

	// Re-order selected chunks back to their original narrative chronological order
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Index < selected[j].Index
	})

	var result strings.Builder
	for i, sc := range selected {
		if i > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(sc.Text)
	}

	return result.String()
}

func splitIntoParagraphChunks(text string, targetChunkLen int) []string {
	if targetChunkLen <= 0 {
		targetChunkLen = 300
	}
	paragraphs := strings.Split(text, "\n")
	var chunks []string

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if len(p) > 0 {
			chunks = append(chunks, p)
		}
	}

	// If only 1 giant paragraph exists, split by sentence
	if len(chunks) <= 1 && utf8.RuneCountInString(text) > targetChunkLen {
		sentences := strings.Split(text, ". ")
		chunks = nil
		var current strings.Builder
		currentRunes := 0
		for _, s := range sentences {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			sRunes := utf8.RuneCountInString(s)
			if currentRunes+sRunes > targetChunkLen && current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
				currentRunes = 0
			}
			if current.Len() > 0 {
				current.WriteString(". ")
				currentRunes += 2
			}
			current.WriteString(s)
			currentRunes += sRunes
		}
		if current.Len() > 0 {
			chunks = append(chunks, current.String())
		}
	}

	return chunks
}

func extractKeywords(text string) []string {
	rawWords := wordRegexp.FindAllString(strings.ToLower(text), -1)
	var keywords []string
	for _, w := range rawWords {
		if len(w) >= 3 && !indonesianStopwords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}

func scoreChunk(chunk string, keywords []string) int {
	lowerChunk := strings.ToLower(chunk)
	score := 0
	for _, kw := range keywords {
		// Count occurrences
		count := strings.Count(lowerChunk, kw)
		score += count * 2

		// Prefix or substring match
		if count == 0 && strings.Contains(lowerChunk, kw) {
			score += 1
		}
	}
	return score
}

func selectLeadingChunks(chunks []string, maxChars int) string {
	var result strings.Builder
	total := 0
	for i, c := range chunks {
		cRunes := utf8.RuneCountInString(c)
		if total+cRunes > maxChars && i > 0 {
			break
		}
		if i > 0 {
			result.WriteString("\n\n")
		}
		result.WriteString(c)
		total += cRunes
	}
	return result.String()
}
