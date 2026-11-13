import React, { useState, useEffect } from 'react'
import { Scale, Plus, Trash2, CheckCircle2, AlertTriangle, RefreshCw, BarChart2, ExternalLink } from 'lucide-react'
import type { NewsCompareResponse } from '../types'
import { api } from '../services/api'

interface MediaBiasSpectrumProps {
  apiKey: string
  prefillUrls?: string[]
  onQuotaUsed?: (amount?: number) => void
}

export const MediaBiasSpectrum: React.FC<MediaBiasSpectrumProps> = ({ apiKey, prefillUrls, onQuotaUsed }) => {
  const [urls, setUrls] = useState<string[]>(['', ''])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<NewsCompareResponse | null>(null)

  // Presets for 1-click playground testing
  const presets = [
    {
      title: 'Inflasi Pangan Nasional Oktober 2026',
      desc: 'Antara News vs CNBC Indonesia vs Republika',
      urls: [
        'https://www.antaranews.com/berita/5825751/bps-ri-alami-inflasi-bulanan-022-persen-pada-oktober-2026',
        'https://www.cnbcindonesia.com/news/20261102093945-4-781412/breaking-news-inflasi-ri-022-pada-oktober-2026',
        'https://ekonomi.republika.co.id/berita/tnuyob423/makanan-dan-tembakau-picu-inflasi-oktober-022-persen',
      ],
    },
    {
      title: 'Surplus Neraca Dagang September 2026',
      desc: 'CNBC Indonesia vs Antara News',
      urls: [
        'https://www.cnbcindonesia.com/news/20261102095945-4-781425/neraca-dagang-ri-september-2026-surplus-usd-336-miliar',
        'https://www.antaranews.com/berita/5825810/bps-neraca-perdagangan-indonesia-surplus-336-miliar-dolar-as',
      ],
    },
  ]

  useEffect(() => {
    if (prefillUrls && prefillUrls.length >= 2) {
      setUrls(prefillUrls)
      handleCompare(prefillUrls)
    } else if (prefillUrls && prefillUrls.length === 1) {
      setUrls([prefillUrls[0], ''])
    }
  }, [prefillUrls])

  const handleUrlChange = (index: number, val: string) => {
    const updated = [...urls]
    updated[index] = val
    setUrls(updated)
  }

  const addUrlField = () => {
    if (urls.length < 5) {
      setUrls([...urls, ''])
    }
  }

  const removeUrlField = (index: number) => {
    if (urls.length > 2) {
      setUrls(urls.filter((_, i) => i !== index))
    }
  }

  const handleCompare = async (targetUrls = urls) => {
    const validUrls = targetUrls.map((u) => u.trim()).filter((u) => u.length > 0)
    if (validUrls.length < 2) {
      setError('Mohon masukkan minimal 2 URL berita berbeda untuk dibandingkan.')
      return
    }

    setLoading(true)
    setError(null)
    setResult(null)

    try {
      const data = await api.compareNews(apiKey, validUrls)
      setResult(data)
      onQuotaUsed?.(1)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Gagal memproses perbandingan bias media')
    } finally {
      setLoading(false)
    }
  }

  const handleApplyPreset = (presetUrls: string[]) => {
    setUrls(presetUrls)
    handleCompare(presetUrls)
  }

  return (
    <div className="space-y-8 sm:space-y-10">
      {/* Header */}
      <div className="border-b border-slate-200 dark:border-slate-800 pb-4 sm:pb-5">
        <span className="text-[11px] sm:text-xs font-mono text-amber-700 dark:text-amber-400 uppercase tracking-wider font-bold">
          Analisis Komparatif Multi-Perspektif
        </span>
        <h2 className="font-editorial text-xl sm:text-2xl lg:text-3xl font-bold text-slate-900 dark:text-slate-100 mt-1">
          Media Bias & Sentiment Spectrum
        </h2>
        <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 mt-1 font-sans leading-relaxed max-w-2xl">
          Membedah secara visual bagaimana berbagai redaksi media membingkai (<i>framing</i>) peristiwa yang sama dengan skor sentimen objektif, konsensus fakta, dan titik divergensi narasi.
        </p>
      </div>

      {/* Preset Buttons for Quick 1-Click Testing */}
      <div className="space-y-2.5 sm:space-y-3">
        <span className="text-[11px] sm:text-xs font-mono text-slate-600 dark:text-slate-400 font-semibold uppercase tracking-wider">
          Uji Coba Cepat dengan Sampel Kasus Nyata:
        </span>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
          {presets.map((p, idx) => (
            <button
              key={idx}
              onClick={() => handleApplyPreset(p.urls)}
              disabled={loading}
              className="p-3.5 sm:p-4 rounded-xl bg-white dark:bg-slate-900/80 hover:bg-slate-50 dark:hover:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-amber-500/50 text-left transition-all shadow-xs dark:shadow-md flex items-start justify-between gap-3 group min-h-[44px] touch-manipulation cursor-pointer"
            >
              <div>
                <h4 className="font-editorial text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100 group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors">
                  {p.title}
                </h4>
                <p className="text-[11px] sm:text-xs text-slate-500 dark:text-slate-400 mt-0.5 font-mono">{p.desc}</p>
              </div>
              <span className="text-xs font-mono font-bold text-amber-700 dark:text-amber-400 shrink-0 mt-0.5 group-hover:translate-x-0.5 transition-transform">
                Tes Kasus ↗
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* URL Inputs Form */}
      <section className="p-4 sm:p-7 rounded-2xl bg-white dark:bg-slate-900/80 border border-slate-200 dark:border-slate-800 space-y-4 sm:space-y-5 shadow-sm dark:shadow-xl transition-colors">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-mono text-slate-700 dark:text-slate-300 font-bold uppercase tracking-wider">
            Tautan Artikel untuk Diaudit (2 - 5 Sumber)
          </h3>
          {urls.length < 5 && (
            <button
              type="button"
              onClick={addUrlField}
              className="text-xs font-mono text-amber-700 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-300 font-bold flex items-center gap-1.5 transition min-h-[40px] px-2 touch-manipulation cursor-pointer"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Tambah URL</span>
            </button>
          )}
        </div>

        <div className="space-y-3">
          {urls.map((url, index) => (
            <div key={index} className="flex items-center gap-2 sm:gap-2.5">
              <span className="text-xs font-mono text-slate-400 dark:text-slate-500 w-5 text-right font-bold shrink-0">
                {index + 1}.
              </span>
              <input
                type="url"
                value={url}
                onChange={(e) => handleUrlChange(index, e.target.value)}
                placeholder={`URL Berita Media #${index + 1}`}
                className="flex-1 bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700/80 focus:border-amber-500 rounded-xl px-3.5 sm:px-4 py-3 text-base sm:text-sm font-mono text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none transition-colors min-h-[44px]"
              />
              {urls.length > 2 && (
                <button
                  type="button"
                  onClick={() => removeUrlField(index)}
                  className="w-11 h-11 text-slate-400 dark:text-slate-500 hover:text-rose-500 active:text-rose-600 flex items-center justify-center transition rounded-xl hover:bg-slate-100 dark:hover:bg-slate-800 shrink-0 touch-manipulation cursor-pointer"
                  title="Hapus URL"
                  aria-label={`Hapus URL #${index + 1}`}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>
          ))}
        </div>

        <div className="flex justify-end pt-1 sm:pt-2">
          <button
            onClick={() => handleCompare()}
            disabled={loading}
            className="w-full sm:w-auto px-6 py-3 rounded-xl bg-amber-500 hover:bg-amber-400 active:bg-amber-600 text-slate-950 text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 shadow-md min-h-[48px] touch-manipulation cursor-pointer"
          >
            {loading ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                <span>Menganalisis Framing Media...</span>
              </>
            ) : (
              <>
                <Scale className="w-4 h-4" />
                <span>Bandingkan Sudut Pandang Media</span>
              </>
            )}
          </button>
        </div>

        {error && (
          <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-xs font-mono text-rose-600 dark:text-rose-300">
            {error}
          </div>
        )}
      </section>

      {/* Loading Skeleton with Shimmer */}
      {loading && (
        <div className="p-6 sm:p-8 rounded-2xl bg-white dark:bg-slate-900/60 border border-slate-200 dark:border-slate-800 space-y-4 shadow-xs shimmer-wrapper">
          <div className="h-6 w-1/3 bg-slate-200 dark:bg-slate-800 rounded" />
          <div className="h-28 w-full bg-slate-100 dark:bg-slate-800/60 rounded-xl" />
          <div className="h-20 w-full bg-slate-100 dark:bg-slate-800/60 rounded-xl" />
        </div>
      )}

      {/* Results View */}
      {result && !loading && (
        <section className="space-y-6 sm:space-y-8 animate-in fade-in">
          {/* Issue Header */}
          <div className="p-4 sm:p-6 rounded-2xl bg-white dark:bg-slate-900/90 border border-slate-200 dark:border-slate-800 space-y-2.5 sm:space-y-3 shadow-sm dark:shadow-xl">
            <span className="text-[11px] sm:text-xs font-mono text-amber-700 dark:text-amber-400 uppercase tracking-wider font-bold">
              Topik Isu Teridentifikasi
            </span>
            <h3 className="font-editorial text-xl sm:text-2xl font-bold text-slate-900 dark:text-slate-100 leading-snug">
              {result.topic}
            </h3>
            <div className="p-3.5 sm:p-4 rounded-xl bg-slate-50 dark:bg-slate-950/80 border border-slate-200 dark:border-slate-800 space-y-1">
              <span className="text-xs font-mono font-semibold text-slate-600 dark:text-slate-400">
                Intisari Objektif Konsensus:
              </span>
              <p className="text-xs sm:text-sm text-slate-800 dark:text-slate-200 leading-relaxed font-sans">
                {result.objective_summary}
              </p>
            </div>
          </div>

          {/* The Media Divergence Spectrum */}
          <div className="p-4 sm:p-8 rounded-2xl bg-white dark:bg-slate-900/80 border border-slate-200 dark:border-slate-800 space-y-5 sm:space-y-6 shadow-sm dark:shadow-xl">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-200 dark:border-slate-800 pb-4">
              <div className="flex items-center gap-2">
                <BarChart2 className="w-5 h-5 text-amber-500" />
                <h4 className="font-editorial text-lg sm:text-xl font-bold text-slate-900 dark:text-slate-100">
                  The Media Divergence Spectrum
                </h4>
              </div>
              <div className="flex flex-wrap items-center gap-3 sm:gap-4 text-xs font-mono">
                <span className="flex items-center gap-1.5 text-rose-700 dark:text-rose-400 font-semibold">
                  <span className="w-2.5 h-2.5 rounded-full bg-rose-500" /> Kritis (-1.0)
                </span>
                <span className="flex items-center gap-1.5 text-slate-600 dark:text-slate-400 font-semibold">
                  <span className="w-2.5 h-2.5 rounded-full bg-slate-500" /> Netral (0.0)
                </span>
                <span className="flex items-center gap-1.5 text-emerald-700 dark:text-emerald-400 font-semibold">
                  <span className="w-2.5 h-2.5 rounded-full bg-emerald-500" /> Positif (+1.0)
                </span>
              </div>
            </div>

            {/* Visual Divergence Bars */}
            <div className="space-y-3.5 sm:space-y-4">
              {result.media_comparisons.map((media, idx) => {
                const clampedScore = Math.max(-1.0, Math.min(1.0, media.sentiment_score || 0))
                const barWidth = Math.abs(clampedScore) * 50
                const isPositive = clampedScore >= 0

                return (
                  <div
                    key={idx}
                    className="p-4 sm:p-5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 space-y-2.5 sm:space-y-3 hover:border-slate-300 dark:hover:border-slate-700 transition"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
                      <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
                        <span className="px-2.5 py-0.5 rounded-md bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 font-bold border border-slate-200 dark:border-slate-700 text-[11px] sm:text-xs">
                          {media.outlet}
                        </span>
                        <span className="text-slate-600 dark:text-slate-400 italic text-[11px] sm:text-xs">
                          Framing: "{media.dominant_frame}"
                        </span>
                      </div>
                      <span className={`font-bold ${isPositive ? 'text-emerald-700 dark:text-emerald-400' : 'text-rose-700 dark:text-rose-400'}`}>
                        Skor: {clampedScore > 0 ? `+${clampedScore.toFixed(2)}` : clampedScore.toFixed(2)} ({media.sentiment})
                      </span>
                    </div>

                    {/* Sentiment Bar Track */}
                    <div className="relative h-4 bg-slate-200 dark:bg-slate-900 rounded-full overflow-hidden border border-slate-300 dark:border-slate-800">
                      <div className="absolute top-0 bottom-0 left-1/2 w-0.5 bg-slate-400 dark:bg-slate-700 z-10" />
                      {isPositive ? (
                        <div
                          className="absolute top-0 bottom-0 left-1/2 bg-emerald-500 rounded-r-full transition-all duration-500 shadow-md"
                          style={{ width: `${barWidth}%` }}
                        />
                      ) : (
                        <div
                          className="absolute top-0 bottom-0 bg-rose-500 rounded-l-full transition-all duration-500 shadow-md"
                          style={{
                            right: '50%',
                            width: `${barWidth}%`,
                          }}
                        />
                      )}
                    </div>

                    {/* Headline and facts */}
                    <div className="pt-1 space-y-2">
                      <p className="font-editorial text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100 flex items-start gap-1.5">
                        <a
                          href={media.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="hover:text-amber-600 dark:hover:text-amber-400 transition flex items-center gap-1.5"
                        >
                          <span>{media.headline}</span>
                          <ExternalLink className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                        </a>
                      </p>
                      {media.selected_facts && media.selected_facts.length > 0 && (
                        <div className="bg-white dark:bg-slate-900/60 p-3 rounded-lg border border-slate-200 dark:border-slate-800">
                          <span className="text-[11px] sm:text-xs font-mono font-semibold text-slate-600 dark:text-slate-400 block mb-1">
                            Fakta yang Disorot Khusus:
                          </span>
                          <ul className="text-xs sm:text-sm text-slate-700 dark:text-slate-300 list-disc list-inside space-y-1">
                            {media.selected_facts.map((fact, fIdx) => (
                              <li key={fIdx}>{fact}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Consensus Facts vs Divergences */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 sm:gap-6">
            {/* Consensus Core */}
            <div className="p-4 sm:p-6 rounded-2xl bg-white dark:bg-slate-900/80 border border-emerald-500/30 space-y-3 sm:space-y-4 shadow-sm dark:shadow-xl">
              <h4 className="font-editorial text-base sm:text-lg font-bold text-emerald-700 dark:text-emerald-400 flex items-center gap-2">
                <CheckCircle2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400 shrink-0" />
                <span>Fakta Konsensus Seluruh Redaksi</span>
              </h4>
              {result.consensus_facts && result.consensus_facts.length > 0 ? (
                <ul className="text-xs sm:text-sm text-slate-700 dark:text-slate-200 space-y-2">
                  {result.consensus_facts.map((f, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <span className="text-emerald-600 dark:text-emerald-400 font-bold shrink-0 mt-0.5">•</span>
                      <span className="leading-relaxed">{f}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400">Tidak ditemukan data konsensus eksplisit.</p>
              )}
            </div>

            {/* Divergence Pointers */}
            <div className="p-4 sm:p-6 rounded-2xl bg-white dark:bg-slate-900/80 border border-amber-500/30 space-y-3 sm:space-y-4 shadow-sm dark:shadow-xl">
              <h4 className="font-editorial text-base sm:text-lg font-bold text-amber-700 dark:text-amber-400 flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400 shrink-0" />
                <span>Titik Perbedaan Narasi Media</span>
              </h4>
              {result.divergences && result.divergences.length > 0 ? (
                <ul className="text-xs sm:text-sm text-slate-700 dark:text-slate-200 space-y-2">
                  {result.divergences.map((d, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <span className="text-amber-600 dark:text-amber-400 font-bold shrink-0 mt-0.5">•</span>
                      <span className="leading-relaxed">{d}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs sm:text-sm text-slate-500 dark:text-slate-400">Tidak ada perbedaan sudut pandang tajam antar-media ini.</p>
              )}
            </div>
          </div>
        </section>
      )}
    </div>
  )
}
