import React from 'react'
import { Flame, Radio, Scale, MessageSquare, SlidersHorizontal } from 'lucide-react'

export type TabType = 'trending' | 'wire' | 'bias' | 'semantic' | 'telemetry'

interface NavigationProps {
  activeTab: TabType
  onTabChange: (tab: TabType) => void
}

export const Navigation: React.FC<NavigationProps> = ({ activeTab, onTabChange }) => {
  const tabs: { id: TabType; label: string; icon: React.FC<{ className?: string }> }[] = [
    { id: 'trending', label: 'Trending Radar', icon: Flame },
    { id: 'wire', label: 'Live Wire & Ingest', icon: Radio },
    { id: 'bias', label: 'Media Bias Spectrum', icon: Scale },
    { id: 'semantic', label: 'Semantic Inquiry & SSE', icon: MessageSquare },
    { id: 'telemetry', label: 'Engine Telemetry', icon: SlidersHorizontal },
  ]

  return (
    <nav 
      role="tablist" 
      aria-label="Navigasi Analisis Berita"
      className="border-b border-slate-200 dark:border-slate-800 bg-white/95 dark:bg-slate-950/95 backdrop-blur-md sticky top-[57px] z-30 px-3 sm:px-6 lg:px-8 shadow-xs transition-colors"
    >
      <div className="max-w-7xl mx-auto flex items-center gap-1 sm:gap-2 overflow-x-auto no-scrollbar overscroll-x-contain py-1">
        {tabs.map((tab) => {
          const Icon = tab.icon
          const isActive = activeTab === tab.id
          return (
            <button
              key={tab.id}
              role="tab"
              aria-selected={isActive}
              onClick={() => onTabChange(tab.id)}
              className={`relative flex items-center gap-2 px-3 sm:px-4 py-2.5 rounded-xl font-mono text-xs whitespace-nowrap shrink-0 transition-all duration-200 min-h-[44px] touch-manipulation cursor-pointer ${
                isActive
                  ? 'bg-amber-500/10 dark:bg-amber-500/15 text-amber-900 dark:text-amber-300 font-bold border border-amber-500/30 shadow-xs'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-900 border border-transparent'
              }`}
            >
              <Icon className={`w-4 h-4 shrink-0 transition-transform duration-200 ${isActive ? 'text-amber-600 dark:text-amber-400 scale-110' : 'text-slate-500 dark:text-slate-400'}`} />
              <span>{tab.label}</span>
              {isActive && (
                <span className="absolute bottom-1 left-4 right-4 h-0.5 bg-amber-500 rounded-full animate-in fade-in" />
              )}
            </button>
          )
        })}
      </div>
    </nav>
  )
}
