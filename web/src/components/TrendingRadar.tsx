import React, { useState, useEffect } from 'react'
import { Flame, RefreshCw, Layers, ExternalLink, ArrowRight, X, AlertCircle } from 'lucide-react'
import type { TrendingClusterItem, ClusterDetailResponse } from '../types'
import { api } from '../services/api'

interface TrendingRadarProps {
  apiKey: string
  onSelectForComparison: (urls: string[]) => void
}

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

  return (
    <div className="space-y-6">
      {/* Section Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-[#1F2937] pb-4">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="font-editorial text-xl lg:text-2xl font-bold text-[#F3F4F6]">
              Trending Topics & Hotness Radar
            </h2>
            <span className="text-xs font-mono px-2 py-0.5 rounded bg-[#D97706]/15 text-[#F59E0B] border border-[#D97706]/30">
              {clusters.length} Kluster Aktif
            </span>
          </div>
          <p className="text-xs text-[#9CA3AF] mt-1 font-sans">
            Pengelompokan artikel berita otomatis lintas redaksi menggunakan kemiripan vektor kosinus dan sintesis konsensus AI.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={fetchTrending}
            disabled={loading}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#111827] hover:bg-[#182032] border border-[#1F2937] text-xs font-mono text-[#F3F4F6] transition disabled:opacity-50"
            title="Muat ulang data"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-[#9CA3AF] ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
          <button
            onClick={handleRecompute}
            disabled={recomputing}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition disabled:opacity-50"
          >
            <Flame className={`w-3.5 h-3.5 ${recomputing ? 'animate-bounce' : ''}`} />
            <span>{recomputing ? 'Recomputing...' : 'Recompute Radar'}</span>
          </button>
        </div>
      </div>

      {/* State 1: Error State */}
      {error && (
        <div className="p-4 rounded-lg bg-[#E11D48]/10 border border-[#E11D48]/30 flex items-center justify-between gap-3 text-xs font-mono text-[#FCA5A5]">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-4 h-4 text-[#E11D48] shrink-0" />
            <span>{error}</span>
          </div>
          <button
            onClick={fetchTrending}
            className="px-2.5 py-1 rounded bg-[#E11D48]/20 hover:bg-[#E11D48]/30 text-white transition"
          >
            Coba Lagi
          </button>
        </div>
      )}

      {/* State 2: Loading Skeletons */}
      {loading && !error && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {[1, 2, 3, 4].map((n) => (
            <div key={n} className="p-5 rounded-lg bg-[#111827] border border-[#1F2937] space-y-3 animate-pulse">
              <div className="flex justify-between items-center">
                <div className="h-4 w-28 bg-[#1F2937] rounded" />
                <div className="h-4 w-16 bg-[#1F2937] rounded" />
              </div>
              <div className="h-6 w-3/4 bg-[#1F2937] rounded" />
              <div className="h-14 w-full bg-[#182032] rounded" />
              <div className="h-4 w-40 bg-[#1F2937] rounded" />
            </div>
          ))}
        </div>
      )}

      {/* State 3: Composed Empty State */}
      {!loading && !error && clusters.length === 0 && (
        <div className="p-12 text-center rounded-lg bg-[#111827] border border-[#1F2937] space-y-3">
          <Layers className="w-10 h-10 text-[#6B7280] mx-auto" />
          <h3 className="font-editorial text-lg text-[#F3F4F6]">Belum Ada Kluster Topik Terdeteksi</h3>
          <p className="text-xs text-[#9CA3AF] max-w-md mx-auto">
            Database belum memiliki cukup artikel berita dalam rentang waktu terkini. Jalankan tombol <b>Recompute Radar</b> atau picu sinkronisasi berita di tab Live Wire.
          </p>
          <button
            onClick={handleRecompute}
            className="mt-2 px-4 py-2 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition inline-flex items-center gap-1.5"
          >
            <Flame className="w-4 h-4" />
            <span>Picu Hitung Kluster Sekarang</span>
          </button>
        </div>
      )}

      {/* State 4: Active Cluster Grid */}
      {!loading && !error && clusters.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {clusters.map((item, idx) => (
            <article
              key={item.id}
              className="p-5 rounded-lg bg-[#111827] hover:bg-[#151D2E] border border-[#1F2937] hover:border-[#D97706]/40 transition space-y-3 flex flex-col justify-between"
            >
              <div className="space-y-2.5">
                {/* Meta row */}
                <div className="flex items-center justify-between gap-2 text-xs font-mono">
                  <div className="flex items-center gap-2">
                    <span className="text-[#9CA3AF]">#{idx + 1}</span>
                    <span className="px-2 py-0.5 rounded bg-[#1F2937] text-[#9CA3AF] border border-[#374151]">
                      {item.article_count} artikel • {item.unique_outlets} media
                    </span>
                  </div>
                  <div className="flex items-center gap-1 text-[#F59E0B] font-semibold bg-[#D97706]/10 px-2 py-0.5 rounded border border-[#D97706]/30">
                    <Flame className="w-3.5 h-3.5" />
                    <span>Hotness {item.hotness_score.toFixed(1)}</span>
                  </div>
                </div>

                {/* Title */}
                <h3 className="font-editorial text-base lg:text-lg font-bold text-[#F3F4F6] leading-snug">
                  {item.title}
                </h3>

                {/* Outlets badges */}
                <div className="flex flex-wrap gap-1.5 pt-1">
                  {item.outlets.map((outlet, i) => (
                    <span
                      key={i}
                      className="text-[11px] font-mono px-2 py-0.5 rounded bg-[#182032] text-[#93C5FD] border border-[#1E293B]"
                    >
                      {outlet}
                    </span>
                  ))}
                </div>

                {/* Consensus Summary */}
                <p className="text-xs text-[#9CA3AF] leading-relaxed line-clamp-3 bg-[#080C14] p-3 rounded-md border border-[#1F2937]">
                  💡 <i>{item.summary}</i>
                </p>
              </div>

              {/* Action Buttons */}
              <div className="pt-2 border-t border-[#1F2937] flex items-center justify-between gap-2">
                <button
                  onClick={() => handleInspect(item.id)}
                  className="text-xs font-mono text-[#F3F4F6] hover:text-[#F59E0B] flex items-center gap-1.5 transition"
                >
                  <span>Inspect Cluster</span>
                  <ArrowRight className="w-3.5 h-3.5" />
                </button>

                <button
                  onClick={() => handleInspect(item.id)}
                  className="px-2.5 py-1 rounded bg-[#1F2937] hover:bg-[#374151] text-[#9CA3AF] hover:text-white text-xs font-mono transition"
                >
                  Detail Sumber
                </button>
              </div>
            </article>
          ))}
        </div>
      )}

      {/* Cluster Detail Modal */}
      {selectedClusterId && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg max-w-2xl w-full p-6 shadow-2xl space-y-4 max-h-[85vh] flex flex-col">
            <div className="flex items-start justify-between gap-3 border-b border-[#1F2937] pb-3">
              <div>
                <span className="text-xs font-mono text-[#D97706] uppercase tracking-wider">
                  Detail Kluster Berita #{selectedClusterId}
                </span>
                <h3 className="font-editorial text-lg font-bold text-[#F3F4F6] mt-0.5">
                  {clusterDetail ? clusterDetail.title : 'Memuat data kluster...'}
                </h3>
              </div>
              <button
                onClick={() => {
                  setSelectedClusterId(null)
                  setClusterDetail(null)
                }}
                className="text-[#9CA3AF] hover:text-[#F3F4F6] p-1 rounded-md"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {detailLoading || !clusterDetail ? (
              <div className="py-12 text-center text-xs font-mono text-[#9CA3AF] animate-pulse">
                Sedang mengumpulkan seluruh artikel terkait...
              </div>
            ) : (
              <div className="overflow-y-auto space-y-4 pr-1">
                {/* Consensus Box */}
                <div className="p-3.5 rounded-md bg-[#080C14] border border-[#1F2937] space-y-1">
                  <span className="text-[11px] font-mono text-[#F59E0B] font-semibold">
                    Konsensus Faktual AI:
                  </span>
                  <p className="text-xs text-[#D1D5DB] leading-relaxed">
                    {clusterDetail.summary}
                  </p>
                </div>

                {/* Articles List */}
                <div className="space-y-2">
                  <h4 className="text-xs font-mono text-[#9CA3AF] uppercase">
                    Daftar Artikel Sumber ({clusterDetail.articles.length} Sumber)
                  </h4>
                  <div className="space-y-2">
                    {clusterDetail.articles.map((art, i) => (
                      <div
                        key={i}
                        className="p-3 rounded-md bg-[#182032] border border-[#1E293B] space-y-1.5"
                      >
                        <div className="flex items-start justify-between gap-2">
                          <a
                            href={art.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="font-editorial text-sm font-semibold text-[#F3F4F6] hover:text-[#F59E0B] leading-snug flex items-center gap-1.5"
                          >
                            <span>{art.title}</span>
                            <ExternalLink className="w-3.5 h-3.5 shrink-0 text-[#9CA3AF]" />
                          </a>
                          <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-[#111827] text-[#93C5FD] border border-[#1F2937] shrink-0">
                            {art.source || 'Media Nasional'}
                          </span>
                        </div>
                        <p className="text-xs text-[#9CA3AF] leading-relaxed line-clamp-2">
                          {art.summary}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Compare Action */}
                {clusterDetail.articles.length >= 2 && (
                  <div className="pt-2 border-t border-[#1F2937] flex justify-end">
                    <button
                      onClick={() => {
                        const urls = clusterDetail.articles.slice(0, 3).map((a) => a.url)
                        onSelectForComparison(urls)
                        setSelectedClusterId(null)
                        setClusterDetail(null)
                      }}
                      className="px-4 py-2 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition flex items-center gap-2"
                    >
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
