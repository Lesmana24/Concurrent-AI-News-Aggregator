import React, { useState, useEffect, useMemo } from 'react'
import {
  Radio,
  Send,
  ExternalLink,
  RefreshCw,
  CheckCircle2,
  AlertCircle,
  Scale,
  Image as ImageIcon,
  Search,
  Bookmark,
  BookmarkCheck,
} from 'lucide-react'
import type { Article } from '../types'
import { api } from '../services/api'
import type { WatchlistItem } from './WatchlistDrawer'

interface QuickScraperProps {
  apiKey: string
  onSelectForComparison: (urls: string[]) => void
  onToggleWatchlist?: (item: WatchlistItem) => void
  isWatched?: (url: string) => boolean
}

const FALLBACK_ARTICLE_IMAGE = 'https://images.unsplash.com/photo-1504711434969-e33886168f5c?auto=format&fit=crop&w=600&q=80'

export const QuickScraper: React.FC<QuickScraperProps> = ({
  apiKey,
  onSelectForComparison,
  onToggleWatchlist,
  isWatched,
}) => {
  const [urlInput, setUrlInput] = useState('')
  const [ingesting, setIngesting] = useState(false)
  const [ingestedResult, setIngestedResult] = useState<Article | null>(null)
  const [ingestError, setIngestError] = useState<string | null>(null)

  // Wire feed state
  const [articles, setArticles] = useState<Article[]>([])
  const [selectedSource, setSelectedSource] = useState<string>('')
  const [searchFilter, setSearchFilter] = useState<string>('')
  const [loadingWire, setLoadingWire] = useState(true)
  const [wireError, setWireError] = useState<string | null>(null)

  const sources = [
    { label: 'Semua Media', value: '' },
    { label: 'CNN Indonesia', value: 'CNN Indonesia' },
    { label: 'Antara News', value: 'Antara News' },
    { label: 'CNBC Indonesia', value: 'CNBC Indonesia' },
    { label: 'Republika', value: 'Republika' },
    { label: 'Detikcom', value: 'Detikcom' },
    { label: 'Kompas', value: 'Kompas' },
  ]

  const fetchWire = async (sourceFilter = selectedSource) => {
    setLoadingWire(true)
    setWireError(null)
    try {
      const data = await api.getRecentArticles(30, sourceFilter)
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

  // Filtered articles based on live search
  const filteredArticles = useMemo(() => {
    if (!searchFilter.trim()) return articles
    const q = searchFilter.toLowerCase().trim()
    return articles.filter(
      (a) =>
        a.title.toLowerCase().includes(q) ||
        a.summary.toLowerCase().includes(q) ||
        a.source.toLowerCase().includes(q)
    )
  }, [articles, searchFilter])

  const handleIngestURL = async (e: React.FormEvent) => {
    e.preventDefault()
    const targetUrl = urlInput.trim()
    if (!targetUrl) return

    // Client-side URL Validation
    try {
      const parsed = new URL(targetUrl)
      if (!['http:', 'https:'].includes(parsed.protocol)) {
        setIngestError('URL harus menggunakan protokol http:// atau https://')
        return
      }
      if (!parsed.hostname.includes('.')) {
        setIngestError('Hostname URL tidak valid')
        return
      }
    } catch {
      setIngestError('Format URL tidak valid. Contoh: https://www.cnnindonesia.com/ekonomi/...')
      return
    }

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

  const handleToggleArticleWatchlist = (art: Article) => {
    if (!onToggleWatchlist) return
    onToggleWatchlist({
      id: `art-${art.id ?? art.url}-${art.url}`,
      title: art.title,
      url: art.url,
      source: art.source,
      type: 'article',
      summary: art.summary,
      dateAdded: new Date().toLocaleDateString('id-ID', { day: 'numeric', month: 'short' }),
    })
  }

  return (
    <div className="space-y-8 sm:space-y-10">
      {/* 1. Quick Scraper Sandbox */}
      <section className="p-4 sm:p-7 rounded-2xl bg-white dark:bg-slate-900/80 border border-slate-200 dark:border-slate-800 space-y-4 sm:space-y-5 shadow-sm dark:shadow-xl transition-colors">
        <div>
          <span className="text-[11px] sm:text-xs font-mono text-amber-700 dark:text-amber-400 uppercase tracking-wider font-bold">
            Portal Ekstraksi Berita Terverifikasi
          </span>
          <h2 className="font-editorial text-xl sm:text-2xl font-bold text-slate-900 dark:text-slate-100 mt-1">
            Instant URL Ingest & Executive Summarizer
          </h2>
          <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 mt-1 font-sans leading-relaxed">
            Tempelkan tautan artikel dari portal media nasional untuk memproses ekstraksi metadata, verifikasi foto redaksi, dan analisis ringkasan seketika.
          </p>
        </div>

        <form onSubmit={handleIngestURL} className="flex flex-col sm:flex-row gap-2.5">
          <input
            type="url"
            required
            placeholder="https://www.cnnindonesia.com/nasional/..."
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            className="flex-1 px-4 py-3 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-amber-500 focus:outline-none text-slate-900 dark:text-slate-100 text-xs sm:text-sm font-sans placeholder-slate-400 transition min-h-[44px]"
          />
          <button
            type="submit"
            disabled={ingesting}
            className="px-6 py-3 rounded-xl bg-amber-500 hover:bg-amber-400 active:scale-95 text-slate-950 font-semibold text-xs sm:text-sm font-mono transition flex items-center justify-center gap-2 shrink-0 shadow-md shadow-amber-500/20 disabled:opacity-50 min-h-[44px] touch-manipulation cursor-pointer"
          >
            {ingesting ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                <span>Menganalisis Konten...</span>
              </>
            ) : (
              <>
                <Send className="w-4 h-4" />
                <span>Ingest & Ringkas</span>
              </>
            )}
          </button>
        </form>

        {ingestError && (
          <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-600 dark:text-rose-300 text-xs font-mono flex items-center gap-2">
            <AlertCircle className="w-4 h-4 shrink-0 text-rose-500" />
            <span>{ingestError}</span>
          </div>
        )}

        {ingestedResult && (
          <div className="p-4 sm:p-5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-amber-500/40 space-y-3">
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-emerald-700 dark:text-emerald-400 flex items-center gap-1.5 font-bold">
                <CheckCircle2 className="w-4 h-4" /> Berhasil Diekstrak & Diringkas
              </span>
              <span className="text-slate-500 dark:text-slate-400">{ingestedResult.source}</span>
            </div>

            <div className="flex flex-col sm:flex-row gap-4 items-start">
              <div className="w-full sm:w-44 h-32 rounded-lg overflow-hidden bg-slate-100 dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shrink-0">
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
                <h4 className="font-editorial text-base sm:text-lg font-bold text-slate-900 dark:text-slate-100 leading-snug">
                  {ingestedResult.title}
                </h4>
                <p className="text-xs sm:text-sm text-slate-700 dark:text-slate-200 leading-relaxed font-sans bg-white dark:bg-slate-900/60 p-3 rounded-lg border border-slate-200 dark:border-slate-800">
                  {ingestedResult.summary}
                </p>
                <div className="flex flex-wrap items-center justify-between gap-2 pt-1 text-xs font-mono">
                  <a
                    href={ingestedResult.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-slate-600 dark:text-slate-400 hover:text-amber-600 dark:hover:text-amber-400 flex items-center gap-1.5 transition min-h-[36px]"
                  >
                    <span>Lihat Halaman Asli</span>
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                  <div className="flex items-center gap-3">
                    <button
                      onClick={() => handleToggleArticleWatchlist(ingestedResult)}
                      className="text-slate-600 dark:text-slate-400 hover:text-amber-600 dark:hover:text-amber-400 min-h-[36px] flex items-center gap-1"
                    >
                      <Bookmark className="w-3.5 h-3.5" />
                      <span>Simpan</span>
                    </button>
                    <button
                      onClick={() => onSelectForComparison([ingestedResult.url])}
                      className="text-amber-700 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-300 font-bold min-h-[36px] flex items-center gap-1"
                    >
                      + Masukkan ke Spektrum Bias
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </section>

      {/* 2. Live Media Wire Feed */}
      <section className="space-y-5 sm:space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 sm:gap-4 border-b border-slate-200 dark:border-slate-800 pb-4">
          <div>
            <div className="flex items-center gap-2">
              <Radio className="w-5 h-5 text-amber-500 shrink-0" />
              <h3 className="font-editorial text-xl sm:text-2xl font-bold text-slate-900 dark:text-slate-100">
                Live News Wire Feed
              </h3>
              <span className="text-xs font-mono font-semibold text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded-full border border-slate-200 dark:border-slate-700">
                {filteredArticles.length} Berita
              </span>
            </div>
            <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-sans mt-0.5">
              Arsip berita yang dikumpulkan mandiri oleh Harvester latar belakang tiap siklus cron.
            </p>
          </div>

          <div className="flex items-center gap-2 self-start sm:self-auto">
            <button
              onClick={() => fetchWire()}
              disabled={loadingWire}
              className="flex items-center justify-center gap-2 px-3.5 py-2 rounded-xl bg-white dark:bg-slate-900 hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700 text-xs font-mono text-slate-700 dark:text-slate-200 transition min-h-[40px] touch-manipulation shadow-xs cursor-pointer"
            >
              <RefreshCw className={`w-3.5 h-3.5 text-slate-400 ${loadingWire ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>
        </div>

        {/* Multi-Facet Filter Bar: Search Box + Media Outlets */}
        <div className="flex flex-col sm:flex-row gap-3 items-stretch sm:items-center justify-between">
          {/* Live Search Input */}
          <div className="relative flex-1 max-w-md">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={searchFilter}
              onChange={(e) => setSearchFilter(e.target.value)}
              placeholder="Cari kata kunci dalam feed (misal: 'inflasi', 'gibran', 'ekonomi')..."
              className="w-full pl-9 pr-8 py-2 rounded-xl bg-white dark:bg-slate-900/90 border border-slate-200 dark:border-slate-700/80 focus:border-amber-500 focus:outline-none text-xs sm:text-sm text-slate-900 dark:text-slate-100 placeholder-slate-400 font-sans min-h-[40px]"
            />
            {searchFilter && (
              <button
                onClick={() => setSearchFilter('')}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-xs font-mono text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
              >
                ✕
              </button>
            )}
          </div>

          {/* Source Filter Pills (Swipeable on Mobile) */}
          <div className="flex items-center gap-1.5 overflow-x-auto no-scrollbar overscroll-x-contain py-1">
            {sources.map((src) => (
              <button
                key={src.value}
                onClick={() => setSelectedSource(src.value)}
                className={`px-3 py-1.5 rounded-lg text-xs font-mono whitespace-nowrap shrink-0 transition min-h-[36px] touch-manipulation cursor-pointer ${
                  selectedSource === src.value
                    ? 'bg-amber-500 text-slate-950 font-bold shadow-xs'
                    : 'bg-slate-100 dark:bg-slate-900 text-slate-700 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 border border-slate-200 dark:border-slate-800'
                }`}
              >
                {src.label}
              </button>
            ))}
          </div>
        </div>

        {/* Wire Articles List with Shimmer */}
        {loadingWire && (
          <div className="space-y-4">
            {[1, 2, 3, 4].map((n) => (
              <div key={n} className="p-4 sm:p-5 rounded-2xl bg-white dark:bg-slate-900/60 border border-slate-200 dark:border-slate-800 space-y-3 flex flex-col sm:flex-row gap-4 shadow-xs shimmer-wrapper">
                <div className="w-full sm:w-40 h-36 sm:h-24 bg-slate-200 dark:bg-slate-800 rounded-xl shrink-0" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 w-32 bg-slate-200 dark:bg-slate-800 rounded" />
                  <div className="h-5 w-3/4 bg-slate-200 dark:bg-slate-800 rounded" />
                  <div className="h-10 w-full bg-slate-100 dark:bg-slate-800/60 rounded" />
                </div>
              </div>
            ))}
          </div>
        )}

        {!loadingWire && wireError && (
          <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-xs font-mono text-rose-600 dark:text-rose-300">
            {wireError}
          </div>
        )}

        {!loadingWire && !wireError && filteredArticles.length === 0 && (
          <div className="p-8 sm:p-10 text-center rounded-2xl bg-white dark:bg-slate-900/40 border border-slate-200 dark:border-slate-800 text-xs sm:text-sm font-mono text-slate-500 dark:text-slate-400">
            Belum ada artikel yang sesuai dengan filter atau kata kunci "{searchFilter}".
          </div>
        )}

        {!loadingWire && !wireError && filteredArticles.length > 0 && (
          <div className="space-y-4">
            {filteredArticles.map((art) => {
              const bookmarked = isWatched ? isWatched(art.url) : false
              return (
                <article
                  key={art.id ?? art.url}
                  className="group p-4 sm:p-5 rounded-2xl bg-white dark:bg-slate-900/80 hover:bg-slate-50 dark:hover:bg-slate-900 border border-slate-200 dark:border-slate-800/90 hover:border-amber-500/40 transition-all shadow-xs hover:shadow-md dark:shadow-md flex flex-col sm:flex-row gap-4 sm:gap-5 items-start"
                >
                  {/* Article Thumbnail */}
                  <div className="w-full sm:w-44 h-40 sm:h-28 rounded-xl overflow-hidden bg-slate-100 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 shrink-0 relative">
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
                      <div className="absolute top-2 left-2 bg-white/90 dark:bg-black/75 p-1 rounded-lg border border-slate-200/80 dark:border-white/20 text-slate-700 dark:text-white shadow-xs">
                        <ImageIcon className="w-3 h-3" />
                      </div>
                    )}
                  </div>

                  {/* Article Content */}
                  <div className="flex-1 space-y-2 w-full flex flex-col justify-between">
                    <div>
                      <div className="flex flex-wrap items-center justify-between gap-1.5 text-xs font-mono mb-1">
                        <span className="px-2.5 py-0.5 rounded-md bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-slate-700/80 font-medium text-[11px] sm:text-xs">
                          {art.source || 'Media Nasional'}
                        </span>
                        <span className="text-slate-500 dark:text-slate-400 text-[11px] sm:text-xs">
                          {formatTimestamp(art.created_at || art.published_at)}
                        </span>
                      </div>

                      <h4 className="font-editorial text-base sm:text-lg font-bold text-slate-900 dark:text-slate-100 group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors leading-snug">
                        <a
                          href={art.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="hover:underline flex items-start gap-1.5"
                        >
                          <span>{art.title}</span>
                          <ExternalLink className="w-4 h-4 text-slate-400 group-hover:text-amber-600 dark:group-hover:text-amber-400 shrink-0 mt-0.5" />
                        </a>
                      </h4>
                    </div>

                    <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed font-sans line-clamp-2">
                      {art.summary}
                    </p>

                    <div className="flex items-center justify-between pt-2 border-t border-slate-100 dark:border-slate-800">
                      {/* Watchlist Toggle Button */}
                      <button
                        onClick={() => handleToggleArticleWatchlist(art)}
                        className={`text-xs font-mono transition flex items-center gap-1.5 min-h-[36px] px-2.5 py-1 rounded-lg border ${
                          bookmarked
                            ? 'bg-amber-500/15 border-amber-500/40 text-amber-800 dark:text-amber-400 font-bold'
                            : 'bg-slate-100 dark:bg-slate-950 border-slate-200 dark:border-slate-800 text-slate-700 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:border-slate-300 dark:hover:border-slate-700'
                        }`}
                        title={bookmarked ? 'Hapus dari Watchlist' : 'Simpan ke Watchlist'}
                      >
                        {bookmarked ? (
                          <>
                            <BookmarkCheck className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
                            <span>Tersimpan</span>
                          </>
                        ) : (
                          <>
                            <Bookmark className="w-3.5 h-3.5" />
                            <span>Watchlist</span>
                          </>
                        )}
                      </button>

                      {/* Bias Spectrum Compare */}
                      <button
                        onClick={() => onSelectForComparison([art.url])}
                        className="text-xs font-mono font-bold text-amber-700 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-300 transition flex items-center gap-1.5 min-h-[36px] touch-manipulation cursor-pointer"
                      >
                        <Scale className="w-3.5 h-3.5" />
                        <span>+ Spektrum Bias</span>
                      </button>
                    </div>
                  </div>
                </article>
              )
            })}
          </div>
        )}
      </section>
    </div>
  )
}
