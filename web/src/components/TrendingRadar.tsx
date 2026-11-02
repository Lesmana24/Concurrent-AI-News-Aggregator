import React, { useState, useEffect } from 'react'
import {
  Flame,
  RefreshCw,
  Layers,
  ExternalLink,
  ArrowRight,
  X,
  AlertCircle,
  Image as ImageIcon,
  Sparkles,
  Scale,
  Clock,
  Download,
  Bookmark,
  BookmarkCheck,
} from 'lucide-react'
import type { TrendingClusterItem, ClusterDetailResponse } from '../types'
import { api } from '../services/api'
import type { WatchlistItem } from './WatchlistDrawer'

interface TrendingRadarProps {
  apiKey: string
  onSelectForComparison: (urls: string[]) => void
  onOpenExport?: () => void
  onToggleWatchlist?: (item: WatchlistItem) => void
  isWatched?: (url: string) => boolean
  onClustersLoaded?: (clusters: TrendingClusterItem[]) => void
}

// Fallback high-resolution editorial photography for news clusters
const FALLBACK_NEWS_HERO = 'https://images.unsplash.com/photo-1504711434969-e33886168f5c?auto=format&fit=crop&w=1000&q=80'
const FALLBACK_NEWS_SECONDARY = 'https://images.unsplash.com/photo-1585829365295-ab7cd400c167?auto=format&fit=crop&w=800&q=80'

export const TrendingRadar: React.FC<TrendingRadarProps> = ({
  apiKey,
  onSelectForComparison,
  onOpenExport,
  onToggleWatchlist,
  isWatched,
  onClustersLoaded,
}) => {
  const [clusters, setClusters] = useState<TrendingClusterItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedClusterId, setSelectedClusterId] = useState<number | null>(null)
  const [clusterDetail, setClusterDetail] = useState<ClusterDetailResponse | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [recomputing, setRecomputing] = useState(false)

  const fetchTrending = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await api.getTrending(8)
      const list = data.topics || []
      setClusters(list)
      onClustersLoaded?.(list)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memuat radar tren')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchTrending()
  }, [])

  const handleInspect = async (id: number) => {
    setSelectedClusterId(id)
    setDetailLoading(true)
    try {
      const detail = await api.getClusterDetail(id)
      setClusterDetail(detail)
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Gagal memuat rincian kluster')
      setSelectedClusterId(null)
    } finally {
      setDetailLoading(false)
    }
  }

  const handleRecompute = async () => {
    setRecomputing(true)
    try {
      await api.recomputeClusters(apiKey)
      await fetchTrending()
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Gagal menghitung ulang kluster')
    } finally {
      setRecomputing(false)
    }
  }

  const handleToggleClusterWatchlist = (item: TrendingClusterItem) => {
    if (!onToggleWatchlist) return
    onToggleWatchlist({
      id: `cluster-${item.id}`,
      title: item.title,
      url: `cluster://${item.id}`,
      source: `Kluster #${item.id} (${item.outlets.join(', ')})`,
      type: 'cluster',
      summary: item.summary,
      dateAdded: new Date().toLocaleDateString('id-ID', { day: 'numeric', month: 'short' }),
    })
  }

  const formatTimeAgo = (dateStr?: string) => {
    if (!dateStr) return 'Terkini'
    const date = new Date(dateStr)
    if (isNaN(date.getTime())) return 'Terkini'
    return date.toLocaleString('id-ID', {
      day: 'numeric',
      month: 'short',
      hour: '2-digit',
      minute: '2-digit',
    }) + ' WIB'
  }

  return (
    <div className="space-y-6 sm:space-y-8">
      {/* Section Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-4 sm:pb-5">
        <div>
          <div className="flex flex-wrap items-center gap-2 sm:gap-3">
            <h2 className="font-editorial text-xl sm:text-2xl lg:text-3xl font-bold tracking-tight text-slate-100">
              Trending Topics & Hotness Radar
            </h2>
            <span className="text-[11px] sm:text-xs font-mono font-semibold px-2 sm:px-2.5 py-0.5 sm:py-1 rounded bg-amber-500/10 text-amber-400 border border-amber-500/30">
              {clusters.length} Kluster Aktif
            </span>
          </div>
          <p className="text-xs sm:text-sm text-slate-300 mt-1 font-sans max-w-2xl leading-relaxed">
            Pengelompokan otomatis artikel berita lintas portal nasional menggunakan kemiripan vektor AI, deteksi foto redaksi, dan kalkulasi indeks <i>Hotness Score</i> real-time.
          </p>
        </div>

        {/* Action Controls - Full width flex on mobile */}
        <div className="flex items-center gap-2 sm:gap-2.5 w-full sm:w-auto shrink-0 flex-wrap">
          {onOpenExport && (
            <button
              onClick={onOpenExport}
              className="flex-1 sm:flex-initial flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 active:bg-slate-750 border border-slate-700 text-xs font-mono font-medium text-slate-200 transition min-h-[40px] touch-manipulation"
              title="Ekspor Briefing Kluster"
            >
              <Download className="w-3.5 h-3.5 text-amber-400" />
              <span>Export</span>
            </button>
          )}

          <button
            onClick={fetchTrending}
            disabled={loading}
            className="flex-1 sm:flex-initial flex items-center justify-center gap-1.5 px-3.5 py-2 rounded-xl bg-slate-900 hover:bg-slate-800 active:bg-slate-750 border border-slate-700 text-xs font-mono font-medium text-slate-200 transition min-h-[40px] disabled:opacity-50 touch-manipulation"
            title="Muat ulang data"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-slate-400 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>

          <button
            onClick={handleRecompute}
            disabled={recomputing}
            className="flex-1 sm:flex-initial flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white text-xs font-mono font-bold shadow-md transition min-h-[40px] disabled:opacity-50 touch-manipulation"
          >
            <Flame className={`w-4 h-4 ${recomputing ? 'animate-bounce' : ''}`} />
            <span>{recomputing ? 'Menghitung...' : 'Recompute'}</span>
          </button>
        </div>
      </div>

      {/* State 1: Error State */}
      {error && (
        <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 flex items-center justify-between gap-3 text-xs font-mono text-rose-300">
          <div className="flex items-center gap-2.5">
            <AlertCircle className="w-5 h-5 text-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
          <button
            onClick={fetchTrending}
            className="px-3 py-1.5 rounded-lg bg-rose-500/20 hover:bg-rose-500/30 text-white font-medium transition min-h-[36px]"
          >
            Coba Lagi
          </button>
        </div>
      )}

      {/* State 2: Loading Skeletons */}
      {loading && !error && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5 sm:gap-6">
          {[1, 2, 3, 4].map((n) => (
            <div key={n} className="rounded-2xl bg-slate-900/60 border border-slate-800 overflow-hidden shadow-lg animate-pulse">
              <div className="h-44 sm:h-56 bg-slate-800/70 w-full" />
              <div className="p-4 sm:p-6 space-y-3 sm:space-y-4">
                <div className="flex justify-between items-center">
                  <div className="h-4 w-24 bg-slate-800 rounded" />
                  <div className="h-4 w-16 bg-slate-800 rounded" />
                </div>
                <div className="h-5 w-4/5 bg-slate-800 rounded" />
                <div className="h-14 w-full bg-slate-800/50 rounded-lg" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* State 3: Empty State */}
      {!loading && !error && clusters.length === 0 && (
        <div className="py-16 px-4 text-center rounded-2xl bg-slate-900/40 border border-slate-800 space-y-4">
          <Layers className="w-12 h-12 text-slate-500 mx-auto" />
          <h3 className="font-editorial text-xl font-bold text-slate-200">
            Belum Ada Kluster Berita Terbentuk
          </h3>
          <p className="text-sm text-slate-300 max-w-md mx-auto">
            Gunakan tombol <b>Recompute</b> di atas atau jalankan sinkronisasi berita dari tab <i>Live Wire</i> untuk membentuk klaster otomatis pertama.
          </p>
        </div>
      )}

      {/* State 4: Clusters Grid Cards */}
      {!loading && !error && clusters.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5 sm:gap-7">
          {clusters.map((item, idx) => {
            const isSaved = isWatched ? isWatched(`cluster://${item.id}`) : false
            return (
              <article
                key={item.id}
                className="group rounded-2xl bg-slate-900/90 border border-slate-800 hover:border-amber-500/50 transition-all duration-200 overflow-hidden shadow-xl hover:shadow-2xl flex flex-col justify-between"
              >
                <div>
                  {/* Visual Header: Editorial Cover Photo with Badges */}
                  <div className="relative h-44 sm:h-56 w-full bg-slate-950 overflow-hidden">
                    <img
                      src={item.image_url || FALLBACK_NEWS_HERO}
                      alt={item.title}
                      loading="lazy"
                      onError={(e) => {
                        ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_HERO
                      }}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500 brightness-95 group-hover:brightness-100"
                    />

                    {/* Dark gradient overlay for text readability */}
                    <div className="absolute inset-0 bg-gradient-to-t from-slate-950 via-slate-950/40 to-transparent" />

                    {/* Hotness Badge & Topic Rank */}
                    <div className="absolute top-3 left-3 flex items-center gap-2">
                      <span className="px-2.5 py-1 rounded-lg bg-black/75 backdrop-blur-md border border-white/10 text-amber-400 font-mono text-xs font-bold flex items-center gap-1 shadow-md">
                        <Flame className="w-3.5 h-3.5 text-amber-500" />
                        <span>#{idx + 1}</span>
                      </span>
                      <span className="px-2.5 py-1 rounded-lg bg-black/75 backdrop-blur-md border border-white/10 text-slate-200 font-mono text-xs font-semibold shadow-md">
                        Hotness {item.hotness_score.toFixed(1)}
                      </span>
                    </div>

                    {/* Coverage Outlets Badge */}
                    <div className="absolute top-3 right-3 flex items-center gap-2">
                      <button
                        onClick={() => handleToggleClusterWatchlist(item)}
                        className={`p-1.5 rounded-lg backdrop-blur-md border transition ${
                          isSaved
                            ? 'bg-amber-500 text-slate-950 border-amber-400 shadow-md'
                            : 'bg-black/75 text-slate-200 border-white/10 hover:text-amber-400'
                        }`}
                        title={isSaved ? 'Hapus dari Watchlist' : 'Simpan Kluster ke Watchlist'}
                      >
                        {isSaved ? <BookmarkCheck className="w-4 h-4" /> : <Bookmark className="w-4 h-4" />}
                      </button>

                      <span className="px-2.5 py-1 rounded-lg bg-black/75 backdrop-blur-md border border-white/10 text-slate-300 font-mono text-xs flex items-center gap-1.5 shadow-md">
                        <Layers className="w-3.5 h-3.5 text-amber-400" />
                        <span>{item.article_count} Artikel • {item.unique_outlets} Redaksi</span>
                      </span>
                    </div>

                    {/* Secondary thumbnail teaser if present */}
                    {item.secondary_image_url && (
                      <div className="absolute bottom-3 right-3 hidden sm:block w-16 h-12 rounded-lg overflow-hidden border border-white/20 shadow-lg">
                        <img
                          src={item.secondary_image_url}
                          alt="Sudut pandang alternatif"
                          onError={(e) => {
                            ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_SECONDARY
                          }}
                          className="w-full h-full object-cover"
                        />
                      </div>
                    )}
                  </div>

                  {/* Body Content */}
                  <div className="p-4 sm:p-6 space-y-3">
                    {/* Media Outlets Badges */}
                    <div className="flex flex-wrap items-center gap-1.5">
                      {item.outlets.map((outlet) => (
                        <span
                          key={outlet}
                          className="px-2 py-0.5 rounded text-[11px] font-mono font-medium bg-slate-800/80 text-slate-300 border border-slate-700/80"
                        >
                          {outlet}
                        </span>
                      ))}
                    </div>

                    <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-100 group-hover:text-amber-400 transition-colors leading-snug line-clamp-2">
                      {item.title}
                    </h3>

                    <p className="text-xs sm:text-sm text-slate-200 leading-relaxed font-sans line-clamp-3">
                      {item.summary}
                    </p>
                  </div>
                </div>

                {/* Card Footer Actions (Min 44px Touch Targets) */}
                <div className="px-4 sm:px-6 pb-4 sm:pb-6 pt-2 border-t border-slate-800/80 flex items-center justify-between gap-2 sm:gap-3 text-xs font-mono">
                  <div className="flex items-center gap-1.5 text-slate-400 truncate">
                    <Clock className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                    <span className="truncate">{formatTimeAgo(item.last_updated)}</span>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      onClick={() => handleInspect(item.id)}
                      className="px-3 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 hover:text-white font-medium transition flex items-center gap-1.5 min-h-[40px] touch-manipulation"
                    >
                      <ImageIcon className="w-3.5 h-3.5 text-amber-400" />
                      <span className="hidden xs:inline">Rincian & Foto</span>
                      <span className="xs:hidden">Detail</span>
                    </button>
                    <button
                      onClick={() => handleInspect(item.id)}
                      className="px-3.5 py-2 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white font-bold transition flex items-center gap-1 shadow-sm min-h-[40px] touch-manipulation"
                    >
                      <span>Telusuri</span>
                      <ArrowRight className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              </article>
            )
          })}
        </div>
      )}

      {/* Cluster Detail Modal with Mobile Responsive Layout */}
      {selectedClusterId && (
        <div className="fixed inset-0 z-50 bg-black/85 backdrop-blur-md flex items-center justify-center p-2.5 sm:p-5 animate-in fade-in duration-200">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl max-w-3xl w-full shadow-2xl flex flex-col max-h-[92vh] overflow-hidden">
            {/* Modal Top Bar */}
            <div className="p-4 sm:p-5 border-b border-slate-800 flex items-start justify-between gap-3 bg-slate-900 sticky top-0 z-10">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-[11px] sm:text-xs font-mono font-bold text-amber-400 uppercase tracking-wider">
                    Kluster #{selectedClusterId}
                  </span>
                  {clusterDetail && (
                    <span className="text-[10px] sm:text-xs font-mono px-2 py-0.5 rounded bg-amber-500/15 text-amber-300 border border-amber-500/30">
                      Hotness {clusterDetail.hotness_score.toFixed(1)}
                    </span>
                  )}
                </div>
                <h3 className="font-editorial text-lg sm:text-2xl font-bold text-slate-100 mt-1 leading-snug line-clamp-2">
                  {clusterDetail ? clusterDetail.title : 'Memuat data kluster...'}
                </h3>
              </div>

              {/* Min 44px close button */}
              <button
                onClick={() => {
                  setSelectedClusterId(null)
                  setClusterDetail(null)
                }}
                className="w-11 h-11 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800 flex items-center justify-center transition shrink-0"
                aria-label="Tutup Rincian Kluster"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Modal Body */}
            {detailLoading || !clusterDetail ? (
              <div className="p-10 text-center text-xs sm:text-sm font-mono text-slate-400 space-y-3">
                <RefreshCw className="w-6 h-6 animate-spin text-amber-400 mx-auto" />
                <p>Sedang mengumpulkan artikel dan galeri foto redaksi...</p>
              </div>
            ) : (
              <div className="p-4 sm:p-6 overflow-y-auto space-y-6">
                {/* Visual Media Gallery: Dual Redaction Photo Showcase */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
                  <div className="relative rounded-xl overflow-hidden bg-slate-950 border border-slate-800 h-44 sm:h-48 group">
                    <img
                      src={clusterDetail.image_url || FALLBACK_NEWS_HERO}
                      alt="Foto Redaksi Utama"
                      onError={(e) => {
                        ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_HERO
                      }}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                    <div className="absolute bottom-2 left-2 px-2 py-0.5 rounded bg-black/80 backdrop-blur-sm text-[10px] font-mono text-amber-400 border border-white/10">
                      Foto Redaksi Utama
                    </div>
                  </div>

                  <div className="relative rounded-xl overflow-hidden bg-slate-950 border border-slate-800 h-44 sm:h-48 group">
                    <img
                      src={clusterDetail.secondary_image_url || FALLBACK_NEWS_SECONDARY}
                      alt="Foto Redaksi Pembanding"
                      onError={(e) => {
                        ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_SECONDARY
                      }}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                    <div className="absolute bottom-2 left-2 px-2 py-0.5 rounded bg-black/80 backdrop-blur-sm text-[10px] font-mono text-slate-300 border border-white/10">
                      Foto Redaksi Sudut Pandang Lain
                    </div>
                  </div>
                </div>

                {/* AI Executive Summary Block */}
                <div className="p-4 rounded-xl bg-slate-950/80 border border-slate-800 space-y-2">
                  <div className="flex items-center gap-1.5 text-xs font-mono text-amber-400 font-bold">
                    <Sparkles className="w-3.5 h-3.5 text-amber-400" />
                    <span>Ringkasan Sintesis Otomatis AI</span>
                  </div>
                  <p className="text-xs sm:text-sm text-slate-200 leading-relaxed font-sans">
                    {clusterDetail.summary}
                  </p>
                </div>

                {/* Aggregated Articles List */}
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs font-mono text-slate-400">
                    <span className="font-semibold text-slate-300">
                      {clusterDetail.articles.length} Liputan Redaksi Terverifikasi
                    </span>
                    <button
                      onClick={() => {
                        const urls = clusterDetail.articles.map((a) => a.url)
                        setSelectedClusterId(null)
                        onSelectForComparison(urls)
                      }}
                      className="text-amber-400 hover:text-amber-300 font-bold flex items-center gap-1"
                    >
                      <Scale className="w-3.5 h-3.5" />
                      <span>Bandingkan Semua di Spektrum Bias →</span>
                    </button>
                  </div>

                  <div className="space-y-2.5">
                    {clusterDetail.articles.map((art, i) => (
                      <div
                        key={i}
                        className="p-3.5 rounded-xl bg-slate-950 border border-slate-800/90 hover:border-slate-700 transition flex flex-col sm:flex-row gap-3 items-start justify-between"
                      >
                        <div className="flex-1 space-y-1">
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                            {art.source}
                          </span>
                          <h4 className="font-semibold text-xs sm:text-sm text-slate-100 hover:text-amber-400 transition">
                            <a href={art.url} target="_blank" rel="noopener noreferrer">
                              {art.title}
                            </a>
                          </h4>
                          <p className="text-xs text-slate-400 line-clamp-2">
                            {art.summary}
                          </p>
                        </div>

                        <div className="flex sm:flex-col gap-2 shrink-0 self-end sm:self-center">
                          <a
                            href={art.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition"
                            title="Buka Artikel Sumber"
                          >
                            <ExternalLink className="w-3.5 h-3.5" />
                          </a>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}

            {/* Modal Bottom Bar */}
            <div className="p-3.5 sm:p-4 border-t border-slate-800 bg-slate-900 flex justify-end gap-2.5">
              <button
                onClick={() => {
                  setSelectedClusterId(null)
                  setClusterDetail(null)
                }}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-mono transition"
              >
                Tutup
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
