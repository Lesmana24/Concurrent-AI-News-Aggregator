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
    <div className="space-y-8 sm:space-y-10">
      {/* Header */}
      <div className="border-b border-slate-800 pb-4 sm:pb-5">
        <span className="text-[11px] sm:text-xs font-mono text-amber-400 uppercase tracking-wider font-bold">
          RAG & pgvector Knowledge Engine
        </span>
        <h2 className="font-editorial text-xl sm:text-2xl lg:text-3xl font-bold text-slate-100 mt-1">
          Semantic Inquiry & Live SSE Q&A
        </h2>
        <p className="text-xs sm:text-sm text-slate-300 mt-1 font-sans leading-relaxed max-w-2xl">
          Pencarian makna semantik mendalam pada arsip berita dan interaksi tanya jawab langsung dengan efek pengetikan <i>real-time typewriter</i> via Server-Sent Events.
        </p>
      </div>

      {/* Grid: 2 Columns on Desktop */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column: Live Streaming Q&A */}
        <section className="p-4 sm:p-7 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-4 sm:space-y-5 flex flex-col justify-between shadow-xl">
          <div className="space-y-4">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <div className="flex items-center gap-2">
                <MessageSquare className="w-5 h-5 text-amber-500" />
                <h3 className="font-editorial text-base sm:text-lg font-bold text-slate-100">
                  Live Typewriter Q&A (SSE)
                </h3>
              </div>
              <span className="text-[10px] sm:text-xs font-mono font-semibold px-2.5 py-0.5 rounded-md bg-amber-500/15 text-amber-400 border border-amber-500/30">
                Server-Sent Events
              </span>
            </div>

            <form onSubmit={handleStreamQA} className="space-y-3">
              <textarea
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="Tanyakan hal apa saja mengenai berita terkini (contoh: Apa penyebab utama inflasi Oktober 2026?)..."
                rows={3}
                required
                className="w-full bg-slate-950 border border-slate-700/80 focus:border-amber-500 rounded-xl p-3.5 text-base sm:text-sm font-sans text-slate-100 placeholder:text-slate-500 focus:outline-none transition resize-none leading-relaxed"
              />
              <div className="flex justify-end">
                <button
                  type="submit"
                  disabled={streaming || !question.trim()}
                  className="w-full sm:w-auto px-5 py-3 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 shadow-lg min-h-[44px] touch-manipulation"
                >
                  {streaming ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span>Mengetik Jawaban...</span>
                    </>
                  ) : (
                    <>
                      <Send className="w-4 h-4" />
                      <span>Kirim Pertanyaan</span>
                    </>
                  )}
                </button>
              </div>
            </form>

            {qaError && (
              <div className="p-3.5 sm:p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-xs font-mono text-rose-300">
                {qaError}
              </div>
            )}

            {/* Answer Box */}
            {(streamedAnswer || streaming) && (
              <div className="p-4 sm:p-5 rounded-xl bg-slate-950 border border-slate-800 space-y-3 animate-in fade-in">
                <div className="flex items-center justify-between text-xs font-mono border-b border-slate-800 pb-2.5">
                  <span className="text-emerald-400 font-bold flex items-center gap-1.5">
                    <Sparkles className="w-4 h-4 text-emerald-400" /> Jawaban Sintesis AI:
                  </span>
                  {provider && (
                    <span className="text-slate-400 font-medium">Provider: {provider}</span>
                  )}
                </div>

                <div className="text-sm sm:text-base text-slate-200 leading-relaxed whitespace-pre-wrap font-sans">
                  {streamedAnswer}
                  {streaming && (
                    <span className="inline-block w-2 h-4 ml-1 bg-amber-400 animate-pulse align-middle" />
                  )}
                </div>

                {citations.length > 0 && (
                  <div className="pt-2.5 border-t border-slate-800 space-y-1.5">
                    <span className="text-xs font-mono font-semibold text-slate-400">Sumber Rujukan:</span>
                    <ul className="text-xs font-mono text-blue-400 space-y-1">
                      {citations.map((cite, i) => (
                        <li key={i} className="truncate hover:underline">
                          <a href={cite} target="_blank" rel="noopener noreferrer" className="flex items-center gap-1">
                            <span>{i + 1}. {cite}</span>
                            <ExternalLink className="w-3 h-3 shrink-0" />
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
        <section className="p-4 sm:p-7 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-4 sm:space-y-5 shadow-xl">
          <div className="flex items-center justify-between border-b border-slate-800 pb-3">
            <div className="flex items-center gap-2">
              <Search className="w-5 h-5 text-amber-500" />
              <h3 className="font-editorial text-base sm:text-lg font-bold text-slate-100">
                pgvector Semantic Search
              </h3>
            </div>
            <span className="text-[10px] sm:text-xs font-mono px-2.5 py-0.5 rounded-md bg-slate-800 text-blue-300 border border-slate-700">
              Cosine Distance
            </span>
          </div>

          <form onSubmit={handleSearch} className="flex flex-col sm:flex-row gap-2.5">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Cari berdasarkan konsep makna berita..."
              required
              className="flex-1 bg-slate-950 border border-slate-700/80 focus:border-amber-500 rounded-xl px-4 py-3 text-base sm:text-sm font-mono text-slate-100 placeholder:text-slate-500 focus:outline-none transition min-h-[44px]"
            />
            <button
              type="submit"
              disabled={searching || !searchQuery.trim()}
              className="w-full sm:w-auto px-5 py-3 rounded-xl bg-slate-800 hover:bg-slate-700 active:bg-slate-750 text-slate-100 text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 border border-slate-700 min-h-[44px] touch-manipulation"
            >
              {searching ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  <span>Mencari...</span>
                </>
              ) : (
                <>
                  <Search className="w-4 h-4" />
                  <span>Cari Makna</span>
                </>
              )}
            </button>
          </form>

          {searchError && (
            <div className="p-3.5 sm:p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-xs font-mono text-rose-300">
              {searchError}
            </div>
          )}

          {/* Results List */}
          <div className="space-y-3 pt-1">
            {searchResults.length > 0 ? (
              searchResults.map((res) => (
                <div
                  key={res.id}
                  className="p-4 rounded-xl bg-slate-950 border border-slate-800 hover:border-slate-700 transition space-y-1.5"
                >
                  <div className="flex items-center justify-between text-xs font-mono">
                    <span className="px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700 text-[11px]">
                      {res.source || 'Media Nasional'}
                    </span>
                    {res.similarity_score !== undefined && (
                      <span className="text-amber-400 font-bold">
                        Skor: {(res.similarity_score * 100).toFixed(1)}%
                      </span>
                    )}
                  </div>
                  <h4 className="font-editorial text-sm sm:text-base font-bold text-slate-100 leading-snug">
                    <a
                      href={res.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:text-amber-400 transition flex items-center gap-1.5"
                    >
                      <span>{res.title}</span>
                      <ExternalLink className="w-3.5 h-3.5 text-slate-500 shrink-0" />
                    </a>
                  </h4>
                  <p className="text-xs sm:text-sm text-slate-300 leading-relaxed font-sans line-clamp-2">
                    {res.summary}
                  </p>
                </div>
              ))
            ) : (
              !searching && (
                <div className="p-6 text-center text-xs sm:text-sm font-mono text-slate-400 bg-slate-950/40 rounded-xl border border-slate-800">
                  Ketikkan kata kunci atau pertanyaan semantik untuk menjelajahi artikel terkait.
                </div>
              )
            )}
          </div>
        </section>
      </div>
    </div>
  )
}
