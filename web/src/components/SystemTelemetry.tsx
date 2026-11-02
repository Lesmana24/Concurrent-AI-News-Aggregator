import React, { useState, useEffect } from 'react'
import { RefreshCw, Server, Database, CheckCircle2, Zap } from 'lucide-react'
import type { UserMe } from '../types'
import { api } from '../services/api'

interface SystemTelemetryProps {
  apiKey: string
  user: UserMe | null
  onRefreshUser: () => void
}

export const SystemTelemetry: React.FC<SystemTelemetryProps> = ({ apiKey, user, onRefreshUser }) => {
  const [crawlerStatus, setCrawlerStatus] = useState<any>(null)
  const [triggeringCrawl, setTriggeringCrawl] = useState(false)
  const [triggerMessage, setTriggerMessage] = useState<string | null>(null)

  const fetchStatus = async () => {
    try {
      const data = await api.getCrawlerStatus()
      setCrawlerStatus(data)
    } catch {
      // ignore
    }
  }

  useEffect(() => {
    fetchStatus()
  }, [])

  const handleTriggerCrawler = async () => {
    setTriggeringCrawl(true)
    setTriggerMessage(null)
    try {
      const res = await api.triggerCrawler(apiKey)
      setTriggerMessage(res.message || 'Penyedotan berita selesai dijalankan')
      fetchStatus()
      onRefreshUser()
    } catch (err: unknown) {
      setTriggerMessage(err instanceof Error ? err.message : 'Gagal memicu crawler')
    } finally {
      setTriggeringCrawl(false)
    }
  }

  return (
    <div className="space-y-8 sm:space-y-10">
      {/* Header */}
      <div className="border-b border-slate-800 pb-4 sm:pb-5">
        <span className="text-[11px] sm:text-xs font-mono text-amber-400 uppercase tracking-wider font-bold">
          Infrastruktur & Monitoring
        </span>
        <h2 className="font-editorial text-xl sm:text-2xl lg:text-3xl font-bold text-slate-100 mt-1">
          Engine Telemetry & Controls
        </h2>
        <p className="text-xs sm:text-sm text-slate-300 mt-1 font-sans leading-relaxed max-w-2xl">
          Pantau status kesehatan Harvester RSS otonom, sisa kuota API Key sandbox, dan keandalan Multi-Provider AI Fallback Chain (Groq $\rightarrow$ NVIDIA NIM $\rightarrow$ Gemini $\rightarrow$ Local Notice).
        </p>
      </div>

      {/* Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
        {/* Card 1: API Quota */}
        <div className="p-5 sm:p-6 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-3.5 shadow-xl">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono font-semibold text-slate-400 uppercase tracking-wider">
              Sisa Kuota Token
            </span>
            <Database className="w-5 h-5 text-amber-500" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl sm:text-4xl font-bold text-amber-400 font-mono">
              {user ? user.quota : 50}
            </span>
            <span className="text-xs font-mono text-slate-400">/ 50 requests</span>
          </div>
          <div className="w-full bg-slate-950 h-2.5 rounded-full overflow-hidden border border-slate-800">
            <div
              className="bg-amber-500 h-full rounded-full transition-all"
              style={{ width: `${Math.min(100, ((user ? user.quota : 50) / 50) * 100)}%` }}
            />
          </div>
          <p className="text-xs text-slate-400 font-mono truncate">
            Active Key: <code className="text-amber-400 font-bold">{apiKey}</code>
          </p>
        </div>

        {/* Card 2: News Harvester Daemon */}
        <div className="p-5 sm:p-6 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-3.5 shadow-xl">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono font-semibold text-slate-400 uppercase tracking-wider">
              Autonomous Harvester
            </span>
            <Server className="w-5 h-5 text-emerald-400" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl sm:text-4xl font-bold text-emerald-400 font-mono">
              {crawlerStatus ? crawlerStatus.active_feeds : 8}
            </span>
            <span className="text-xs font-mono text-slate-400">Portal Media Aktif</span>
          </div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-300">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
            <span>Siklus Otomatis Tiap 15 Menit</span>
          </div>
          <p className="text-xs text-slate-400 font-mono">
            Antara, CNBC, CNN, Republika (RSS & Atom)
          </p>
        </div>

        {/* Card 3: AI Fallback Chain */}
        <div className="p-5 sm:p-6 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-3.5 shadow-xl sm:col-span-2 lg:col-span-1">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono font-semibold text-slate-400 uppercase tracking-wider">
              AI Resilience Architecture
            </span>
            <Zap className="w-5 h-5 text-amber-500" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl sm:text-4xl font-bold text-slate-100 font-mono">
              4-Tier
            </span>
            <span className="text-xs font-mono text-slate-400">Zero-Downtime Chain</span>
          </div>
          <div className="space-y-1 text-xs font-mono text-slate-300">
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
              <span>Tier 1: Groq (Llama-3.3 / Qwen)</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
              <span>Tier 2: NVIDIA NIM (Mistral-7B)</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
              <span>Tier 3: Gemini 2.0 Flash</span>
            </div>
          </div>
        </div>
      </div>

      {/* Manual Control Action (Min 44px Touch Target) */}
      <section className="p-5 sm:p-7 rounded-2xl bg-slate-900/80 border border-slate-800 space-y-4 shadow-xl">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-100">
              Picu Siklus Harvester Berita Secara Manual
            </h3>
            <p className="text-xs sm:text-sm text-slate-300 font-sans mt-0.5 leading-relaxed">
              Memerintahkan engine untuk langsung menyedot artikel terkini dari seluruh RSS feed media nasional tanpa menunggu jadwal 15 menit.
            </p>
          </div>

          <button
            onClick={handleTriggerCrawler}
            disabled={triggeringCrawl}
            className="w-full sm:w-auto px-6 py-3 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 shrink-0 shadow-lg min-h-[44px] touch-manipulation"
          >
            {triggeringCrawl ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                <span>Menyedot Berita...</span>
              </>
            ) : (
              <>
                <RefreshCw className="w-4 h-4" />
                <span>Picu Crawl Sekarang</span>
              </>
            )}
          </button>
        </div>

        {triggerMessage && (
          <div className="p-4 rounded-xl bg-slate-950 border border-slate-700 text-xs sm:text-sm font-mono text-amber-300 flex items-center gap-2 animate-in fade-in">
            <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
            <span>{triggerMessage}</span>
          </div>
        )}
      </section>
    </div>
  )
}
