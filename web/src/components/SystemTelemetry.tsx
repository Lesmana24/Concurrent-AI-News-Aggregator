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

  const isGuest = Boolean(user?.is_guest ?? (!apiKey || apiKey.startsWith('guest_') || apiKey === 'test-key-123'))
  const dailyLimit = user?.daily_limit ?? (isGuest ? 5 : (user ? 50 : 5))
  const currentQuota = user ? user.quota : dailyLimit
  const quotaPercent = Math.min(100, Math.round((currentQuota / (dailyLimit || 1)) * 100))

  const displayKey = apiKey && apiKey !== 'undefined' && apiKey !== 'null'
    ? (isGuest ? `${apiKey.slice(0, 16)}... (Guest Pass)` : apiKey)
    : 'Guest Pass (Auto Init)'

  return (
    <div className="space-y-8 sm:space-y-10">
      {/* Header */}
      <div className="border-b border-slate-200 dark:border-slate-800 pb-4 sm:pb-5">
        <span className="text-[11px] sm:text-xs font-mono text-amber-700 dark:text-amber-400 uppercase tracking-wider font-bold">
          Infrastruktur & Monitoring
        </span>
        <h2 className="font-editorial text-xl sm:text-2xl lg:text-3xl font-bold text-slate-900 dark:text-slate-100 mt-1">
          Engine Telemetry & Controls
        </h2>
        <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 mt-1 font-sans leading-relaxed max-w-2xl">
          Pantau status kesehatan Harvester berita otonom, alokasi kuota komputasi, dan keandalan arsitektur Multi-Tier High-Availability Neural Network.
        </p>
      </div>

      {/* Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
        {/* Card 1: API Quota */}
        <div className="p-5 sm:p-6 rounded-2xl bg-white dark:bg-slate-900/80 border border-slate-200 dark:border-slate-800 space-y-3.5 shadow-sm dark:shadow-xl transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              {isGuest ? 'Sisa Kuota Tamu (Harian)' : 'Sisa Kuota Token'}
            </span>
            <Database className="w-5 h-5 text-amber-500" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl sm:text-4xl font-bold text-amber-700 dark:text-amber-400 font-mono">
              {currentQuota}
            </span>
            <span className="text-xs font-mono text-slate-500 dark:text-slate-400">/ {dailyLimit} requests</span>
          </div>
          <div className="w-full bg-slate-100 dark:bg-slate-950 h-2.5 rounded-full overflow-hidden border border-slate-200 dark:border-slate-800">
            <div
              className="bg-amber-500 h-full rounded-full transition-all"
              style={{ width: `${quotaPercent}%` }}
            />
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 font-mono truncate">
            Active Key: <code className="text-amber-700 dark:text-amber-400 font-bold">{displayKey}</code>
          </p>
        </div>

        {/* Card 2: News Harvester Daemon */}
        <div className="p-5 sm:p-6 rounded-2xl bg-white dark:bg-slate-900/80 border border-slate-200 dark:border-slate-800 space-y-3.5 shadow-sm dark:shadow-xl transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Autonomous Harvester
            </span>
            <Server className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl sm:text-4xl font-bold text-emerald-700 dark:text-emerald-400 font-mono">
              {crawlerStatus ? crawlerStatus.active_feeds : 8}
            </span>
            <span className="text-xs font-mono text-slate-500 dark:text-slate-400">Portal Media Aktif</span>
          </div>
          <div className="flex items-center gap-2 text-xs font-mono text-slate-700 dark:text-slate-300">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse" />
            <span>Siklus Otomatis Tiap 15 Menit</span>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 font-mono">
            Antara, CNBC, CNN, Republika (RSS & Atom)
          </p>
        </div>

        {/* Card 3: AI Fallback Chain */}
        <div className="p-5 sm:p-6 rounded-2xl bg-white dark:bg-slate-900/80 border border-slate-200 dark:border-slate-800 space-y-3.5 shadow-sm dark:shadow-xl sm:col-span-2 lg:col-span-1 transition-colors">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              AI Resilience Architecture
            </span>
            <Zap className="w-5 h-5 text-amber-500" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl sm:text-4xl font-bold text-slate-900 dark:text-slate-100 font-mono">
              4-Tier
            </span>
            <span className="text-xs font-mono text-slate-500 dark:text-slate-400">Zero-Downtime Chain</span>
          </div>
          <div className="space-y-1 text-xs font-mono text-slate-700 dark:text-slate-300">
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 dark:text-emerald-400" />
              <span>Tier 1: Ultra-Fast Neural Engine</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 dark:text-emerald-400" />
              <span>Tier 2: High-Capacity Inference Core</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 dark:text-emerald-400" />
              <span>Tier 3: Multimodal Deep Synthesis</span>
            </div>
          </div>
        </div>
      </div>

      {/* Manual Control Action (Min 44px Touch Target) */}
      <section className="p-5 sm:p-7 rounded-2xl bg-white dark:bg-slate-900/80 border border-slate-200 dark:border-slate-800 space-y-4 shadow-sm dark:shadow-xl transition-colors">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-900 dark:text-slate-100">
              Picu Siklus Harvester Berita Secara Manual
            </h3>
            <p className="text-xs sm:text-sm text-slate-600 dark:text-slate-300 font-sans mt-0.5 leading-relaxed">
              Memerintahkan engine untuk langsung menyedot artikel terkini dari seluruh RSS feed media nasional tanpa menunggu jadwal 15 menit.
            </p>
          </div>

          <button
            onClick={handleTriggerCrawler}
            disabled={triggeringCrawl}
            className="w-full sm:w-auto px-6 py-3 rounded-xl bg-amber-500 hover:bg-amber-400 active:bg-amber-600 text-slate-950 text-xs sm:text-sm font-mono font-bold transition disabled:opacity-50 flex items-center justify-center gap-2 shrink-0 shadow-md min-h-[44px] touch-manipulation cursor-pointer"
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
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-700 text-xs sm:text-sm font-mono text-amber-800 dark:text-amber-300 flex items-center gap-2 animate-in fade-in">
            <CheckCircle2 className="w-4 h-4 text-emerald-500 dark:text-emerald-400 shrink-0" />
            <span>{triggerMessage}</span>
          </div>
        )}
      </section>
    </div>
  )
}
