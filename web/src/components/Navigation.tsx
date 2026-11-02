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
    <nav className="border-b border-[#1F2937] bg-[#0E131F] px-4 lg:px-8 overflow-x-auto">
      <div className="max-w-7xl mx-auto flex items-center gap-1 sm:gap-2">
        {tabs.map((tab) => {
          const Icon = tab.icon
          const isActive = activeTab === tab.id
          return (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              className={`flex items-center gap-2 py-3 px-3.5 border-b-2 font-mono text-xs whitespace-nowrap transition-colors ${
                isActive
                  ? 'border-[#D97706] text-[#F59E0B] font-semibold bg-[#111827]/50'
                  : 'border-transparent text-[#9CA3AF] hover:text-[#F3F4F6] hover:border-[#374151]'
              }`}
            >
              <Icon className={`w-4 h-4 ${isActive ? 'text-[#D97706]' : 'text-[#6B7280]'}`} />
              <span>{tab.label}</span>
            </button>
          )
        })}
      </div>
    </nav>
  )
}

