import React from 'react'
import {
  Key,
  CheckCircle2,
  Search,
  Download,
  Bookmark,
} from 'lucide-react'
import type { UserMe } from '../types'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'

interface HeaderProps {
  user: UserMe | null
  apiKey: string
  isOnline: boolean
  watchlistCount: number
  theme: ThemeMode
  onThemeChange: (theme: ThemeMode) => void
  onOpenCommandPalette: () => void
  onOpenApiKeyModal: () => void
  onOpenExportModal: () => void
  onOpenWatchlist: () => void
}

export const Header: React.FC<HeaderProps> = ({
  user,
  isOnline,
  watchlistCount,
  theme,
  onThemeChange,
  onOpenCommandPalette,
  onOpenApiKeyModal,
  onOpenExportModal,
  onOpenWatchlist,
}) => {
  return (
    <header className="border-b border-slate-800 bg-slate-950/90 backdrop-blur-md sticky top-0 z-40 px-3 sm:px-6 lg:px-8 py-2.5 sm:py-3 flex items-center justify-between gap-2 sm:gap-4">
      {/* Brand & Editorial Title with Official Telegram Bot Avatar */}
      <div className="flex items-center gap-2.5 sm:gap-3 shrink-0">
        <div className="relative shrink-0">
          <img
            src="/logo.jpg"
            alt="The Dispatch • @PengumpulBeritaBot Logo"
            className="w-8 h-8 sm:w-9 sm:h-9 rounded-xl object-cover border border-amber-500/40 shadow-md ring-1 ring-amber-500/20"
            onError={(e) => {
              const target = e.target as HTMLElement
              target.style.display = 'none'
              const parent = target.parentElement
              if (parent) {
                const fallback = document.createElement('div')
                fallback.className = 'w-8 h-8 sm:w-9 sm:h-9 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-center text-amber-400 font-editorial font-bold text-lg sm:text-xl'
                fallback.innerText = 'D'
                parent.appendChild(fallback)
              }
            }}
          />
        </div>
        <div>
          <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
            <span className="font-editorial text-base sm:text-xl font-bold tracking-tight text-slate-100">
              THE DISPATCH
            </span>
            <a
              href="https://t.me/PengumpulBeritaBot"
              target="_blank"
              rel="noopener noreferrer"
              className="text-[10px] uppercase font-mono px-2 py-0.5 rounded-full bg-blue-500/15 text-blue-400 border border-blue-500/30 hover:bg-blue-500/25 transition flex items-center gap-1 font-bold shadow-sm"
              title="Buka Bot Terverifikasi di Telegram"
            >
              <CheckCircle2 className="w-3 h-3 text-blue-400" />
              <span>@PengumpulBeritaBot</span>
            </a>
          </div>
          <p className="text-xs text-slate-400 font-sans hidden md:block">
            Newsroom Intelligence & Multi-Perspective Radar Terminal
          </p>
        </div>
      </div>

      {/* Central Search / Command Palette Trigger (Desktop / Tablet) */}
      <button
        onClick={onOpenCommandPalette}
        className="hidden md:flex items-center gap-3 px-3.5 py-1.5 rounded-xl bg-slate-900 hover:bg-slate-850 active:bg-slate-800 border border-slate-700/80 hover:border-amber-500/50 transition text-xs font-mono text-slate-400 max-w-sm w-full mx-4 shadow-inner"
        title="Buka Command Palette (Ctrl+K / Cmd+K)"
      >
        <Search className="w-4 h-4 text-amber-500" />
        <span className="flex-1 text-left truncate">Cari berita atau perintah...</span>
        <span className="flex items-center gap-1 text-[10px] bg-slate-800 text-slate-300 px-1.5 py-0.5 rounded border border-slate-700">
          <kbd>Ctrl+K</kbd>
        </span>
      </button>

      {/* Action Controls & Telemetry Status (Mobile & Touch Optimized) */}
      <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
        {/* Mobile Search Button */}
        <button
          onClick={onOpenCommandPalette}
          className="md:hidden p-2 rounded-lg bg-slate-900 border border-slate-800 text-slate-300 hover:text-amber-400 min-h-[40px] min-w-[40px] flex items-center justify-center transition"
          title="Buka Command Palette"
        >
          <Search className="w-4 h-4" />
        </button>

        {/* Export Briefing Trigger */}
        <button
          onClick={onOpenExportModal}
          className="flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-lg bg-slate-900 hover:bg-slate-850 active:bg-slate-800 border border-slate-700/80 hover:border-amber-500/50 transition text-xs font-mono text-slate-200 min-h-[40px] touch-manipulation shadow-sm"
          title="Ekspor Executive Briefing (Markdown / CSV)"
        >
          <Download className="w-3.5 h-3.5 text-amber-400 shrink-0" />
          <span className="hidden sm:inline">Export</span>
        </button>

        {/* Watchlist Drawer Trigger */}
        <button
          onClick={onOpenWatchlist}
          className="relative flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-lg bg-slate-900 hover:bg-slate-850 active:bg-slate-800 border border-slate-700/80 hover:border-amber-500/50 transition text-xs font-mono text-slate-200 min-h-[40px] touch-manipulation shadow-sm"
          title="Buka Saved Watchlist"
        >
          <Bookmark className="w-3.5 h-3.5 text-amber-400 shrink-0" />
          <span className="hidden sm:inline">Saved</span>
          {watchlistCount > 0 && (
            <span className="ml-0.5 px-1.5 py-0.2 rounded-full bg-amber-500 text-slate-950 font-bold text-[10px]">
              {watchlistCount}
            </span>
          )}
        </button>

        {/* Theme Toggle (Device / Light / Dark) */}
        <ThemeToggle theme={theme} onThemeChange={onThemeChange} />

        {/* Health Status Indicator */}
        <div
          className="hidden xs:flex items-center gap-1.5 px-2 sm:px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono text-slate-300 min-h-[40px]"
          title={isOnline ? 'Sistem Berjalan Normal' : 'Menghubungkan ke Server'}
        >
          <span className={`w-2 h-2 rounded-full ${isOnline ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'}`} />
          <span className="hidden sm:inline">{isOnline ? 'Live' : 'Off'}</span>
        </div>

        {/* Quota Meter Button (Min 44px Touch Target) */}
        <button
          onClick={onOpenApiKeyModal}
          className="flex items-center gap-1.5 sm:gap-2 px-3 py-1.5 rounded-lg bg-slate-900 hover:bg-slate-850 active:bg-slate-800 border border-slate-700/80 hover:border-amber-500/50 transition text-xs font-mono text-slate-200 min-h-[40px] touch-manipulation shadow-sm"
          title="Klik untuk konfigurasi API Key dan Quota"
        >
          <Key className="w-3.5 h-3.5 text-amber-500 shrink-0" />
          <span className="hidden xs:inline text-slate-400">Quota:</span>
          <span className="text-amber-400 font-bold">{user ? user.quota : '50'}</span>
        </button>
      </div>
    </header>
  )
}
