import React, { useState, useEffect } from 'react'
import { Scale, Plus, Trash2, CheckCircle2, AlertTriangle, RefreshCw, BarChart2 } from 'lucide-react'
import type { NewsCompareResponse } from '../types'
import { api } from '../services/api'

interface MediaBiasSpectrumProps {
  apiKey: string
  prefillUrls?: string[]
}

export const MediaBiasSpectrum: React.FC<MediaBiasSpectrumProps> = ({ apiKey, prefillUrls }) => {
  const [urls, setUrls] = useState<string[]>(['', ''])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<NewsCompareResponse | null>(null)

  // Presets for 1-click playground testing
  const presets = [
    {
      title: 'Inflasi Pangan Oktober 2026',
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
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-[#1F2937] pb-4">
        <span className="text-[11px] font-mono text-[#D97706] uppercase tracking-wider font-semibold">
          Analisis Komparatif Multi-Perspektif
        </span>
        <h2 className="font-editorial text-xl lg:text-2xl font-bold text-[#F3F4F6] mt-0.5">
          Media Bias & Sentiment Spectrum
        </h2>
        <p className="text-xs text-[#9CA3AF] mt-1">
          Membedah secara visual bagaimana berbagai redaksi media membingkai (*framing*) peristiwa yang sama dengan skor sentimen objektif.
        </p>
      </div>

      {/* Preset Buttons for Quick 1-Click Testing */}
      <div className="space-y-2">
        <span className="text-xs font-mono text-[#9CA3AF]">Uji Coba Cepat dengan Sampel Kasus Nyata:</span>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {presets.map((p, idx) => (
            <button
              key={idx}
              onClick={() => handleApplyPreset(p.urls)}
              disabled={loading}
              className="p-3 rounded-lg bg-[#111827] hover:bg-[#182032] border border-[#1F2937] hover:border-[#D97706]/40 text-left transition flex items-start justify-between gap-2"
            >
              <div>
                <h4 className="font-editorial text-sm font-bold text-[#F3F4F6]">{p.title}</h4>
                <p className="text-xs text-[#9CA3AF] mt-0.5 font-mono">{p.desc}</p>
              </div>
              <span className="text-xs font-mono text-[#D97706] shrink-0 mt-0.5">Tes Kasus ↗</span>
            </button>
          ))}
        </div>
      </div>

      {/* URL Inputs Form */}
      <section className="p-6 rounded-lg bg-[#111827] border border-[#1F2937] space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-mono text-[#F3F4F6] font-semibold uppercase">
            Tautan Artikel untuk Diaudit (2 - 5 Sumber)
          </h3>
          {urls.length < 5 && (
            <button
              type="button"
              onClick={addUrlField}
              className="text-xs font-mono text-[#D97706] hover:text-[#F59E0B] flex items-center gap-1 transition"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Tambah URL</span>
            </button>
          )}
        </div>

        <div className="space-y-2.5">
          {urls.map((url, index) => (
            <div key={index} className="flex items-center gap-2">
              <span className="text-xs font-mono text-[#6B7280] w-6 text-right">
                {index + 1}.
              </span>
              <input
                type="url"
                value={url}
                onChange={(e) => handleUrlChange(index, e.target.value)}
                placeholder={`URL Berita Media #${index + 1}`}
                className="flex-1 bg-[#080C14] border border-[#1F2937] focus:border-[#D97706] rounded-md px-3.5 py-2 text-xs font-mono text-[#F3F4F6] placeholder:text-[#4B5563] focus:outline-none"
              />
              {urls.length > 2 && (
                <button
                  type="button"
                  onClick={() => removeUrlField(index)}
                  className="text-[#6B7280] hover:text-[#E11D48] p-1.5 transition"
                  title="Hapus URL"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>
          ))}
        </div>

        <div className="flex justify-end pt-2">
          <button
            onClick={() => handleCompare()}
            disabled={loading}
            className="px-6 py-2.5 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition disabled:opacity-50 flex items-center gap-2"
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
          <div className="p-3 rounded-md bg-[#E11D48]/10 border border-[#E11D48]/30 text-xs font-mono text-[#FCA5A5]">
            {error}
          </div>
        )}
      </section>

      {/* Loading Skeleton */}
      {loading && (
        <div className="p-8 rounded-lg bg-[#111827] border border-[#1F2937] space-y-4 animate-pulse">
          <div className="h-6 w-1/3 bg-[#1F2937] rounded" />
          <div className="h-28 w-full bg-[#182032] rounded" />
          <div className="h-20 w-full bg-[#182032] rounded" />
        </div>
      )}

      {/* Results View */}
      {result && !loading && (
        <section className="space-y-6">
          {/* Issue Header */}
          <div className="p-5 rounded-lg bg-[#182032] border border-[#1E293B]">
            <span className="text-[11px] font-mono text-[#D97706] uppercase tracking-wider font-semibold">
              Topik Isu Teridentifikasi
            </span>
            <h3 className="font-editorial text-xl font-bold text-[#F3F4F6] mt-0.5">
              {result.topic}
            </h3>
            <p className="text-xs text-[#9CA3AF] mt-2 leading-relaxed bg-[#0E131F] p-3 rounded border border-[#1F2937]">
              <b>Intisari Objektif Konsensus:</b> {result.objective_summary}
            </p>
          </div>

          {/* The Media Divergence Spectrum (SVG Bar Chart) */}
          <div className="p-6 rounded-lg bg-[#111827] border border-[#1F2937] space-y-4">
            <div className="flex items-center justify-between border-b border-[#1F2937] pb-3">
              <div className="flex items-center gap-2">
                <BarChart2 className="w-4 h-4 text-[#D97706]" />
                <h4 className="font-editorial text-base font-bold text-[#F3F4F6]">
                  The Media Divergence Spectrum
                </h4>
              </div>
              <div className="flex items-center gap-4 text-xs font-mono text-[#9CA3AF]">
                <span className="flex items-center gap-1">
                  <span className="w-2.5 h-2.5 rounded-full bg-[#E11D48]" /> Kritis / Negatif (-1.0)
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2.5 h-2.5 rounded-full bg-[#9CA3AF]" /> Netral (0.0)
                </span>
                <span className="flex items-center gap-1">
                  <span className="w-2.5 h-2.5 rounded-full bg-[#059669]" /> Positif (+1.0)
                </span>
              </div>
            </div>

            {/* Visual Divergence Bars */}
            <div className="space-y-4 pt-2">
              {result.media_comparisons.map((media, idx) => {
                // sentiment_score is between -1.0 and 1.0.
                // Map to percentage: 0% is -1.0, 50% is 0.0, 100% is +1.0.
                const clampedScore = Math.max(-1.0, Math.min(1.0, media.sentiment_score || 0))
                const barWidth = Math.abs(clampedScore) * 50 // width from center
                const isPositive = clampedScore >= 0

                return (
                  <div key={idx} className="p-4 rounded-md bg-[#080C14] border border-[#1F2937] space-y-2">
                    <div className="flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
                      <div className="flex items-center gap-2">
                        <span className="px-2 py-0.5 rounded bg-[#182032] text-[#93C5FD] font-semibold border border-[#1E293B]">
                          {media.outlet}
                        </span>
                        <span className="text-[#9CA3AF] italic">
                          Framing: "{media.dominant_frame}"
                        </span>
                      </div>
                      <span className={`font-semibold ${isPositive ? 'text-[#10B981]' : 'text-[#F43F5E]'}`}>
                        Skor: {clampedScore > 0 ? `+${clampedScore.toFixed(2)}` : clampedScore.toFixed(2)} ({media.sentiment})
                      </span>
                    </div>

                    {/* Sentiment Bar Track */}
                    <div className="relative h-4 bg-[#111827] rounded-full overflow-hidden border border-[#1F2937]">
                      {/* Center zero mark */}
                      <div className="absolute top-0 bottom-0 left-1/2 w-0.5 bg-[#374151] z-10" />

                      {/* Bar fill from center */}
                      {isPositive ? (
                        <div
                          className="absolute top-0 bottom-0 left-1/2 bg-[#059669] rounded-r-full transition-all duration-500"
                          style={{ width: `${barWidth}%` }}
                        />
                      ) : (
                        <div
                          className="absolute top-0 bottom-0 bg-[#E11D48] rounded-l-full transition-all duration-500"
                          style={{
                            right: '50%',
                            width: `${barWidth}%`,
                          }}
                        />
                      )}
                    </div>

                    {/* Headline and facts */}
                    <div className="pt-1 space-y-1">
                      <p className="font-editorial text-xs font-semibold text-[#F3F4F6]">
                        {media.headline}
                      </p>
                      {media.selected_facts && media.selected_facts.length > 0 && (
                        <ul className="text-[11px] text-[#9CA3AF] list-disc list-inside space-y-0.5">
                          {media.selected_facts.map((fact, fIdx) => (
                            <li key={fIdx}>{fact}</li>
                          ))}
                        </ul>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Consensus Facts vs Divergences */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Consensus Core */}
            <div className="p-5 rounded-lg bg-[#111827] border border-[#10B981]/30 space-y-3">
              <h4 className="font-editorial text-sm font-bold text-[#10B981] flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 text-[#10B981]" />
                Fakta Konsensus yang Disepakati Bersama
              </h4>
              {result.consensus_facts && result.consensus_facts.length > 0 ? (
                <ul className="text-xs text-[#D1D5DB] space-y-2">
                  {result.consensus_facts.map((f, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <span className="text-[#10B981] font-bold">•</span>
                      <span>{f}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs text-[#9CA3AF]">Tidak ditemukan data konsensus eksplisit.</p>
              )}
            </div>

            {/* Divergence Pointers */}
            <div className="p-5 rounded-lg bg-[#111827] border border-[#F59E0B]/30 space-y-3">
              <h4 className="font-editorial text-sm font-bold text-[#F59E0B] flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 text-[#F59E0B]" />
                Titik Perbedaan Narasi & Framing Media
              </h4>
              {result.divergences && result.divergences.length > 0 ? (
                <ul className="text-xs text-[#D1D5DB] space-y-2">
                  {result.divergences.map((d, i) => (
                    <li key={i} className="flex items-start gap-2">
                      <span className="text-[#F59E0B] font-bold">•</span>
                      <span>{d}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-xs text-[#9CA3AF]">Tidak ada perbedaan sudut pandang tajam antar-media ini.</p>
              )}
            </div>
          </div>
        </section>
      )}
    </div>
  )
}
