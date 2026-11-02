import React, { useState, useEffect, useCallback } from 'react'
import { Header } from './components/Header'
import { TelegramBanner } from './components/TelegramBanner'
import { Navigation, type TabType } from './components/Navigation'
import { TrendingRadar } from './components/TrendingRadar'
import { QuickScraper } from './components/QuickScraper'
import { MediaBiasSpectrum } from './components/MediaBiasSpectrum'
import { SemanticInquiry } from './components/SemanticInquiry'
import { SystemTelemetry } from './components/SystemTelemetry'
import { CommandPalette } from './components/CommandPalette'
import { ApiKeyModal } from './components/ApiKeyModal'
import { ExportModal } from './components/ExportModal'
import {
  WatchlistDrawer,
  type WatchlistItem,
  getStoredWatchlist,
  saveStoredWatchlist,
} from './components/WatchlistDrawer'
import {
  type ThemeMode,
  getInitialTheme,
  applyThemeToDOM,
} from './components/ThemeToggle'
import type { UserMe, TrendingClusterItem } from './types'
import { api, DEFAULT_DEMO_KEY } from './services/api'

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabType>('trending')
  const [apiKey, setApiKey] = useState<string>(() => {
    return localStorage.getItem('the_dispatch_api_key') || DEFAULT_DEMO_KEY
  })
  const [theme, setTheme] = useState<ThemeMode>(() => getInitialTheme())
  const [user, setUser] = useState<UserMe | null>(null)
  const [isOnline, setIsOnline] = useState(true)
  const [prefillComparisonUrls, setPrefillComparisonUrls] = useState<string[]>([])

  // SaaS Modals State
  const [isCmdOpen, setIsCmdOpen] = useState(false)
  const [isApiKeyOpen, setIsApiKeyOpen] = useState(false)
  const [isExportOpen, setIsExportOpen] = useState(false)
  const [isWatchlistOpen, setIsWatchlistOpen] = useState(false)

  // Global Clusters State (for Export Suite)
  const [trendingClusters, setTrendingClusters] = useState<TrendingClusterItem[]>([])

  // Watchlist State (persistent via localStorage)
  const [watchlist, setWatchlist] = useState<WatchlistItem[]>(() => getStoredWatchlist())

  const loadUserData = async (key = apiKey) => {
    try {
      const data = await api.getMe(key)
      setUser(data)
    } catch {
      setUser(null)
    }
  }

  const checkHealth = async () => {
    try {
      const res = await api.getHealth()
      setIsOnline(res.status === 'ok')
    } catch {
      setIsOnline(false)
    }
  }

  useEffect(() => {
    localStorage.setItem('the_dispatch_api_key', apiKey)
    loadUserData(apiKey)
    checkHealth()

    const interval = setInterval(checkHealth, 30000)
    return () => clearInterval(interval)
  }, [apiKey])

  // Sync and listen for device theme changes
  useEffect(() => {
    applyThemeToDOM(theme)
    localStorage.setItem('the_dispatch_theme', theme)

    if (theme === 'system' && typeof window !== 'undefined' && window.matchMedia) {
      const mql = window.matchMedia('(prefers-color-scheme: dark)')
      const listener = () => applyThemeToDOM('system')
      mql.addEventListener('change', listener)
      return () => mql.removeEventListener('change', listener)
    }
  }, [theme])

  // Global Cmd+K / Ctrl+K keyboard shortcut listener
  useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setIsCmdOpen((prev) => !prev)
      }
    }

    window.addEventListener('keydown', handleGlobalKeyDown)
    return () => window.removeEventListener('keydown', handleGlobalKeyDown)
  }, [])

  // Watchlist Helpers
  const handleToggleWatchlist = useCallback((item: WatchlistItem) => {
    setWatchlist((prev) => {
      const exists = prev.some((x) => x.id === item.id || (item.url && x.url === item.url))
      let updated: WatchlistItem[]
      if (exists) {
        updated = prev.filter((x) => x.id !== item.id && x.url !== item.url)
      } else {
        updated = [item, ...prev]
      }
      saveStoredWatchlist(updated)
      return updated
    })
  }, [])

  const handleRemoveWatchlistItem = useCallback((id: string) => {
    setWatchlist((prev) => {
      const updated = prev.filter((x) => x.id !== id)
      saveStoredWatchlist(updated)
      return updated
    })
  }, [])

  const handleClearWatchlist = useCallback(() => {
    saveStoredWatchlist([])
    setWatchlist([])
  }, [])

  const isWatched = useCallback(
    (url: string) => {
      return watchlist.some((x) => x.url === url)
    },
    [watchlist]
  )

  const handleSelectForComparison = (urls: string[]) => {
    setPrefillComparisonUrls(urls)
    setActiveTab('bias')
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const handleTriggerRecompute = async () => {
    try {
      await api.recomputeClusters(apiKey)
      const data = await api.getTrending(8)
      setTrendingClusters(data.topics || [])
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Gagal menghitung ulang kluster')
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans bg-grid-pattern selection:bg-amber-500/30 selection:text-amber-300 max-w-full overflow-x-hidden">
      {/* 1. Sticky Header Bar with SaaS Actions */}
      <Header
        user={user}
        apiKey={apiKey}
        isOnline={isOnline}
        watchlistCount={watchlist.length}
        theme={theme}
        onThemeChange={setTheme}
        onOpenCommandPalette={() => setIsCmdOpen(true)}
        onOpenApiKeyModal={() => setIsApiKeyOpen(true)}
        onOpenExportModal={() => setIsExportOpen(true)}
        onOpenWatchlist={() => setIsWatchlistOpen(true)}
      />

      {/* 2. Telegram Bot Gateway Showcase Banner */}
      <TelegramBanner />

      {/* 3. Navigation Bar (Touch & Swipe Friendly) */}
      <Navigation activeTab={activeTab} onTabChange={setActiveTab} />

      {/* 4. Main Active Tab View Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-3 sm:px-6 lg:px-8 py-5 sm:py-8">
        {activeTab === 'trending' && (
          <TrendingRadar
            apiKey={apiKey}
            onSelectForComparison={handleSelectForComparison}
            onOpenExport={() => setIsExportOpen(true)}
            onToggleWatchlist={handleToggleWatchlist}
            isWatched={isWatched}
            onClustersLoaded={setTrendingClusters}
          />
        )}

        {activeTab === 'wire' && (
          <QuickScraper
            apiKey={apiKey}
            onSelectForComparison={handleSelectForComparison}
            onToggleWatchlist={handleToggleWatchlist}
            isWatched={isWatched}
          />
        )}

        {activeTab === 'bias' && (
          <MediaBiasSpectrum
            apiKey={apiKey}
            prefillUrls={prefillComparisonUrls}
          />
        )}

        {activeTab === 'semantic' && <SemanticInquiry apiKey={apiKey} />}

        {activeTab === 'telemetry' && (
          <SystemTelemetry
            apiKey={apiKey}
            user={user}
            onRefreshUser={() => loadUserData(apiKey)}
          />
        )}
      </main>

      {/* 5. Editorial Footer */}
      <footer className="border-t border-slate-800 bg-slate-950 py-6 px-4 lg:px-8 text-xs text-slate-400 font-mono">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-3 text-center sm:text-left">
          <div className="flex items-center gap-2">
            <span className="font-editorial text-sm font-bold text-slate-100">
              THE DISPATCH
            </span>
            <span>• Newsroom Intelligence & Media Radar</span>
          </div>

          <div className="flex items-center gap-4 text-slate-400">
            <a
              href="https://t.me/PengumpulBeritaBot"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-amber-400 transition"
            >
              @PengumpulBeritaBot
            </a>
            <span>•</span>
            <a
              href="/swagger/index.html"
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-amber-400 transition"
            >
              OpenAPI Swagger
            </a>
            <span>•</span>
            <span>v1.0 Go + React 19</span>
          </div>
        </div>
      </footer>

      {/* 6. SaaS Modals & Drawers */}
      <CommandPalette
        isOpen={isCmdOpen}
        onClose={() => setIsCmdOpen(false)}
        onSelectTab={setActiveTab}
        onOpenApiKeyModal={() => setIsApiKeyOpen(true)}
        onOpenExportModal={() => setIsExportOpen(true)}
        onOpenWatchlist={() => setIsWatchlistOpen(true)}
        onTriggerRecompute={handleTriggerRecompute}
        onSelectTheme={setTheme}
      />

      <ApiKeyModal
        isOpen={isApiKeyOpen}
        onClose={() => setIsApiKeyOpen(false)}
        apiKey={apiKey}
        onApiKeyChange={(k) => setApiKey(k)}
        user={user}
      />

      <ExportModal
        isOpen={isExportOpen}
        onClose={() => setIsExportOpen(false)}
        clusters={trendingClusters}
      />

      <WatchlistDrawer
        isOpen={isWatchlistOpen}
        onClose={() => setIsWatchlistOpen(false)}
        items={watchlist}
        onRemoveItem={handleRemoveWatchlistItem}
        onClearAll={handleClearWatchlist}
      />
    </div>
  )
}

export default App
