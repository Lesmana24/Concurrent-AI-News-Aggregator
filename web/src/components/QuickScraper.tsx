import React, { useState, useEffect } from 'react'
import { Radio, Send, ExternalLink, RefreshCw, CheckCircle2, AlertCircle } from 'lucide-react'
import type { Article } from '../types'
import { api } from '../services/api'

interface QuickScraperProps {
  apiKey: string
  onSelectForComparison: (urls: string[]) => void
}

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
      // Refresh feed in background
      fetchWire()
    } catch (err: unknown) {
      setIngestError(err instanceof Error ? err.message : 'Gagal mengekstrak berita dari URL')
    } finally {
      setIngesting(false)
    }
  }

  return (
    <div className="space-y-8">
      {/* 1. Quick Scraper Sandbox */}
      <section className="p-6 rounded-lg bg-[#111827] border border-[#1F2937] space-y-4">
        <div>
          <span className="text-[11px] font-mono text-[#D97706] uppercase tracking-wider font-semibold">
            Sandbox Ekstraksi Berita Instan
          </span>
          <h2 className="font-editorial text-xl font-bold text-[#F3F4F6] mt-0.5">
            Quick URL Scraper & AI Summarizer
          </h2>
          <p className="text-xs text-[#9CA3AF] mt-1">
            Tempelkan tautan artikel berita apa saja dari portal media nasional untuk menguji scraper cerdas dan ringkasan AI seketika.
          </p>
        </div>

        <form onSubmit={handleIngestURL} className="flex flex-col sm:flex-row gap-2">
          <input
            type="url"
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            placeholder="https://www.antaranews.com/berita/... atau cnbcindonesia.com/..."
            required
            className="flex-1 bg-[#080C14] border border-[#1F2937] focus:border-[#D97706] rounded-md px-3.5 py-2.5 text-xs font-mono text-[#F3F4F6] placeholder:text-[#4B5563] focus:outline-none"
          />
          <button
            type="submit"
            disabled={ingesting || !urlInput.trim()}
            className="px-5 py-2.5 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition disabled:opacity-50 flex items-center justify-center gap-2 shrink-0"
          >
            {ingesting ? (
              <>
                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                <span>Ingesting...</span>
              </>
            ) : (
              <>
                <Send className="w-3.5 h-3.5" />
                <span>Ingest URL</span>
              </>
            )}
          </button>
        </form>

        {/* Ingest Error State */}
        {ingestError && (
          <div className="p-3 rounded-md bg-[#E11D48]/10 border border-[#E11D48]/30 flex items-center gap-2 text-xs font-mono text-[#FCA5A5]">
            <AlertCircle className="w-4 h-4 text-[#E11D48] shrink-0" />
            <span>{ingestError}</span>
          </div>
        )}

        {/* Ingested Result Card */}
        {ingestedResult && (
          <div className="p-4 rounded-md bg-[#080C14] border border-[#10B981]/30 space-y-2">
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-[#10B981] flex items-center gap-1.5 font-semibold">
                <CheckCircle2 className="w-4 h-4" /> Ekstraksi Berhasil
              </span>
              <span className="px-2 py-0.5 rounded bg-[#182032] text-[#93C5FD] border border-[#1E293B]">
                {ingestedResult.source || 'Media Terverifikasi'}
              </span>
            </div>
            <h4 className="font-editorial text-base font-bold text-[#F3F4F6] leading-snug">
              {ingestedResult.title}
            </h4>
            <p className="text-xs text-[#D1D5DB] leading-relaxed bg-[#111827] p-3 rounded border border-[#1F2937]">
              {ingestedResult.summary}
            </p>
            <div className="flex justify-end pt-1">
              <a
                href={ingestedResult.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs font-mono text-[#9CA3AF] hover:text-[#F3F4F6] flex items-center gap-1 transition"
              >
                <span>Lihat Artikel Asli</span>
                <ExternalLink className="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
        )}
      </section>

      {/* 2. Live Media Wire Feed */}
      <section className="space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-[#1F2937] pb-3">
          <div>
            <div className="flex items-center gap-2">
              <Radio className="w-5 h-5 text-[#D97706]" />
              <h3 className="font-editorial text-xl font-bold text-[#F3F4F6]">
                Live News Wire Feed
              </h3>
            </div>
            <p className="text-xs text-[#9CA3AF] font-sans mt-0.5">
              Arsip berita yang dikumpulkan otomatis oleh Harvester latar belakang secara mandiri.
            </p>
          </div>

          <button
            onClick={() => fetchWire()}
            disabled={loadingWire}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#111827] hover:bg-[#182032] border border-[#1F2937] text-xs font-mono text-[#F3F4F6] transition self-start sm:self-auto"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-[#9CA3AF] ${loadingWire ? 'animate-spin' : ''}`} />
            <span>Refresh Wire</span>
          </button>
        </div>

        {/* Source Filter Pills */}
        <div className="flex flex-wrap items-center gap-1.5">
          {sources.map((src) => (
            <button
              key={src.value}
              onClick={() => setSelectedSource(src.value)}
              className={`px-3 py-1 rounded text-xs font-mono transition ${
                selectedSource === src.value
                  ? 'bg-[#D97706] text-white font-semibold'
                  : 'bg-[#111827] text-[#9CA3AF] hover:text-[#F3F4F6] border border-[#1F2937]'
              }`}
            >
              {src.label}
            </button>
          ))}
        </div>

        {/* Wire Articles List */}
        {loadingWire && (
          <div className="space-y-3">
            {[1, 2, 3, 4].map((n) => (
              <div key={n} className="p-4 rounded-lg bg-[#111827] border border-[#1F2937] space-y-2 animate-pulse">
                <div className="h-4 w-32 bg-[#1F2937] rounded" />
                <div className="h-5 w-3/4 bg-[#1F2937] rounded" />
                <div className="h-10 w-full bg-[#182032] rounded" />
              </div>
            ))}
          </div>
        )}

        {!loadingWire && wireError && (
          <div className="p-4 rounded-lg bg-[#E11D48]/10 border border-[#E11D48]/30 text-xs font-mono text-[#FCA5A5]">
            {wireError}
          </div>
        )}

        {!loadingWire && !wireError && articles.length === 0 && (
          <div className="p-8 text-center rounded-lg bg-[#111827] border border-[#1F2937] text-xs font-mono text-[#9CA3AF]">
            Belum ada artikel yang sesuai dengan filter ini.
          </div>
        )}

        {!loadingWire && !wireError && articles.length > 0 && (
          <div className="space-y-3">
            {articles.map((art) => (
              <article
                key={art.id}
                className="p-4 rounded-lg bg-[#111827] hover:bg-[#151D2E] border border-[#1F2937] hover:border-[#374151] transition space-y-2"
              >
                <div className="flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
                  <span className="px-2 py-0.5 rounded bg-[#182032] text-[#93C5FD] border border-[#1E293B]">
                    {art.source || 'Media Nasional'}
                  </span>
                  <span className="text-[#6B7280]">
                    {art.created_at ? new Date(art.created_at).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' }) : 'Baru saja'}
                  </span>
                </div>

                <h4 className="font-editorial text-base font-bold text-[#F3F4F6] leading-snug">
                  <a
                    href={art.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-[#F59E0B] transition flex items-center gap-1.5"
                  >
                    <span>{art.title}</span>
                    <ExternalLink className="w-3.5 h-3.5 text-[#6B7280] shrink-0" />
                  </a>
                </h4>

                <p className="text-xs text-[#9CA3AF] leading-relaxed line-clamp-2">
                  {art.summary}
                </p>

                <div className="flex justify-end pt-1">
                  <button
                    onClick={() => onSelectForComparison([art.url])}
                    className="text-xs font-mono text-[#D97706] hover:text-[#F59E0B] transition"
                  >
                    + Masukkan ke Spektrum Bias
                  </button>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
