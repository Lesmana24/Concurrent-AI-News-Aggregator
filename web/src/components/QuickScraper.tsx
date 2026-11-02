import React, { useState, useEffect } from 'react'
import { Radio, Send, ExternalLink, RefreshCw, CheckCircle2, AlertCircle, Scale, Image as ImageIcon } from 'lucide-react'
import type { Article } from '../types'
import { api } from '../services/api'

interface QuickScraperProps {
  apiKey: string
  onSelectForComparison: (urls: string[]) => void
}

const FALLBACK_ARTICLE_IMAGE = 'https://images.unsplash.com/photo-1504711434969-e33886168f5c?auto=format&fit=crop&w=600&q=80'

export const QuickScraper: React.FC<QuickScraperProps> = ({ apiKey, onSelectForComparison }) => {
  const [urlInput, setUrlInput] = useState('')
  const [ingesting, setIngesting] = useState(false)
  const [ingestedResult, setIngestedResult] = useState<Article | null>(null)
  const [ingestError, setIngestError] = useState<string | null>(null)

  // Wire feed state
  const [articles, setArticles] = useState<Article[]>([])
  const [selectedSource, setSelectedSource] = useState<string>('')
  const [loadingWire, setLoadingWire] = useState(true)
  const [wireError, setWireError] = useState<string | null>(null)

  const sources = [
    { label: 'Semua Media', value: '' },
    { label: 'Antara News', value: 'Antara News' },
    { label: 'CNBC Indonesia', value: 'CNBC Indonesia' },
    { label: 'CNN Indonesia', value: 'CNN Indonesia' },
    { label: 'Republika', value: 'Republika' },
  ]

  const fetchWire = async (sourceFilter = selectedSource) => {
    setLoadingWire(true)
    setWireError(null)
    try {
      const data = await api.getRecentArticles(20, sourceFilter)
      setArticles(data.articles || [])
    } catch (err: unknown) {
      setWireError(err instanceof Error ? err.message : 'Gagal memuat feed berita terkini')
    } finally {
      setLoadingWire(false)
    }
  }

  useEffect(() => {
    fetchWire(selectedSource)
  }, [selectedSource])

  const handleIngestURL = async (e: React.FormEvent) => {
    e.preventDefault()
    const targetUrl = urlInput.trim()
    if (!targetUrl) return

    setIngesting(true)
    setIngestedResult(null)
    setIngestError(null)

    try {
      const result = await api.summarizeURL(apiKey, targetUrl)
      setIngestedResult(result)
      setUrlInput('')
      fetchWire()
    } catch (err: unknown) {
      setIngestError(err instanceof Error ? err.message : 'Gagal mengekstrak berita dari URL')
    } finally {
      setIngesting(false)
    }
  }

  const formatTimestamp = (dateStr?: string) => {
    if (!dateStr) return 'Baru saja'
    const date = new Date(dateStr)
    if (isNaN(date.getTime())) return 'Baru saja'
    return date.toLocaleString('id-ID', {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    }) + ' WIB'
  }

  return (
    <div className="space-y-8 sm:space-y-10">
      {/* 1. Quick Scraper Sandbox */}
      <section className="p-4 sm:p-7 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-4 sm:space-y-5 shadow-xl">
        <div>
          <span className="text-[11px] sm:text-xs font-mono text-amber-400 uppercase tracking-wider font-bold">
            Sandbox Ekstraksi Berita Instan
          </span>
          <h2 className="font-editorial text-xl sm:text-2xl font-bold text-slate-100 mt-1">
            Quick URL Scraper & AI Summarizer
          </h2>
          <p className="text-xs sm:text-sm text-slate-300 mt-1 font-sans leading-relaxed">
            Tempelkan tautan artikel dari portal media nasional untuk menguji ekstraksi metadata, foto redaksi, dan ringkasan AI seketika.
          </p>
        </div>

        <form onSubmit={handleIngestURL} className="flex flex-col sm:flex-row gap-2.5">
          <input
            type="url"
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            placeholder="https://www.antaranews.com/berita/... atau cnbcindonesia.com/..."
            required
            className="flex-1 bg-slate-950 border border-slate-700/80 focus:border-amber-500 rounded-xl px-4 py-3 text-base sm:text-sm font-mono text-slate-100 placeholder:text-slate-500 focus:outline-none transition-colors min-h-[44px]"
          />
          <button
            type="submit"
            disabled={ingesting || !urlInput.trim()}
            className="w-full sm:w-auto px-6 py-3 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 shrink-0 shadow-lg min-h-[44px] touch-manipulation"
          >
            {ingesting ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                <span>Mengekstrak...</span>
              </>
            ) : (
              <>
                <Send className="w-4 h-4" />
                <span>Ingest URL</span>
              </>
            )}
          </button>
        </form>

        {/* Ingest Error State */}
        {ingestError && (
          <div className="p-3.5 sm:p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 flex items-center gap-2.5 text-xs font-mono text-rose-300">
            <AlertCircle className="w-4 h-4 text-rose-400 shrink-0" />
            <span>{ingestError}</span>
          </div>
        )}

        {/* Ingested Result Card */}
        {ingestedResult && (
          <div className="p-4 sm:p-5 rounded-xl bg-slate-950 border border-emerald-500/40 space-y-3 sm:space-y-4 shadow-lg animate-in fade-in">
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-emerald-400 flex items-center gap-1.5 font-bold">
                <CheckCircle2 className="w-4 h-4" /> Ekstraksi Berhasil
              </span>
              <span className="px-2.5 py-0.5 rounded-md bg-slate-800 text-slate-200 border border-slate-700 font-medium">
                {ingestedResult.source || 'Media Terverifikasi'}
              </span>
            </div>

            <div className="flex flex-col sm:flex-row gap-3.5 sm:gap-4 items-start">
              {/* Image */}
              <div className="w-full sm:w-48 h-40 sm:h-32 rounded-xl overflow-hidden bg-slate-900 border border-slate-800 shrink-0">
                <img
                  src={ingestedResult.image_url || FALLBACK_ARTICLE_IMAGE}
                  alt={ingestedResult.title}
                  onError={(e) => {
                    ;(e.target as HTMLImageElement).src = FALLBACK_ARTICLE_IMAGE
                  }}
                  className="w-full h-full object-cover"
                />
              </div>

              <div className="flex-1 space-y-2 w-full">
                <h4 className="font-editorial text-base sm:text-lg font-bold text-slate-100 leading-snug">
                  {ingestedResult.title}
                </h4>
                <p className="text-xs sm:text-sm text-slate-200 leading-relaxed font-sans bg-slate-900/60 p-3 rounded-lg border border-slate-800">
                  {ingestedResult.summary}
                </p>
                <div className="flex flex-wrap items-center justify-between gap-2 pt-1 text-xs font-mono">
                  <a
                    href={ingestedResult.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-slate-400 hover:text-amber-400 flex items-center gap-1.5 transition min-h-[36px]"
                  >
                    <span>Lihat Halaman Asli</span>
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                  <button
                    onClick={() => onSelectForComparison([ingestedResult.url])}
                    className="text-amber-400 hover:text-amber-300 font-bold min-h-[36px] flex items-center gap-1"
                  >
                    + Masukkan ke Spektrum Bias
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </section>

      {/* 2. Live Media Wire Feed */}
      <section className="space-y-5 sm:space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-4 border-b border-slate-800 pb-4">
          <div>
            <div className="flex items-center gap-2">
              <Radio className="w-5 h-5 text-amber-500 shrink-0" />
              <h3 className="font-editorial text-xl sm:text-2xl font-bold text-slate-100">
                Live News Wire Feed
              </h3>
            </div>
            <p className="text-xs sm:text-sm text-slate-300 font-sans mt-0.5">
              Arsip berita yang dikumpulkan mandiri oleh Harvester latar belakang tiap siklus cron.
            </p>
          </div>

          <button
            onClick={() => fetchWire()}
            disabled={loadingWire}
            className="flex items-center justify-center gap-2 px-3.5 py-2.5 rounded-xl bg-slate-900 hover:bg-slate-800 border border-slate-700 text-xs font-mono text-slate-200 transition min-h-[44px] touch-manipulation self-start sm:self-auto"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-slate-400 ${loadingWire ? 'animate-spin' : ''}`} />
            <span>Refresh Wire</span>
          </button>
        </div>

        {/* Source Filter Pills (Swipeable on Mobile) */}
        <div className="flex items-center gap-2 overflow-x-auto no-scrollbar overscroll-x-contain py-1">
          {sources.map((src) => (
            <button
              key={src.value}
              onClick={() => setSelectedSource(src.value)}
              className={`px-3.5 py-2 rounded-xl text-xs font-mono whitespace-nowrap shrink-0 transition min-h-[40px] touch-manipulation ${
                selectedSource === src.value
                  ? 'bg-amber-600 text-white font-bold shadow-md'
                  : 'bg-slate-900 text-slate-400 hover:text-slate-100 border border-slate-800 hover:border-slate-700'
              }`}
            >
              {src.label}
            </button>
          ))}
        </div>

        {/* Wire Articles List */}
        {loadingWire && (
          <div className="space-y-4">
            {[1, 2, 3, 4].map((n) => (
              <div key={n} className="p-4 sm:p-5 rounded-2xl bg-slate-900/60 border border-slate-800 space-y-3 animate-pulse flex flex-col sm:flex-row gap-4">
                <div className="w-full sm:w-40 h-36 sm:h-24 bg-slate-800 rounded-xl shrink-0" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 w-32 bg-slate-800 rounded" />
                  <div className="h-5 w-3/4 bg-slate-800 rounded" />
                  <div className="h-10 w-full bg-slate-800/60 rounded" />
                </div>
              </div>
            ))}
          </div>
        )}

        {!loadingWire && wireError && (
          <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-xs font-mono text-rose-300">
            {wireError}
          </div>
        )}

        {!loadingWire && !wireError && articles.length === 0 && (
          <div className="p-8 sm:p-10 text-center rounded-2xl bg-slate-900/40 border border-slate-800 text-xs sm:text-sm font-mono text-slate-400">
            Belum ada artikel yang sesuai dengan filter ini.
          </div>
        )}

        {!loadingWire && !wireError && articles.length > 0 && (
          <div className="space-y-4">
            {articles.map((art) => (
              <article
                key={art.id}
                className="group p-4 sm:p-5 rounded-2xl bg-slate-900/80 hover:bg-slate-900 border border-slate-800/90 hover:border-amber-500/40 transition-all shadow-md flex flex-col sm:flex-row gap-4 sm:gap-5 items-start"
              >
                {/* Article Thumbnail */}
                <div className="w-full sm:w-44 h-40 sm:h-28 rounded-xl overflow-hidden bg-slate-950 border border-slate-800 shrink-0 relative">
                  <img
                    src={art.image_url || FALLBACK_ARTICLE_IMAGE}
                    alt={art.title}
                    loading="lazy"
                    onError={(e) => {
                      ;(e.target as HTMLImageElement).src = FALLBACK_ARTICLE_IMAGE
                    }}
                    className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                  />
                  {!art.image_url && (
                    <div className="absolute top-2 left-2 bg-slate-950/80 p-1 rounded text-slate-400">
                      <ImageIcon className="w-3 h-3" />
                    </div>
                  )}
                </div>

                {/* Article Content */}
                <div className="flex-1 space-y-2 w-full flex flex-col justify-between">
                  <div>
                    <div className="flex flex-wrap items-center justify-between gap-1.5 text-xs font-mono mb-1">
                      <span className="px-2.5 py-0.5 rounded-md bg-slate-800 text-slate-200 border border-slate-700/80 font-medium text-[11px] sm:text-xs">
                        {art.source || 'Media Nasional'}
                      </span>
                      <span className="text-slate-400 text-[11px] sm:text-xs">
                        {formatTimestamp(art.created_at || art.published_at)}
                      </span>
                    </div>

                    <h4 className="font-editorial text-base sm:text-lg font-bold text-slate-100 group-hover:text-amber-400 transition-colors leading-snug">
                      <a
                        href={art.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="hover:underline flex items-start gap-1.5"
                      >
                        <span>{art.title}</span>
                        <ExternalLink className="w-4 h-4 text-slate-400 group-hover:text-amber-400 shrink-0 mt-0.5" />
                      </a>
                    </h4>
                  </div>

                  <p className="text-xs sm:text-sm text-slate-200 leading-relaxed font-sans line-clamp-2">
                    {art.summary}
                  </p>

                  <div className="flex justify-end pt-1">
                    <button
                      onClick={() => onSelectForComparison([art.url])}
                      className="text-xs font-mono font-bold text-amber-400 hover:text-amber-300 transition flex items-center gap-1.5 min-h-[40px] touch-manipulation"
                    >
                      <Scale className="w-3.5 h-3.5" />
                      <span>+ Masukkan ke Spektrum Bias</span>
                    </button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
