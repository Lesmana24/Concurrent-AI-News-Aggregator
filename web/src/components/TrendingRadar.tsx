import React, { useState, useEffect } from 'react'
import { Flame, RefreshCw, Layers, ExternalLink, ArrowRight, X, AlertCircle, Image as ImageIcon, Sparkles, Scale, Clock } from 'lucide-react'
import type { TrendingClusterItem, ClusterDetailResponse } from '../types'
import { api } from '../services/api'

interface TrendingRadarProps {
  apiKey: string
  onSelectForComparison: (urls: string[]) => void
}

// Fallback high-resolution editorial photography for news clusters
const FALLBACK_NEWS_HERO = 'https://images.unsplash.com/photo-1504711434969-e33886168f5c?auto=format&fit=crop&w=1000&q=80'
const FALLBACK_NEWS_SECONDARY = 'https://images.unsplash.com/photo-1585829365295-ab7cd400c167?auto=format&fit=crop&w=800&q=80'

export const TrendingRadar: React.FC<TrendingRadarProps> = ({ apiKey, onSelectForComparison }) => {
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
      setClusters(data.topics || [])
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
    <div className="space-y-8">
      {/* Section Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div>
          <div className="flex items-center gap-3">
            <h2 className="font-editorial text-2xl lg:text-3xl font-bold tracking-tight text-slate-100">
              Trending Topics & Hotness Radar
            </h2>
            <span className="text-xs font-mono font-semibold px-2.5 py-1 rounded bg-amber-500/10 text-amber-400 border border-amber-500/30">
              {clusters.length} Kluster Terverifikasi
            </span>
          </div>
          <p className="text-sm text-slate-300 mt-1.5 font-sans max-w-2xl leading-relaxed">
            Pengelompokan otomatis artikel berita lintas portal nasional menggunakan kemiripan vektor AI, deteksi gambar redaksi, dan kalkulasi indeks <i>Hotness Score</i> real-time.
          </p>
        </div>

        <div className="flex items-center gap-2.5 shrink-0">
          <button
            onClick={fetchTrending}
            disabled={loading}
            className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-slate-900 hover:bg-slate-800 border border-slate-700/80 text-xs font-mono font-medium text-slate-200 transition disabled:opacity-50"
            title="Muat ulang data"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-slate-400 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
          <button
            onClick={handleRecompute}
            disabled={recomputing}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-amber-600 hover:bg-amber-500 text-white text-xs font-mono font-semibold shadow-md transition disabled:opacity-50"
          >
            <Flame className={`w-4 h-4 ${recomputing ? 'animate-bounce' : ''}`} />
            <span>{recomputing ? 'Menghitung Ulang...' : 'Recompute Radar'}</span>
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
            className="px-3 py-1.5 rounded-md bg-rose-500/20 hover:bg-rose-500/30 text-white font-medium transition"
          >
            Coba Lagi
          </button>
        </div>
      )}

      {/* State 2: Loading Skeletons */}
      {loading && !error && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {[1, 2, 3, 4].map((n) => (
            <div key={n} className="rounded-xl bg-slate-900/60 border border-slate-800 overflow-hidden shadow-lg animate-pulse">
              <div className="h-48 sm:h-56 bg-slate-800/70 w-full" />
              <div className="p-6 space-y-4">
                <div className="flex justify-between items-center">
                  <div className="h-4 w-28 bg-slate-800 rounded" />
                  <div className="h-4 w-20 bg-slate-800 rounded" />
                </div>
                <div className="h-6 w-4/5 bg-slate-800 rounded" />
                <div className="h-16 w-full bg-slate-800/50 rounded-lg" />
                <div className="h-4 w-36 bg-slate-800 rounded" />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* State 3: Empty State */}
      {!loading && !error && clusters.length === 0 && (
        <div className="p-12 text-center rounded-2xl bg-slate-900/40 border border-slate-800 space-y-4 max-w-xl mx-auto">
          <div className="w-14 h-14 rounded-full bg-slate-800/80 border border-slate-700 flex items-center justify-center mx-auto text-slate-400">
            <Layers className="w-7 h-7" />
          </div>
          <h3 className="font-editorial text-xl font-bold text-slate-100">Belum Ada Kluster Berita Terdeteksi</h3>
          <p className="text-sm text-slate-300 leading-relaxed">
            Database belum memiliki cukup artikel berita dalam rentang waktu terkini. Jalankan tombol <b>Recompute Radar</b> atau picu sinkronisasi berita di tab Live Wire.
          </p>
          <button
            onClick={handleRecompute}
            className="mt-2 px-5 py-2.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-white text-xs font-mono font-semibold transition inline-flex items-center gap-2 shadow-lg"
          >
            <Flame className="w-4 h-4" />
            <span>Picu Hitung Kluster Sekarang</span>
          </button>
        </div>
      )}

      {/* State 4: Active Cluster Grid with Rich Visual Cards */}
      {!loading && !error && clusters.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {clusters.map((item, idx) => {
            const primaryImg = item.image_url || FALLBACK_NEWS_HERO
            const secondaryImg = item.secondary_image_url || null

            return (
              <article
                key={item.id}
                className="group rounded-2xl bg-slate-900/80 hover:bg-slate-900 border border-slate-800/90 hover:border-amber-500/40 transition-all duration-300 shadow-xl overflow-hidden flex flex-col justify-between"
              >
                <div>
                  {/* Visual Header: 16:9 Hero Image Banner with Dual-Image Feature */}
                  <div className="relative aspect-[16/9] w-full bg-slate-950 overflow-hidden border-b border-slate-800/80">
                    {/* Primary Hero Image */}
                    <img
                      src={primaryImg}
                      alt={item.title}
                      loading="lazy"
                      onError={(e) => {
                        ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_HERO
                      }}
                      className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500 ease-out"
                    />

                    {/* Gradient Overlay for Readable Badges */}
                    <div className="absolute inset-0 bg-gradient-to-t from-slate-950 via-slate-950/30 to-black/30 pointer-events-none" />

                    {/* Top Floating Badges */}
                    <div className="absolute top-3 left-3 right-3 flex items-center justify-between gap-2">
                      {/* Hotness Badge */}
                      <div className="flex items-center gap-1.5 bg-amber-500 text-slate-950 px-3 py-1 rounded-md text-xs font-mono font-bold shadow-lg backdrop-blur-md">
                        <Flame className="w-3.5 h-3.5 fill-slate-950" />
                        <span>Hotness {item.hotness_score.toFixed(1)}</span>
                      </div>

                      {/* Rank & Outlet Counter */}
                      <div className="flex items-center gap-1.5 bg-slate-900/90 border border-slate-700/80 text-slate-200 px-3 py-1 rounded-md text-xs font-mono backdrop-blur-md shadow-md">
                        <span className="text-amber-400 font-bold">#{idx + 1}</span>
                        <span className="text-slate-400">•</span>
                        <span>{item.unique_outlets} Redaksi</span>
                        <span className="text-slate-400">•</span>
                        <span>{item.article_count} Berita</span>
                      </div>
                    </div>

                    {/* Secondary Visual: Dual-Angle Thumbnail Inset if 2nd image available */}
                    {secondaryImg && (
                      <div
                        className="absolute bottom-3 right-3 group/thumb"
                        title="Foto Perspektif Sumber Redaksi Lain"
                      >
                        <div className="relative w-20 sm:w-24 h-14 sm:h-16 rounded-lg overflow-hidden border-2 border-slate-900 shadow-2xl bg-slate-800">
                          <img
                            src={secondaryImg}
                            alt="Perspektif Media Lain"
                            onError={(e) => {
                              ;(e.target as HTMLImageElement).style.display = 'none'
                            }}
                            className="w-full h-full object-cover group-hover/thumb:scale-110 transition-transform duration-300"
                          />
                          <div className="absolute inset-0 bg-black/20" />
                          <div className="absolute bottom-0 inset-x-0 bg-slate-950/90 text-[10px] font-mono text-center text-amber-300 py-0.5 border-t border-slate-800">
                            +1 Foto
                          </div>
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Card Content Body */}
                  <div className="p-6 space-y-4">
                    {/* Media Outlets Tags */}
                    <div className="flex flex-wrap items-center gap-1.5">
                      {item.outlets.map((outlet, i) => (
                        <span
                          key={i}
                          className="text-xs font-mono px-2.5 py-0.5 rounded bg-slate-800/80 text-slate-200 border border-slate-700/60 font-medium"
                        >
                          {outlet}
                        </span>
                      ))}
                    </div>

                    {/* Editorial Headline */}
                    <h3
                      onClick={() => handleInspect(item.id)}
                      className="font-editorial text-xl font-bold text-slate-100 group-hover:text-amber-400 transition-colors leading-snug cursor-pointer line-clamp-2"
                    >
                      {item.title}
                    </h3>

                    {/* Synthesized Consensus Summary */}
                    <div className="p-4 rounded-xl bg-slate-950/70 border border-slate-800/80 space-y-1.5">
                      <div className="flex items-center gap-1.5 text-xs font-mono font-semibold text-amber-400">
                        <Sparkles className="w-3.5 h-3.5" />
                        <span>Konsensus Fakta AI</span>
                      </div>
                      <p className="text-sm text-slate-200 leading-relaxed font-sans line-clamp-3">
                        {item.summary}
                      </p>
                    </div>
                  </div>
                </div>

                {/* Card Footer Actions */}
                <div className="px-6 pb-6 pt-2 border-t border-slate-800/80 flex items-center justify-between gap-3 text-xs font-mono">
                  <div className="flex items-center gap-1.5 text-slate-400">
                    <Clock className="w-3.5 h-3.5 text-slate-400" />
                    <span>{formatTimeAgo(item.last_updated)}</span>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleInspect(item.id)}
                      className="px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 hover:text-white font-medium transition flex items-center gap-1.5"
                    >
                      <ImageIcon className="w-3.5 h-3.5 text-amber-400" />
                      <span>Rincian & Foto</span>
                    </button>
                    <button
                      onClick={() => handleInspect(item.id)}
                      className="px-3.5 py-1.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-white font-semibold transition flex items-center gap-1 shadow-sm"
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

      {/* Cluster Detail Modal with Rich Multi-Media Presentation */}
      {selectedClusterId && (
        <div className="fixed inset-0 z-50 bg-black/85 backdrop-blur-md flex items-center justify-center p-3 sm:p-5 animate-in fade-in duration-200">
          <div className="bg-slate-900 border border-slate-700/80 rounded-2xl max-w-3xl w-full shadow-2xl flex flex-col max-h-[90vh] overflow-hidden">
            {/* Modal Top Bar */}
            <div className="p-5 border-b border-slate-800 flex items-start justify-between gap-4 bg-slate-900/95 sticky top-0 z-10">
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-mono font-bold text-amber-400 uppercase tracking-wider">
                    Kluster Berita #{selectedClusterId}
                  </span>
                  {clusterDetail && (
                    <span className="text-xs font-mono px-2 py-0.5 rounded bg-amber-500/15 text-amber-300 border border-amber-500/30">
                      Hotness {clusterDetail.hotness_score.toFixed(1)}
                    </span>
                  )}
                </div>
                <h3 className="font-editorial text-xl sm:text-2xl font-bold text-slate-100 mt-1 leading-snug">
                  {clusterDetail ? clusterDetail.title : 'Memuat data kluster...'}
                </h3>
              </div>

              <button
                onClick={() => {
                  setSelectedClusterId(null)
                  setClusterDetail(null)
                }}
                className="text-slate-400 hover:text-white p-1.5 rounded-lg hover:bg-slate-800 transition shrink-0"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Modal Body */}
            {detailLoading || !clusterDetail ? (
              <div className="p-12 text-center text-sm font-mono text-slate-400 space-y-3">
                <RefreshCw className="w-6 h-6 animate-spin text-amber-400 mx-auto" />
                <p>Sedang menyusun artikel dan galeri foto dari seluruh redaksi...</p>
              </div>
            ) : (
              <div className="overflow-y-auto p-6 space-y-6">
                {/* 1. Visual Showcase Header: 1 or 2 Photos */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 rounded-xl overflow-hidden border border-slate-800 bg-slate-950 p-2">
                  <div className="relative aspect-[16/10] rounded-lg overflow-hidden bg-slate-900">
                    <img
                      src={clusterDetail.image_url || FALLBACK_NEWS_HERO}
                      alt="Foto Redaksi Utama"
                      onError={(e) => {
                        ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_HERO
                      }}
                      className="w-full h-full object-cover"
                    />
                    <div className="absolute bottom-2 left-2 bg-slate-950/85 backdrop-blur-md px-2 py-0.5 rounded text-[11px] font-mono text-slate-300 border border-slate-800">
                      Foto Sumber Utama
                    </div>
                  </div>

                  <div className="relative aspect-[16/10] rounded-lg overflow-hidden bg-slate-900">
                    <img
                      src={clusterDetail.secondary_image_url || FALLBACK_NEWS_SECONDARY}
                      alt="Foto Redaksi Pembanding"
                      onError={(e) => {
                        ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_SECONDARY
                      }}
                      className="w-full h-full object-cover"
                    />
                    <div className="absolute bottom-2 left-2 bg-slate-950/85 backdrop-blur-md px-2 py-0.5 rounded text-[11px] font-mono text-slate-300 border border-slate-800">
                      Perspektif Visual Terkait
                    </div>
                  </div>
                </div>

                {/* 2. AI Consensus Narrative */}
                <div className="p-5 rounded-xl bg-slate-950/80 border border-slate-800 space-y-2">
                  <div className="flex items-center gap-2 text-xs font-mono font-bold text-amber-400">
                    <Sparkles className="w-4 h-4" />
                    <span>RANGKUMAN KONSENSUS FAKTUAL LINTAS MEDIA</span>
                  </div>
                  <p className="text-sm sm:text-base text-slate-200 leading-relaxed font-sans">
                    {clusterDetail.summary}
                  </p>
                </div>

                {/* 3. Source Articles List with Photography Thumbnails */}
                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-xs font-mono uppercase tracking-wider text-slate-400 font-bold">
                      Artikel Sumber Berita ({clusterDetail.articles.length} Sumber)
                    </h4>
                    <span className="text-xs font-mono text-slate-400">
                      {clusterDetail.unique_outlets} Redaksi Berbeda
                    </span>
                  </div>

                  <div className="space-y-3">
                    {clusterDetail.articles.map((art, i) => (
                      <div
                        key={i}
                        className="p-4 rounded-xl bg-slate-950/60 hover:bg-slate-950 border border-slate-800/90 hover:border-slate-700 transition flex flex-col sm:flex-row gap-4"
                      >
                        {/* Article Thumbnail */}
                        <div className="w-full sm:w-36 h-28 sm:h-24 rounded-lg overflow-hidden bg-slate-900 border border-slate-800 shrink-0">
                          <img
                            src={art.image_url || clusterDetail.image_url || FALLBACK_NEWS_HERO}
                            alt={art.title}
                            onError={(e) => {
                              ;(e.target as HTMLImageElement).src = FALLBACK_NEWS_HERO
                            }}
                            className="w-full h-full object-cover"
                          />
                        </div>

                        {/* Article Info */}
                        <div className="flex-1 space-y-2 flex flex-col justify-between">
                          <div>
                            <div className="flex items-center justify-between gap-2 mb-1">
                              <span className="text-xs font-mono font-medium px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                                {art.source || 'Media Nasional'}
                              </span>
                            </div>
                            <a
                              href={art.url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="font-editorial text-base font-bold text-slate-100 hover:text-amber-400 transition leading-snug flex items-start gap-1.5 group"
                            >
                              <span>{art.title}</span>
                              <ExternalLink className="w-4 h-4 text-slate-400 group-hover:text-amber-400 shrink-0 mt-0.5" />
                            </a>
                          </div>
                          <p className="text-xs sm:text-sm text-slate-300 leading-relaxed line-clamp-2">
                            {art.summary}
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* 4. Action: Compare Bias & Perspectives */}
                {clusterDetail.articles.length >= 2 && (
                  <div className="pt-4 border-t border-slate-800 flex justify-end">
                    <button
                      onClick={() => {
                        const urls = clusterDetail.articles.slice(0, 3).map((a) => a.url)
                        onSelectForComparison(urls)
                        setSelectedClusterId(null)
                        setClusterDetail(null)
                      }}
                      className="w-full sm:w-auto px-5 py-2.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-white text-xs font-mono font-semibold transition flex items-center justify-center gap-2 shadow-lg"
                    >
                      <Scale className="w-4 h-4" />
                      <span>Bandingkan Sudut Pandang Media Ini di Spektrum Bias</span>
                      <ArrowRight className="w-4 h-4" />
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
