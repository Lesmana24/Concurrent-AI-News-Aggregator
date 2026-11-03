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
  LayoutGrid,
  List,
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

  // View density mode: 'card' (visual editorial) vs 'compact' (high-density newsroom wire)
  const [viewMode, setViewMode] = useState<'card' | 'compact'>(() => {
    return (localStorage.getItem('the_dispatch_trending_view_mode') as 'card' | 'compact') || 'card'
  })

  const handleToggleViewMode = (mode: 'card' | 'compact') => {
    setViewMode(mode)
    localStorage.setItem('the_dispatch_trending_view_mode', mode)
  }

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
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-200 dark:border-slate-800 pb-4 sm:pb-5">
        <div>
          <div className="flex flex-wrap items-center gap-2 sm:gap-3">
            <h2 className="font-editorial text-xl sm:text-2xl lg:text-3xl font-bold tracking-tight text-slate-900 dark:text-slate-100">
              Trending Topics & Hotness Radar
            </h2>
            <span className="text-[11px] sm:text-xs font-mono font-semibold px-2 sm:px-2.5 py-0.5 sm:py-1 rounded bg-amber-500/15 text-amber-700 dark:text-amber-400 border border-amber-500/30">
              {clusters.length} Kluster Aktif
            </span>
          </div>
          <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 mt-1 font-sans max-w-2xl leading-relaxed">
            Pengelompokan analitis artikel berita lintas portal nasional, deteksi liputan utama, dan kalkulasi indeks tren <i>Hotness Score</i> real-time.
          </p>
        </div>

        {/* Action Controls - Density Toggle & Actions */}
        <div className="flex items-center gap-2 sm:gap-2.5 w-full sm:w-auto shrink-0 flex-wrap">
          {/* View Mode Toggle: Cards vs Compact Wire */}
          <div className="flex items-center p-0.5 rounded-xl bg-slate-100 dark:bg-slate-900 border border-slate-200 dark:border-slate-700/80 shadow-xs">
            <button
              onClick={() => handleToggleViewMode('card')}
              className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-mono transition cursor-pointer ${
                viewMode === 'card'
                  ? 'bg-white dark:bg-slate-800 text-amber-700 dark:text-amber-400 font-bold shadow-xs'
                  : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
              }`}
              title="Tampilan Kartu Editorial Visual (Cards View)"
              aria-label="Tampilan Kartu Editorial"
            >
              <LayoutGrid className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Cards</span>
            </button>
            <button
              onClick={() => handleToggleViewMode('compact')}
              className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-mono transition cursor-pointer ${
                viewMode === 'compact'
                  ? 'bg-white dark:bg-slate-800 text-amber-700 dark:text-amber-400 font-bold shadow-xs'
                  : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200'
              }`}
              title="Tampilan Ringkas Terminal (Compact Newsroom Wire)"
              aria-label="Tampilan Ringkas Terminal"
            >
              <List className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Compact</span>
            </button>
          </div>

          {onOpenExport && (
            <button
              onClick={onOpenExport}
              className="flex-1 sm:flex-initial flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl bg-white dark:bg-slate-900 hover:bg-slate-100 dark:hover:bg-slate-800 active:bg-slate-200 dark:active:bg-slate-750 border border-slate-200 dark:border-slate-700 text-xs font-mono font-medium text-slate-700 dark:text-slate-200 transition min-h-[40px] touch-manipulation shadow-xs cursor-pointer"
              title="Ekspor Briefing Kluster"
            >
              <Download className="w-3.5 h-3.5 text-amber-500 dark:text-amber-400" />
              <span>Export</span>
            </button>
          )}

          <button
            onClick={fetchTrending}
            disabled={loading}
            className="flex-1 sm:flex-initial flex items-center justify-center gap-1.5 px-3.5 py-2 rounded-xl bg-white dark:bg-slate-900 hover:bg-slate-100 dark:hover:bg-slate-800 active:bg-slate-200 dark:active:bg-slate-750 border border-slate-200 dark:border-slate-700 text-xs font-mono font-medium text-slate-700 dark:text-slate-200 transition min-h-[40px] disabled:opacity-50 touch-manipulation shadow-xs cursor-pointer"
            title="Muat ulang data"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-slate-400 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>

          <button
            onClick={handleRecompute}
            disabled={recomputing}
            className="flex-1 sm:flex-initial flex items-center justify-center gap-2 px-4 py-2 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white text-xs font-mono font-bold shadow-md transition min-h-[40px] disabled:opacity-50 touch-manipulation cursor-pointer"
          >
            <Flame className={`w-4 h-4 text-white ${recomputing ? 'animate-bounce' : ''}`} />
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

      {/* State 2: Loading Skeletons with Shimmer */}
      {loading && !error && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5 sm:gap-6">
          {[1, 2, 3, 4].map((n) => (
            <div key={n} className="rounded-2xl bg-white dark:bg-slate-900/60 border border-slate-200 dark:border-slate-800 overflow-hidden shadow-sm dark:shadow-lg shimmer-wrapper">
              <div className="h-44 sm:h-56 bg-slate-200 dark:bg-slate-800/70 w-full" />
              <div className="p-4 sm:p-6 space-y-3 sm:space-y-4">
                <div className="flex justify-between items-center">
                  <div className="h-4 w-24 bg-slate-200 dark:bg-slate-800 rounded" />
                  <div className="h-4 w-16 bg-slate-200 dark:bg-slate-800 rounded" />
                </div>
                <div className="h-5 w-4/5 bg-slate-200 dark:bg-slate-800 rounded" />
                <div className="h-14 w-full bg-slate-100 dark:bg-slate-800/50 rounded-lg" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* State 3: Empty State */}
      {!loading && !error && clusters.length === 0 && (
        <div className="py-16 px-4 text-center rounded-2xl bg-white dark:bg-slate-900/40 border border-slate-200 dark:border-slate-800 space-y-4 shadow-sm">
          <Layers className="w-12 h-12 text-slate-400 dark:text-slate-500 mx-auto" />
          <h3 className="font-editorial text-xl font-bold text-slate-800 dark:text-slate-200">
            Belum Ada Kluster Berita Terbentuk
          </h3>
          <p className="text-sm text-slate-600 dark:text-slate-300 max-w-md mx-auto">
            Gunakan tombol <b>Recompute</b> di atas atau jalankan sinkronisasi berita dari tab <i>Live Wire</i> untuk membentuk klaster otomatis pertama.
          </p>
        </div>
      )}

      {/* State 4: Clusters View (Cards vs Compact Wire) */}
      {!loading && !error && clusters.length > 0 && viewMode === 'card' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5 sm:gap-7 animate-in fade-in duration-200">
          {clusters.map((item, idx) => {
            const isSaved = isWatched ? isWatched(`cluster://${item.id}`) : false
            return (
              <article
                key={item.id}
                className="group rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-amber-500/50 transition-all duration-200 hover:-translate-y-0.5 overflow-hidden shadow-sm hover:shadow-md dark:shadow-xl dark:hover:shadow-2xl flex flex-col justify-between"
              >
                <div>
                  {/* Visual Header: Editorial Cover Photo with Badges */}
                  <div className="relative h-40 sm:h-52 w-full bg-slate-950 overflow-hidden">
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

                    {/* Top Badges Bar - Full-width flex container preventing any mobile overlap */}
                    <div className="absolute top-2.5 sm:top-3 inset-x-2.5 sm:inset-x-3 flex items-center justify-between gap-1.5 z-10 pointer-events-none">
                      {/* Left: Rank & Hotness */}
                      <div className="flex items-center gap-1 sm:gap-1.5 pointer-events-auto shrink-0">
                        <span className="px-2 py-0.5 sm:px-2.5 sm:py-1 rounded-lg bg-white/90 dark:bg-black/75 backdrop-blur-md border border-slate-200/80 dark:border-white/20 text-amber-800 dark:text-amber-400 font-mono text-[11px] sm:text-xs font-bold flex items-center gap-1 shadow-sm">
                          <Flame className="w-3 h-3 sm:w-3.5 sm:h-3.5 text-amber-600 dark:text-amber-400" />
                          <span>#{idx + 1}</span>
                        </span>
                        <span className="px-2 py-0.5 sm:px-2.5 sm:py-1 rounded-lg bg-white/90 dark:bg-black/75 backdrop-blur-md border border-slate-200/80 dark:border-white/20 text-slate-800 dark:text-white font-mono text-[11px] sm:text-xs font-semibold shadow-sm">
                          <span className="hidden sm:inline">Hotness </span>{item.hotness_score.toFixed(1)}
                        </span>
                      </div>

                      {/* Right: Coverage & Bookmark */}
                      <div className="flex items-center gap-1 sm:gap-1.5 pointer-events-auto shrink-0">
                        <span className="px-2 py-0.5 sm:px-2.5 sm:py-1 rounded-lg bg-white/90 dark:bg-black/75 backdrop-blur-md border border-slate-200/80 dark:border-white/20 text-slate-800 dark:text-white font-mono text-[11px] sm:text-xs flex items-center gap-1 shadow-sm">
                          <Layers className="w-3 h-3 sm:w-3.5 sm:h-3.5 text-amber-600 dark:text-amber-400" />
                          <span>{item.article_count} B<span className="hidden xs:inline">erita</span><span className="hidden sm:inline"> • {item.unique_outlets} Media</span></span>
                        </span>

                        <button
                          onClick={() => handleToggleClusterWatchlist(item)}
                          aria-label={isSaved ? `Hapus Kluster #${item.id} dari Watchlist` : `Simpan Kluster #${item.id} ke Watchlist`}
                          className={`p-1 sm:p-1.5 rounded-lg backdrop-blur-md border transition cursor-pointer shadow-sm ${
                            isSaved
                              ? 'bg-amber-500 text-white border-amber-400 font-bold'
                              : 'bg-white/90 dark:bg-black/75 text-slate-700 dark:text-white border-slate-200/80 dark:border-white/20 hover:text-amber-600 dark:hover:text-amber-400'
                          }`}
                          title={isSaved ? 'Hapus dari Watchlist' : 'Simpan Kluster ke Watchlist'}
                        >
                          {isSaved ? <BookmarkCheck className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-white" /> : <Bookmark className="w-3.5 h-3.5 sm:w-4 sm:h-4" />}
                        </button>
                      </div>
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
                  <div className="p-4 sm:p-5 space-y-2.5">
                    {/* Media Outlets Badges */}
                    <div className="flex flex-wrap items-center gap-1.5">
                      {item.outlets.map((outlet) => (
                        <span
                          key={outlet}
                          className="px-2 py-0.5 rounded text-[11px] font-mono font-medium bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700/80"
                        >
                          {outlet}
                        </span>
                      ))}
                    </div>

                    <h3 
                      onClick={() => handleInspect(item.id)}
                      className="font-editorial text-lg sm:text-xl font-bold text-slate-900 dark:text-slate-100 group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors leading-snug line-clamp-2 cursor-pointer"
                    >
                      {item.title}
                    </h3>

                    <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 leading-relaxed font-sans line-clamp-2">
                      {item.summary}
                    </p>
                  </div>
                </div>

                {/* Card Footer Actions */}
                <div className="px-4 sm:p-5 pb-4 sm:pb-5 pt-2 border-t border-slate-100 dark:border-slate-800/80 flex items-center justify-between gap-2 sm:gap-3 text-xs font-mono">
                  <div className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400 truncate">
                    <Clock className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                    <span className="truncate">{formatTimeAgo(item.last_updated)}</span>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      onClick={() => handleInspect(item.id)}
                      className="px-3 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 hover:text-slate-900 dark:hover:text-white font-medium transition flex items-center gap-1.5 min-h-[40px] touch-manipulation border border-slate-200 dark:border-slate-700 cursor-pointer"
                    >
                      <ImageIcon className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
                      <span className="hidden xs:inline">Rincian</span>
                      <span className="xs:hidden">Detail</span>
                    </button>
                    <button
                      onClick={() => handleInspect(item.id)}
                      className="px-3.5 py-2 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white font-bold transition flex items-center gap-1 shadow-xs min-h-[40px] touch-manipulation cursor-pointer"
                    >
                      <span>Telusuri</span>
                      <ArrowRight className="w-3.5 h-3.5 text-white" />
                    </button>
                  </div>
                </div>
              </article>
            )
          })}
        </div>
      )}

      {/* State 4 Alternative: Compact Newsroom Wire Mode */}
      {!loading && !error && clusters.length > 0 && viewMode === 'compact' && (
        <div className="space-y-2.5 animate-in fade-in duration-200">
          {clusters.map((item, idx) => {
            const isSaved = isWatched ? isWatched(`cluster://${item.id}`) : false
            return (
              <div
                key={item.id}
                className="group p-3 sm:p-4 rounded-xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-amber-500/50 hover:bg-slate-50/75 dark:hover:bg-slate-850/50 transition-all flex flex-col md:flex-row md:items-center justify-between gap-3 shadow-xs"
              >
                <div className="flex items-start gap-3 min-w-0 flex-1">
                  {/* Hotness Score Tag */}
                  <div className="flex flex-col items-center justify-center w-12 sm:w-14 h-12 sm:h-14 rounded-xl bg-amber-500/10 dark:bg-amber-500/15 border border-amber-500/30 shrink-0 text-center">
                    <span className="text-[10px] font-mono text-amber-700 dark:text-amber-400 font-bold">#{idx + 1}</span>
                    <span className="text-xs sm:text-sm font-mono font-bold text-amber-900 dark:text-amber-300">
                      {item.hotness_score.toFixed(1)}
                    </span>
                  </div>

                  <div className="space-y-1 min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-1.5">
                      {item.outlets.map((outlet) => (
                        <span
                          key={outlet}
                          className="px-1.5 py-0.2 rounded text-[10px] font-mono bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700"
                        >
                          {outlet}
                        </span>
                      ))}
                      <span className="text-[10px] font-mono text-slate-400 dark:text-slate-500">
                        • {item.article_count} Berita • {formatTimeAgo(item.last_updated)}
                      </span>
                    </div>

                    <h3 
                      onClick={() => handleInspect(item.id)}
                      className="font-editorial text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100 group-hover:text-amber-600 dark:group-hover:text-amber-400 transition cursor-pointer truncate"
                      title={item.title}
                    >
                      {item.title}
                    </h3>

                    <p className="text-xs text-slate-600 dark:text-slate-400 font-sans line-clamp-1">
                      {item.summary}
                    </p>
                  </div>
                </div>

                {/* Compact Row Actions */}
                <div className="flex items-center gap-2 shrink-0 self-end md:self-center">
                  <button
                    onClick={() => handleToggleClusterWatchlist(item)}
                    aria-label={isSaved ? `Hapus Kluster #${item.id} dari Watchlist` : `Simpan Kluster #${item.id} ke Watchlist`}
                    className={`p-2 rounded-lg border transition cursor-pointer ${
                      isSaved
                        ? 'bg-amber-500 text-slate-950 border-amber-400 font-bold'
                        : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 border-slate-200 dark:border-slate-700 hover:text-amber-500'
                    }`}
                    title={isSaved ? 'Hapus dari Watchlist' : 'Simpan Kluster ke Watchlist'}
                  >
                    {isSaved ? <BookmarkCheck className="w-3.5 h-3.5" /> : <Bookmark className="w-3.5 h-3.5" />}
                  </button>

                  <button
                    onClick={() => handleInspect(item.id)}
                    className="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-800 dark:text-slate-200 text-xs font-mono transition border border-slate-200 dark:border-slate-700 cursor-pointer"
                  >
                    Rincian
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Cluster Detail Modal with Mobile Responsive Layout */}
      {selectedClusterId && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-2.5 sm:p-5 animate-in fade-in duration-200">
          <div className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-2xl max-w-3xl w-full shadow-2xl flex flex-col max-h-[92vh] overflow-hidden text-slate-900 dark:text-slate-100">
            {/* Modal Top Bar */}
            <div className="p-4 sm:p-5 border-b border-slate-200 dark:border-slate-800 flex items-start justify-between gap-3 bg-white dark:bg-slate-900 sticky top-0 z-10">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-[11px] sm:text-xs font-mono font-bold text-amber-700 dark:text-amber-400 uppercase tracking-wider">
                    Kluster #{selectedClusterId}
                  </span>
                  {clusterDetail && (
                    <span className="text-[10px] sm:text-xs font-mono px-2 py-0.5 rounded bg-amber-500/15 text-amber-800 dark:text-amber-300 border border-amber-500/30 font-semibold">
                      Hotness {clusterDetail.hotness_score.toFixed(1)}
                    </span>
                  )}
                </div>
                <h3 className="font-editorial text-lg sm:text-2xl font-bold text-slate-900 dark:text-slate-100 mt-1 leading-snug line-clamp-2">
                  {clusterDetail ? clusterDetail.title : 'Memuat data kluster...'}
                </h3>
              </div>

              {/* Min 44px close button */}
              <button
                onClick={() => {
                  setSelectedClusterId(null)
                  setClusterDetail(null)
                }}
                className="w-11 h-11 rounded-xl text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 flex items-center justify-center transition shrink-0"
                aria-label="Tutup Rincian Kluster"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Modal Body */}
            {detailLoading || !clusterDetail ? (
              <div className="p-10 text-center text-xs sm:text-sm font-mono text-slate-500 dark:text-slate-400 space-y-3">
                <RefreshCw className="w-6 h-6 animate-spin text-amber-500 dark:text-amber-400 mx-auto" />
                <p>Sedang mengumpulkan artikel dan galeri foto redaksi...</p>
              </div>
            ) : (
              <div className="p-4 sm:p-6 overflow-y-auto space-y-6">
                {/* Visual Media Gallery: Dual Redaction Photo Showcase */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
                  <div className="relative rounded-xl overflow-hidden bg-slate-950 border border-slate-200 dark:border-slate-800 h-44 sm:h-48 group">
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

                  <div className="relative rounded-xl overflow-hidden bg-slate-950 border border-slate-200 dark:border-slate-800 h-44 sm:h-48 group">
                    <img
                      src={clusterDetail.secondary_image_url || FALLBACK_NEWS_SECONDARY}
                      alt="Foto Redaksi Pembanding"
                      onError={(e) => {
                        ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_SECONDARY
                      }}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                    />
                    <div className="absolute bottom-2 left-2 px-2 py-0.5 rounded bg-black/80 backdrop-blur-sm text-[10px] font-mono text-white border border-white/10">
                      Foto Redaksi Sudut Pandang Lain
                    </div>
                  </div>
                </div>

                {/* AI Executive Summary Block */}
                <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950/80 border border-slate-200 dark:border-slate-800 space-y-2">
                  <div className="flex items-center gap-1.5 text-xs font-mono text-amber-700 dark:text-amber-400 font-bold">
                    <Sparkles className="w-3.5 h-3.5 text-amber-500" />
                    <span>Ringkasan Sintesis Otomatis AI</span>
                  </div>
                  <p className="text-xs sm:text-sm text-slate-800 dark:text-slate-200 leading-relaxed font-sans">
                    {clusterDetail.summary}
                  </p>
                </div>

                {/* Aggregated Articles List */}
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs font-mono text-slate-500 dark:text-slate-400">
                    <span className="font-semibold text-slate-800 dark:text-slate-300">
                      {clusterDetail.articles.length} Liputan Redaksi Terverifikasi
                    </span>
                    <button
                      onClick={() => {
                        const urls = clusterDetail.articles.map((a) => a.url)
                        setSelectedClusterId(null)
                        onSelectForComparison(urls)
                      }}
                      className="text-amber-700 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-300 font-bold flex items-center gap-1"
                    >
                      <Scale className="w-3.5 h-3.5" />
                      <span>Bandingkan Semua di Spektrum Bias →</span>
                    </button>
                  </div>

                  <div className="space-y-2.5">
                    {clusterDetail.articles.map((art, i) => (
                      <div
                        key={i}
                        className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800/90 hover:border-slate-300 dark:hover:border-slate-700 transition flex flex-col sm:flex-row gap-3 items-start justify-between"
                      >
                        <div className="flex-1 space-y-1">
                          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-700 font-medium">
                            {art.source}
                          </span>
                          <h4 className="font-semibold text-xs sm:text-sm text-slate-900 dark:text-slate-100 hover:text-amber-600 dark:hover:text-amber-400 transition">
                            <a href={art.url} target="_blank" rel="noopener noreferrer">
                              {art.title}
                            </a>
                          </h4>
                          <p className="text-xs text-slate-600 dark:text-slate-400 line-clamp-2">
                            {art.summary}
                          </p>
                        </div>

                        <div className="flex sm:flex-col gap-2 shrink-0 self-end sm:self-center">
                          <a
                            href={art.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="p-2 rounded-lg bg-white dark:bg-slate-800 hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-300 hover:text-slate-900 dark:hover:text-white transition border border-slate-200 dark:border-slate-700"
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
            <div className="p-3.5 sm:p-4 border-t border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 flex justify-end gap-2.5">
              <button
                onClick={() => {
                  setSelectedClusterId(null)
                  setClusterDetail(null)
                }}
                className="px-4 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-slate-700 text-xs font-mono transition"
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
