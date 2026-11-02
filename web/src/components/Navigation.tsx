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
    <nav className="border-b border-slate-800 bg-slate-950/95 sticky top-[57px] z-30 px-3 sm:px-6 lg:px-8 shadow-sm">
      <div className="max-w-7xl mx-auto flex items-center gap-1 sm:gap-2 overflow-x-auto no-scrollbar overscroll-x-contain py-1">
        {tabs.map((tab) => {
          const Icon = tab.icon
          const isActive = activeTab === tab.id
          return (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`flex items-center gap-2 px-3 sm:px-4 py-2.5 rounded-xl font-mono text-xs whitespace-nowrap shrink-0 transition-all min-h-[44px] touch-manipulation ${
                isActive
                  ? 'bg-amber-500/15 text-amber-400 font-bold border border-amber-500/30 shadow-sm'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900 border border-transparent'
              }`}
            >
              <Icon className={`w-4 h-4 shrink-0 ${isActive ? 'text-amber-500' : 'text-slate-400'}`} />
              <span>{tab.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}
