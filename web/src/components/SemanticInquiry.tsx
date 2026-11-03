import React, { useState, useRef, useEffect } from 'react'
import { MessageSquare, Search, Send, Sparkles, RefreshCw, ExternalLink, Newspaper, AlertCircle } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import type { SemanticSearchResult } from '../types'
import { api } from '../services/api'

function parseCitationInfo(url: string) {
  try {
    const parsed = new URL(url)
    const host = parsed.hostname.replace('www.', '')
    let outlet = host
    let badgeColor = 'bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-500/30 font-bold'

    if (host.includes('cnnindonesia')) {
      outlet = 'CNN Indonesia'
      badgeColor = 'bg-rose-500/15 text-rose-700 dark:text-rose-400 border-rose-500/30 font-bold'
    } else if (host.includes('cnbcindonesia')) {
      outlet = 'CNBC Indonesia'
      badgeColor = 'bg-cyan-600/15 text-cyan-800 dark:text-cyan-400 border-cyan-500/30 font-bold'
    } else if (host.includes('antaranews')) {
      outlet = 'Antara News'
      badgeColor = 'bg-amber-500/15 text-amber-800 dark:text-amber-400 border-amber-500/30 font-bold'
    } else if (host.includes('republika')) {
      outlet = 'Republika'
      badgeColor = 'bg-emerald-500/15 text-emerald-800 dark:text-emerald-400 border-emerald-500/30 font-bold'
    } else if (host.includes('kompas')) {
      outlet = 'Kompas'
      badgeColor = 'bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-500/30 font-bold'
    } else if (host.includes('detik')) {
      outlet = 'Detikcom'
      badgeColor = 'bg-indigo-500/15 text-indigo-700 dark:text-indigo-400 border-indigo-500/30 font-bold'
    }

    const segments = parsed.pathname.split('/').filter(Boolean)
    const lastSeg = segments[segments.length - 1] || ''
    let title = lastSeg
      .replace(/^[\d-]+/, '')
      .replace(/-/g, ' ')
      .trim()
    if (!title || title.length < 3) {
      title = parsed.pathname
    } else {
      title = title.charAt(0).toUpperCase() + title.slice(1)
    }

    return { outlet, title, badgeColor }
  } catch {
    return { outlet: 'Media Nasional', title: url, badgeColor: 'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-700 font-bold' }
  }
}

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

  // Resilient stream abort and auto-scroll refs
  const abortControllerRef = useRef<AbortController | null>(null)
  const answerEndRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort()
    }
  }, [])

  useEffect(() => {
    if (streaming && answerEndRef.current) {
      answerEndRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    }
  }, [streamedAnswer, streaming])

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

  const handleStreamQA = async (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    if (!question.trim()) return

    // Cancel any ongoing stream before initiating new one
    abortControllerRef.current?.abort()
    const controller = new AbortController()
    abortControllerRef.current = controller

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
        if (err.name !== 'AbortError') {
          setQaError(err.message)
        }
      },
      controller.signal
    )
  }

  return (
    <div className="space-y-8 sm:space-y-10">
      {/* Header */}
      <div className="border-b border-slate-200 dark:border-slate-800 pb-4 sm:pb-5">
        <span className="text-[11px] sm:text-xs font-mono text-amber-700 dark:text-amber-400 uppercase tracking-wider font-bold">
          RAG & pgvector Knowledge Engine
        </span>
        <h2 className="font-editorial text-xl sm:text-2xl lg:text-3xl font-bold text-slate-900 dark:text-slate-100 mt-1">
          Semantic Inquiry & Live SSE Q&A
        </h2>
        <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 mt-1 font-sans leading-relaxed max-w-2xl">
          Pencarian makna semantik mendalam pada arsip berita dan interaksi tanya jawab langsung dengan efek pengetikan <i>real-time typewriter</i> via Server-Sent Events.
        </p>
      </div>

      {/* Grid: 2 Columns on Desktop */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left Column: Live Streaming Q&A */}
        <section className="p-4 sm:p-7 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 space-y-4 sm:space-y-5 flex flex-col justify-between shadow-sm transition-colors">
          <div className="space-y-4">
            <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
              <div className="flex items-center gap-2">
                <MessageSquare className="w-5 h-5 text-amber-500" />
                <h3 className="font-editorial text-base sm:text-lg font-bold text-slate-900 dark:text-slate-100">
                  Live Typewriter Q&A (SSE)
                </h3>
              </div>
              <span className="text-[10px] sm:text-xs font-mono font-bold px-2.5 py-0.5 rounded-md bg-amber-500/10 dark:bg-amber-500/15 text-amber-800 dark:text-amber-400 border border-amber-500/30">
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
                className="w-full bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700/80 focus:border-amber-500 rounded-xl p-3.5 text-base sm:text-sm font-sans text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none transition resize-none leading-relaxed"
              />

              {/* Quick Prompt Suggestions */}
              <div className="space-y-2 pt-1">
                <span className="text-[11px] sm:text-xs font-sans text-slate-600 dark:text-slate-400 font-medium flex items-center gap-1.5">
                  <Sparkles className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" /> Contoh pertanyaan siap uji:
                </span>
                <div className="flex flex-wrap gap-2">
                  {[
                    'Apa penyebab inflasi Oktober 2026?',
                    'Berapa surplus neraca dagang RI?',
                    'Bagaimana kinerja sektor ekspor dan tambang?',
                  ].map((sample) => (
                    <button
                      key={sample}
                      type="button"
                      onClick={() => setQuestion(sample)}
                      className="text-xs font-sans px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-amber-500/10 dark:hover:bg-amber-500/15 text-slate-800 dark:text-slate-300 hover:text-amber-900 dark:hover:text-amber-200 border border-slate-200 dark:border-slate-700 hover:border-amber-500/40 transition flex items-center gap-1.5 cursor-pointer shadow-xs active:scale-98 text-left font-medium"
                    >
                      <span className="w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0" />
                      <span>{sample}</span>
                    </button>
                  ))}
                </div>
              </div>

              <div className="flex justify-end pt-2">
                <button
                  type="submit"
                  disabled={streaming || !question.trim()}
                  className="w-full sm:w-auto px-5 py-3 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 shadow-sm min-h-[44px] touch-manipulation cursor-pointer"
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
              <div className="p-3.5 sm:p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 flex items-center justify-between gap-3 text-xs font-mono text-rose-700 dark:text-rose-300">
                <div className="flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 text-rose-600 dark:text-rose-400 shrink-0" />
                  <span>{qaError}</span>
                </div>
                <button
                  type="button"
                  onClick={() => handleStreamQA()}
                  className="px-3 py-1.5 rounded-lg bg-rose-500/20 hover:bg-rose-500/30 text-rose-800 dark:text-rose-200 font-bold transition shrink-0 cursor-pointer"
                >
                  Coba Lagi
                </button>
              </div>
            )}

            {/* Answer Box with Live Screen Reader Announcement */}
            {(streamedAnswer || streaming) && (
              <div 
                aria-live="polite"
                aria-atomic="false"
                className="p-4 sm:p-5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 space-y-4 animate-in fade-in shadow-xs"
              >
                <div className="flex items-center justify-between text-xs font-mono border-b border-slate-200 dark:border-slate-800 pb-3">
                  <span className="text-emerald-700 dark:text-emerald-400 font-bold flex items-center gap-1.5">
                    <Sparkles className="w-4 h-4 text-emerald-600 dark:text-emerald-400" /> Jawaban Sintesis AI:
                  </span>
                  {provider && (
                    <span className="text-slate-600 dark:text-slate-400 font-medium px-2 py-0.5 rounded bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800">
                      Provider: <strong className="text-amber-700 dark:text-amber-400 font-bold">{provider}</strong>
                    </span>
                  )}
                </div>

                {/* Formatted Markdown Content - High Contrast Ink Text */}
                <div className="text-sm sm:text-base text-slate-900 dark:text-slate-100 leading-relaxed font-sans space-y-2">
                  <ReactMarkdown
                    components={{
                      strong: ({ children }) => (
                        <strong className="font-bold text-amber-950 dark:text-amber-200 bg-amber-400/30 dark:bg-amber-500/20 px-1.5 py-0.5 rounded border border-amber-500/40 dark:border-amber-500/30">
                          {children}
                        </strong>
                      ),
                      em: ({ children }) => <em className="italic text-slate-900 dark:text-slate-100">{children}</em>,
                      p: ({ children }) => <p className="leading-relaxed mb-2.5 last:mb-0 text-slate-900 dark:text-slate-100 font-normal">{children}</p>,
                      ul: ({ children }) => <ul className="list-disc pl-5 my-2 space-y-1.5 text-slate-800 dark:text-slate-300">{children}</ul>,
                      ol: ({ children }) => <ol className="list-decimal pl-5 my-2 space-y-1.5 text-slate-800 dark:text-slate-300">{children}</ol>,
                      li: ({ children }) => <li className="leading-relaxed">{children}</li>,
                    }}
                  >
                    {streamedAnswer}
                  </ReactMarkdown>
                  {streaming && (
                    <span className="inline-block w-2 h-4 ml-1 bg-amber-500 animate-pulse align-middle rounded-sm shadow-[0_0_8px_rgba(251,191,36,0.8)]" />
                  )}
                  {/* Auto-scroll anchor */}
                  <div ref={answerEndRef} />
                </div>

                {/* Verified Citations Cards */}
                {citations.length > 0 && (
                  <div className="pt-3 border-t border-slate-200 dark:border-slate-800 space-y-2">
                    <span className="text-xs font-mono font-semibold text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                      <Newspaper className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" /> Sumber Rujukan Terverifikasi:
                    </span>
                    <div className="grid grid-cols-1 gap-2 pt-1">
                      {citations.map((cite, i) => {
                        const info = parseCitationInfo(cite)
                        return (
                          <a
                            key={i}
                            href={cite}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-2.5 p-2.5 rounded-xl bg-white dark:bg-slate-900/90 hover:bg-slate-100 dark:hover:bg-slate-850 border border-slate-200 dark:border-slate-800 hover:border-amber-500/40 text-slate-800 dark:text-slate-300 transition group shadow-xs"
                          >
                            <span className="text-[11px] font-mono text-slate-500 shrink-0">
                              #{i + 1}
                            </span>
                            <span className={`px-2 py-0.5 rounded-md text-[10px] font-mono font-bold border uppercase shrink-0 ${info.badgeColor}`}>
                              {info.outlet}
                            </span>
                            <span className="text-xs font-sans text-slate-800 dark:text-slate-300 group-hover:text-amber-800 dark:group-hover:text-amber-100 truncate flex-1 font-medium">
                              {info.title}
                            </span>
                            <ExternalLink className="w-3.5 h-3.5 text-slate-400 group-hover:text-amber-500 shrink-0 ml-1" />
                          </a>
                        )
                      })}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </section>

        {/* Right Column: pgvector Semantic Search */}
        <section className="p-4 sm:p-7 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 space-y-4 sm:space-y-5 shadow-sm transition-colors">
          <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-800 pb-3">
            <div className="flex items-center gap-2">
              <Search className="w-5 h-5 text-amber-500" />
              <h3 className="font-editorial text-base sm:text-lg font-bold text-slate-900 dark:text-slate-100">
                pgvector Semantic Search
              </h3>
            </div>
            <span className="text-[10px] sm:text-xs font-mono font-bold px-2.5 py-0.5 rounded-md bg-blue-500/10 dark:bg-blue-500/15 text-blue-700 dark:text-blue-300 border border-blue-500/30">
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
              className="flex-1 bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700/80 focus:border-amber-500 rounded-xl px-4 py-3 text-base sm:text-sm font-mono text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none transition min-h-[44px]"
            />
            <button
              type="submit"
              disabled={searching || !searchQuery.trim()}
              className="w-full sm:w-auto px-5 py-3 rounded-xl bg-slate-900 dark:bg-slate-800 hover:bg-slate-800 dark:hover:bg-slate-700 active:scale-98 text-white text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 border border-slate-700 min-h-[44px] touch-manipulation cursor-pointer shadow-xs"
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
            <div className="p-3.5 sm:p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-xs font-mono text-rose-600 dark:text-rose-300">
              {searchError}
            </div>
          )}

          {/* Shimmer Search Loader */}
          {searching && (
            <div className="space-y-3 pt-1">
              {[1, 2, 3].map((n) => (
                <div
                  key={n}
                  className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 space-y-2.5 shimmer-wrapper"
                >
                  <div className="flex justify-between">
                    <div className="h-4 w-20 bg-slate-200 dark:bg-slate-800 rounded" />
                    <div className="h-4 w-14 bg-slate-200 dark:bg-slate-800 rounded" />
                  </div>
                  <div className="h-5 w-3/4 bg-slate-200 dark:bg-slate-800 rounded" />
                  <div className="h-10 w-full bg-slate-100 dark:bg-slate-800/50 rounded" />
                </div>
              ))}
            </div>
          )}

          {/* Results List */}
          <div className="space-y-3 pt-1">
            {!searching && searchResults.length > 0 ? (
              searchResults.map((res) => (
                <div
                  key={res.id}
                  className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700 transition space-y-1.5 shadow-xs"
                >
                  <div className="flex items-center justify-between text-xs font-mono">
                    <span className="px-2 py-0.5 rounded bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-300 border border-slate-200 dark:border-slate-700 text-[11px] font-medium">
                      {res.source || 'Media Nasional'}
                    </span>
                    {res.similarity_score !== undefined && (
                      <span className="text-amber-700 dark:text-amber-400 font-bold">
                        Skor: {(res.similarity_score * 100).toFixed(1)}%
                      </span>
                    )}
                  </div>
                  <h4 className="font-editorial text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100 leading-snug">
                    <a
                      href={res.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="hover:text-amber-600 dark:hover:text-amber-400 transition flex items-center gap-1.5"
                    >
                      <span>{res.title}</span>
                      <ExternalLink className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                    </a>
                  </h4>
                  <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed font-sans line-clamp-2">
                    {res.summary}
                  </p>
                </div>
              ))
            ) : (
              !searching && (
                <div className="p-6 text-center text-xs sm:text-sm font-mono text-slate-500 dark:text-slate-400 bg-slate-50 dark:bg-slate-950/40 rounded-xl border border-slate-200 dark:border-slate-800">
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
