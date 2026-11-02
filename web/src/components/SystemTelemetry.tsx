import React, { useState, useEffect } from 'react'
import { RefreshCw, Server, Cpu, Database, CheckCircle2 } from 'lucide-react'
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
    <div className="space-y-8">
      {/* Header */}
      <div className="border-b border-[#1F2937] pb-4">
        <span className="text-[11px] font-mono text-[#D97706] uppercase tracking-wider font-semibold">
          Infrastruktur & Monitoring
        </span>
        <h2 className="font-editorial text-xl lg:text-2xl font-bold text-[#F3F4F6] mt-0.5">
          Engine Telemetry & Controls
        </h2>
        <p className="text-xs text-[#9CA3AF] mt-1">
          Pantau status kesehatan Harvester RSS, sisa kuota API Key, dan keandalan Multi-Provider AI Fallback Chain.
        </p>
      </div>

      {/* Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {/* Card 1: API Quota */}
        <div className="p-5 rounded-lg bg-[#111827] border border-[#1F2937] space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono text-[#9CA3AF]">Sisa Kuota Token</span>
            <Database className="w-4 h-4 text-[#D97706]" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl font-bold text-[#F59E0B] font-mono">
              {user ? user.quota : 50}
            </span>
            <span className="text-xs font-mono text-[#9CA3AF]">/ 50 requests</span>
          </div>
          <div className="w-full bg-[#080C14] h-2 rounded-full overflow-hidden border border-[#1F2937]">
            <div
              className="bg-[#D97706] h-full rounded-full transition-all"
              style={{ width: `${Math.min(100, ((user ? user.quota : 50) / 50) * 100)}%` }}
            />
          </div>
          <p className="text-[11px] text-[#6B7280] font-mono">
            Key: <code className="text-[#9CA3AF]">{apiKey}</code>
          </p>
        </div>

        {/* Card 2: News Harvester Daemon */}
        <div className="p-5 rounded-lg bg-[#111827] border border-[#1F2937] space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono text-[#9CA3AF]">Autonomous Harvester</span>
            <Server className="w-4 h-4 text-[#10B981]" />
          </div>
          <div className="flex items-baseline gap-2">
            <span className="font-editorial text-3xl font-bold text-[#10B981] font-mono">
              {crawlerStatus ? crawlerStatus.active_feeds : 8}
            </span>
            <span className="text-xs font-mono text-[#9CA3AF]">RSS Feeds Aktif</span>
          </div>
          <div className="flex items-center gap-2 text-xs font-mono text-[#9CA3AF]">
            <span className="w-2 h-2 rounded-full bg-[#10B981] animate-pulse" />
            <span>Siklus Otomatis Tiap 15 Menit</span>
          </div>
          <p className="text-[11px] text-[#6B7280] font-mono">
            Antara • CNBC • CNN • Republika
          </p>
        </div>

        {/* Card 3: Multi-Provider AI Fallback */}
        <div className="p-5 rounded-lg bg-[#111827] border border-[#1F2937] space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-mono text-[#9CA3AF]">Multi-Provider AI Chain</span>
            <Cpu className="w-4 h-4 text-[#60A5FA]" />
          </div>
          <div className="space-y-1.5 pt-1">
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-[#D1D5DB]">1. Groq (Qwen/Llama)</span>
              <span className="text-[#10B981]">Primary</span>
            </div>
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-[#D1D5DB]">2. NVIDIA NIM</span>
              <span className="text-[#9CA3AF]">Secondary</span>
            </div>
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-[#D1D5DB]">3. Google AI Studio</span>
              <span className="text-[#9CA3AF]">Tertiary</span>
            </div>
          </div>
        </div>
      </div>

      {/* Manual Actions Panel */}
      <section className="p-6 rounded-lg bg-[#111827] border border-[#1F2937] space-y-4">
        <div>
          <h3 className="font-editorial text-base font-bold text-[#F3F4F6]">
            Pemicu Aksi Manual Latar Belakang
          </h3>
          <p className="text-xs text-[#9CA3AF] mt-0.5 font-sans">
            Gunakan kontrol ini untuk memicu siklus penyedotan berita atau perhitungan kluster secara langsung tanpa menunggu waktu cron terjadwal.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <button
            onClick={handleTriggerCrawler}
            disabled={triggeringCrawl}
            className="px-4 py-2.5 rounded-md bg-[#182032] hover:bg-[#1E293B] border border-[#2563EB]/40 text-[#93C5FD] text-xs font-mono font-medium transition disabled:opacity-50 flex items-center gap-2"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${triggeringCrawl ? 'animate-spin' : ''}`} />
            <span>{triggeringCrawl ? 'Sedang Menyedot RSS...' : 'Trigger Harvest News Feeds'}</span>
          </button>
        </div>

        {triggerMessage && (
          <div className="p-3 rounded-md bg-[#080C14] border border-[#1F2937] text-xs font-mono text-[#F59E0B] flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 text-[#10B981] shrink-0" />
            <span>{triggerMessage}</span>
          </div>
        )}
      </section>
    </div>
  )
}
