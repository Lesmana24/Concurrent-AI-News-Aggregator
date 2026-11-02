import React, { useState } from 'react'
import { MessageSquare, Search, Send, Sparkles, RefreshCw, ExternalLink } from 'lucide-react'
import type { SemanticSearchResult } from '../types'
import { api } from '../services/api'

interface SemanticInquiryProps {
  apiKey: string
}

export const SemanticInquiry: React.FC<SemanticInquiryProps> = ({ apiKey }) => {
  // Search state
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<SemanticSearchResult[]>([])
  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState<string | null>(null)

  // SSE Q&A state
  const [question, setQuestion] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [streamedAnswer, setStreamedAnswer] = useState('')
  const [provider, setProvider] = useState('')
  const [citations, setCitations] = useState<string[]>([])
  const [qaError, setQaError] = useState<string | null>(null)

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!searchQuery.trim()) return

    setSearching(true)
    setSearchError(null)

    try {
      const data = await api.semanticSearch(searchQuery.trim(), 5)
      setSearchResults(data.results || [])
    } catch (err: unknown) {
      setSearchError(err instanceof Error ? err.message : 'Pencarian semantik gagal')
    } finally {
      setSearching(false)
    }
  }

  const handleStreamQA = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!question.trim()) return

    setStreaming(true)
    setStreamedAnswer('')
    setProvider('')
    setCitations([])
    setQaError(null)

    await api.streamQA(
      apiKey,
      question.trim(),
      (token) => {
        setStreamedAnswer((prev) => prev + token)
      },
      (doneData) => {
        setStreaming(false)
        if (doneData.answer) setStreamedAnswer(doneData.answer)
        if (doneData.provider) setProvider(doneData.provider)
        if (doneData.citations) setCitations(doneData.citations)
      },
      (err) => {
        setStreaming(false)
        setQaError(err.message)
      }
    )
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-[#1F2937] pb-4">
        <span className="text-[11px] font-mono text-[#D97706] uppercase tracking-wider font-semibold">
          RAG & pgvector Knowledge Engine
        </span>
        <h2 className="font-editorial text-xl lg:text-2xl font-bold text-[#F3F4F6] mt-0.5">
          Semantic Inquiry & Live SSE Q&A
        </h2>
        <p className="text-xs text-[#9CA3AF] mt-1">
          Pencarian makna semantik mendalam pada arsip berita dan interaksi tanya jawab langsung dengan efek pengetikan <i>real-time typewriter</i>.
        </p>
      </div>

      {/* Grid: 2 Columns on Desktop */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column: Live Streaming Q&A */}
        <section className="p-6 rounded-lg bg-[#111827] border border-[#1F2937] space-y-4 flex flex-col justify-between">
          <div className="space-y-4">
            <div className="flex items-center justify-between border-b border-[#1F2937] pb-3">
              <div className="flex items-center gap-2">
                <MessageSquare className="w-4 h-4 text-[#D97706]" />
                <h3 className="font-editorial text-base font-bold text-[#F3F4F6]">
                  Live Typewriter Q&A (SSE)
                </h3>
              </div>
              <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-[#D97706]/15 text-[#F59E0B] border border-[#D97706]/30">
                Server-Sent Events
              </span>
            </div>

            <form onSubmit={handleStreamQA} className="space-y-2">
              <textarea
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="Tanyakan hal apa saja mengenai isu terkini (contoh: Apa penyebab utama inflasi Oktober 2026?)..."
                rows={3}
                required
                className="w-full bg-[#080C14] border border-[#1F2937] focus:border-[#D97706] rounded-md p-3 text-xs font-sans text-[#F3F4F6] placeholder:text-[#4B5563] focus:outline-none resize-none"
              />
              <div className="flex justify-end">
                <button
                  type="submit"
                  disabled={streaming || !question.trim()}
                  className="px-4 py-2 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition disabled:opacity-50 flex items-center gap-1.5"
                >
                  {streaming ? (
                    <>
                      <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                      <span>Mengetik Jawaban...</span>
                    </>
                  ) : (
                    <>
                      <Send className="w-3.5 h-3.5" />
                      <span>Kirim Pertanyaan</span>
                    </>
                  )}
                </button>
              </div>
            </form>

            {qaError && (
              <div className="p-3 rounded-md bg-[#E11D48]/10 border border-[#E11D48]/30 text-xs font-mono text-[#FCA5A5]">
                {qaError}
              </div>
            )}

            {/* Answer Box */}
            {(streamedAnswer || streaming) && (
              <div className="p-4 rounded-md bg-[#080C14] border border-[#1F2937] space-y-3">
                <div className="flex items-center justify-between text-xs font-mono border-b border-[#1F2937] pb-2">
                  <span className="text-[#10B981] font-semibold flex items-center gap-1.5">
                    <Sparkles className="w-3.5 h-3.5" /> Jawaban Sintesis AI:
                  </span>
                  {provider && (
                    <span className="text-[#9CA3AF]">Provider: {provider}</span>
                  )}
                </div>

                <div className="text-xs text-[#D1D5DB] leading-relaxed whitespace-pre-wrap font-sans">
                  {streamedAnswer}
                  {streaming && (
                    <span className="inline-block w-2 h-4 ml-1 bg-[#D97706] animate-pulse align-middle" />
                  )}
                </div>

                {citations.length > 0 && (
                  <div className="pt-2 border-t border-[#1F2937] space-y-1">
                    <span className="text-[11px] font-mono text-[#9CA3AF]">Sumber Rujukan:</span>
                    <ul className="text-xs font-mono text-[#60A5FA] space-y-1">
                      {citations.map((cite, i) => (
                        <li key={i} className="truncate hover:underline">
                          <a href={cite} target="_blank" rel="noopener noreferrer">
                            {i + 1}. {cite}
                          </a>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            )}
          </div>
        </section>

        {/* Right Column: pgvector Semantic Search */}
        <section className="p-6 rounded-lg bg-[#111827] border border-[#1F2937] space-y-4">
          <div className="flex items-center justify-between border-b border-[#1F2937] pb-3">
            <div className="flex items-center gap-2">
              <Search className="w-4 h-4 text-[#D97706]" />
              <h3 className="font-editorial text-base font-bold text-[#F3F4F6]">
                pgvector Semantic Search
              </h3>
            </div>
            <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-[#182032] text-[#93C5FD] border border-[#1E293B]">
              Cosine Distance
            </span>
          </div>

          <form onSubmit={handleSearch} className="flex gap-2">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Cari berdasarkan konsep makna berita..."
              required
              className="flex-1 bg-[#080C14] border border-[#1F2937] focus:border-[#D97706] rounded-md px-3.5 py-2 text-xs font-mono text-[#F3F4F6] placeholder:text-[#4B5563] focus:outline-none"
            />
            <button
              type="submit"
              disabled={searching || !searchQuery.trim()}
              className="px-4 py-2 rounded-md bg-[#1F2937] hover:bg-[#374151] text-[#F3F4F6] text-xs font-mono transition disabled:opacity-50"
            >
              {searching ? 'Mencari...' : 'Cari'}
            </button>
          </form>

          {searchError && (
            <div className="p-3 rounded-md bg-[#E11D48]/10 border border-[#E11D48]/30 text-xs font-mono text-[#FCA5A5]">
              {searchError}
            </div>
          )}

          {/* Search Results */}
          <div className="space-y-3 pt-2">
            {searching && (
              <div className="space-y-2">
                {[1, 2, 3].map((n) => (
                  <div key={n} className="p-3 rounded-md bg-[#080C14] border border-[#1F2937] space-y-1.5 animate-pulse">
                    <div className="h-4 w-1/2 bg-[#1F2937] rounded" />
                    <div className="h-3 w-3/4 bg-[#182032] rounded" />
                  </div>
                ))}
              </div>
            )}

            {!searching && searchResults.length === 0 && searchQuery && (
              <p className="text-xs font-mono text-[#9CA3AF] text-center py-6">
                Tidak ada artikel yang cocok dengan query semantik ini.
              </p>
            )}

            {!searching && searchResults.length > 0 && (
              <div className="space-y-2.5 max-h-[420px] overflow-y-auto pr-1">
                {searchResults.map((item) => (
                  <div
                    key={item.id}
                    className="p-3.5 rounded-md bg-[#080C14] hover:bg-[#151D2E] border border-[#1F2937] space-y-1.5 transition"
                  >
                    <div className="flex items-center justify-between gap-2 text-[11px] font-mono">
                      <span className="px-1.5 py-0.5 rounded bg-[#182032] text-[#93C5FD]">
                        {item.source || 'Media Nasional'}
                      </span>
                      {item.similarity_score !== undefined && (
                        <span className="text-[#10B981] font-semibold">
                          Kesesuaian: {(item.similarity_score * 100).toFixed(1)}%
                        </span>
                      )}
                    </div>

                    <a
                      href={item.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-editorial text-sm font-semibold text-[#F3F4F6] hover:text-[#F59E0B] leading-snug flex items-center gap-1.5"
                    >
                      <span>{item.title}</span>
                      <ExternalLink className="w-3 h-3 text-[#6B7280] shrink-0" />
                    </a>

                    <p className="text-xs text-[#9CA3AF] line-clamp-2 leading-relaxed">
                      {item.summary}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  )
}
