import React, { useState, useEffect } from 'react'
import { Header } from './components/Header'
import { TelegramBanner } from './components/TelegramBanner'
import { Navigation, type TabType } from './components/Navigation'
import { TrendingRadar } from './components/TrendingRadar'
import { QuickScraper } from './components/QuickScraper'
import { MediaBiasSpectrum } from './components/MediaBiasSpectrum'
import { SemanticInquiry } from './components/SemanticInquiry'
import { SystemTelemetry } from './components/SystemTelemetry'
import type { UserMe } from './types'
import { api, DEFAULT_DEMO_KEY } from './services/api'

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabType>('trending')
  const [apiKey, setApiKey] = useState<string>(() => {
    return localStorage.getItem('the_dispatch_api_key') || DEFAULT_DEMO_KEY
  })
  const [user, setUser] = useState<UserMe | null>(null)
  const [isOnline, setIsOnline] = useState(true)
  const [prefillComparisonUrls, setPrefillComparisonUrls] = useState<string[]>([])

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

  const handleSelectForComparison = (urls: string[]) => {
    setPrefillComparisonUrls(urls)
    setActiveTab('bias')
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans bg-grid-pattern selection:bg-amber-500/30 selection:text-amber-300 max-w-full overflow-x-hidden">
      {/* 1. Sticky Header Bar */}
      <Header
        user={user}
        apiKey={apiKey}
        onApiKeyChange={(k) => setApiKey(k)}
        isOnline={isOnline}
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
          />
        )}

        {activeTab === 'wire' && (
          <QuickScraper
            apiKey={apiKey}
            onSelectForComparison={handleSelectForComparison}
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
    </div>
  )
}

export default App
