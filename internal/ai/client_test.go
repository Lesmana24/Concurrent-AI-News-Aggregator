package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanSummary_StripsThinkTagsAndXML(t *testing.T) {
	raw := `<think>
Model reasoning process:
1. Analisis fakta utama
2. Identifikasi 3 kalimat
</think>
<summary>
Pemerintah meresmikan jalan tol baru hari ini. Proyek ini memangkas waktu tempuh antar kota secara signifikan. Masyarakat menyambut baik pengoperasian infrastruktur tersebut.
</summary>`

	cleaned := cleanSummary(raw)
	assert.NotContains(t, cleaned, "<think>")
	assert.NotContains(t, cleaned, "</think>")
	assert.NotContains(t, cleaned, "Model reasoning process")
	assert.NotContains(t, cleaned, "<summary>")
	assert.NotContains(t, cleaned, "</summary>")
	assert.Contains(t, cleaned, "Pemerintah meresmikan jalan tol baru")
}

func TestCleanSummary_HandlesPrefixedSentences(t *testing.T) {
	raw := `Sentence 1: Pertumbuhan ekonomi nasional mencapai target tahun ini.
Sentence 2: Sektor manufaktur dan konsumsi domestik menjadi penopang utama.
Sentence 3: Bank sentral optimistis stabilitas moneter akan tetap terjaga.`

	cleaned := cleanSummary(raw)
	assert.NotContains(t, cleaned, "Sentence 1:")
	assert.NotContains(t, cleaned, "Sentence 2:")
	assert.NotContains(t, cleaned, "Sentence 3:")
	assert.Equal(t, "Pertumbuhan ekonomi nasional mencapai target tahun ini. Sektor manufaktur dan konsumsi domestik menjadi penopang utama. Bank sentral optimistis stabilitas moneter akan tetap terjaga.", cleaned)
}

func BenchmarkCleanSummary(b *testing.B) {
	raw := `<think>Internal thoughts</think><summary>Kalimat 1: Berita satu. Kalimat 2: Berita dua. Kalimat 3: Berita tiga.</summary>`
	b.ReportAllocs()
	for b.Loop() {
		_ = cleanSummary(raw)
	}
}

