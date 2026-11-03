import React, { useState, useEffect, useMemo, useRef } from 'react'
import {
  Search,
  Flame,
  Radio,
  Scale,
  MessageSquareCode,
  Activity,
  Key,
  Download,
  Bookmark,
  ExternalLink,
  RefreshCw,
  Command,
  ArrowRight,
  Sun,
  Moon,
  Monitor,
  X,
} from 'lucide-react'
import type { TabType } from './Navigation'
import type { ThemeMode } from './ThemeToggle'

export interface CommandPaletteProps {
  isOpen: boolean
  onClose: () => void
  onSelectTab: (tab: TabType) => void
  onOpenApiKeyModal: () => void
  onOpenExportModal: () => void
  onOpenWatchlist: () => void
  onTriggerRecompute?: () => void
  onSelectTheme?: (mode: ThemeMode) => void
}

interface CommandAction {
  id: string
  title: string
  subtitle: string
  category: 'Navigasi' | 'Aksi Sistem' | 'Intelijen & Alat'
  icon: React.ComponentType<{ className?: string }>
  run: () => void
  shortcut?: string
}

export const CommandPalette: React.FC<CommandPaletteProps> = ({
  isOpen,
  onClose,
  onSelectTab,
  onOpenApiKeyModal,
  onOpenExportModal,
  onOpenWatchlist,
  onTriggerRecompute,
  onSelectTheme,
}) => {
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)

  // Focus input when modal opens
  useEffect(() => {
    if (isOpen) {
      setQuery('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [isOpen])

  // Build command actions list
  const actions: CommandAction[] = useMemo(() => [
    {
      id: 'nav-trending',
      title: 'Buka Trending Topics & Hotness Radar',
      subtitle: 'Analisis klaster berita panas lintas portal redaksi',
      category: 'Navigasi',
      icon: Flame,
      run: () => { onSelectTab('trending'); onClose() },
      shortcut: 'Tab 1',
    },
    {
      id: 'nav-wire',
      title: 'Buka Live Wire & Ingest Feed',
      subtitle: 'Pantau arus berita real-time dan uji scraping URL instan',
      category: 'Navigasi',
      icon: Radio,
      run: () => { onSelectTab('wire'); onClose() },
      shortcut: 'Tab 2',
    },
    {
      id: 'nav-bias',
      title: 'Buka Media Bias Spectrum',
      subtitle: 'Analisis komparasi sudut pandang dan framing redaksi berita',
      category: 'Navigasi',
      icon: Scale,
      run: () => { onSelectTab('bias'); onClose() },
      shortcut: 'Tab 3',
    },
    {
      id: 'nav-semantic',
      title: 'Buka Semantic Inquiry & Executive Q&A',
      subtitle: 'Tanya jawab kecerdasan berita dan pencarian konsep liputan',
      category: 'Navigasi',
      icon: MessageSquareCode,
      run: () => { onSelectTab('semantic'); onClose() },
      shortcut: 'Tab 4',
    },
    {
      id: 'nav-telemetry',
      title: 'Buka System & Infrastructure Telemetry',
      subtitle: 'Pantau status kesehatan harvester dan infrastruktur analitik sistem',
      category: 'Navigasi',
      icon: Activity,
      run: () => { onSelectTab('telemetry'); onClose() },
      shortcut: 'Tab 5',
    },
    {
      id: 'act-recompute',
      title: 'Hitung Ulang Klaster Berita (Recompute)',
      subtitle: 'Perbarui pemetaan klaster topik dan tren berita terkini',
      category: 'Aksi Sistem',
      icon: RefreshCw,
      run: () => {
        onTriggerRecompute?.()
        onSelectTab('trending')
        onClose()
      },
      shortcut: 'Sync',
    },
    {
      id: 'tool-export',
      title: 'Ekspor Executive Briefing (Markdown & CSV)',
      subtitle: 'Unduh laporan intelijen harian dan dataset klaster terstruktur',
      category: 'Intelijen & Alat',
      icon: Download,
      run: () => { onOpenExportModal(); onClose() },
      shortcut: 'Exp',
    },
    {
      id: 'tool-watchlist',
      title: 'Buka Saved Watchlist & Bookmarks',
      subtitle: 'Lihat artikel dan klaster sensitif yang Anda tandai',
      category: 'Intelijen & Alat',
      icon: Bookmark,
      run: () => { onOpenWatchlist(); onClose() },
      shortcut: 'Saved',
    },
    {
      id: 'tool-api-key',
      title: 'Kelola Developer API Key & Token Quota',
      subtitle: 'Lihat sisa kuota, rate limits, dan contoh kode integrasi',
      category: 'Intelijen & Alat',
      icon: Key,
      run: () => { onOpenApiKeyModal(); onClose() },
      shortcut: 'API',
    },
    {
      id: 'theme-system',
      title: 'Ganti Tema: Bawaan Sistem / Device',
      subtitle: 'Otomatis sesuaikan dengan preferensi gelap/terang perangkat Anda',
      category: 'Aksi Sistem',
      icon: Monitor,
      run: () => { onSelectTheme?.('system'); onClose() },
      shortcut: 'Auto',
    },
    {
      id: 'theme-light',
      title: 'Ganti Tema: Mode Terang (Light Mode)',
      subtitle: 'Aktifkan palet putih bersih dengan kontras tinggi',
      category: 'Aksi Sistem',
      icon: Sun,
      run: () => { onSelectTheme?.('light'); onClose() },
      shortcut: 'Light',
    },
    {
      id: 'theme-dark',
      title: 'Ganti Tema: Mode Gelap (Dark Mode)',
      subtitle: 'Aktifkan palet obsidian yang nyaman di mata',
      category: 'Aksi Sistem',
      icon: Moon,
      run: () => { onSelectTheme?.('dark'); onClose() },
      shortcut: 'Dark',
    },
    {
      id: 'act-telegram',
      title: 'Buka Telegram Bot (@PengumpulBeritaBot)',
      subtitle: 'Akses notifikasi berita harian dan tanya jawab di ponsel Anda',
      category: 'Aksi Sistem',
      icon: ExternalLink,
      run: () => {
        window.open('https://t.me/PengumpulBeritaBot', '_blank')
        onClose()
      },
    },
  ], [onSelectTab, onClose, onTriggerRecompute, onOpenExportModal, onOpenWatchlist, onOpenApiKeyModal, onSelectTheme])

  // Filter actions based on query
  const filteredActions = useMemo(() => {
    if (!query.trim()) return actions
    const q = query.toLowerCase().trim()
    return actions.filter(
      (a) =>
        a.title.toLowerCase().includes(q) ||
        a.subtitle.toLowerCase().includes(q) ||
        a.category.toLowerCase().includes(q)
    )
  }, [actions, query])

  // Keyboard navigation inside command palette
  useEffect(() => {
    if (!isOpen) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((prev) => (prev + 1 < filteredActions.length ? prev + 1 : 0))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((prev) => (prev - 1 >= 0 ? prev - 1 : filteredActions.length - 1))
      } else if (e.key === 'Enter') {
        e.preventDefault()
        if (filteredActions[selectedIndex]) {
          filteredActions[selectedIndex].run()
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, filteredActions, selectedIndex, onClose])

  // Scroll selected item into view
  useEffect(() => {
    if (listRef.current) {
      const selectedEl = listRef.current.children[selectedIndex] as HTMLElement
      if (selectedEl) {
        selectedEl.scrollIntoView({ block: 'nearest' })
      }
    }
  }, [selectedIndex])

  if (!isOpen) return null

  return (
    <div 
      onClick={onClose}
      className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-start justify-center pt-8 sm:pt-20 px-3 sm:px-4 animate-in fade-in duration-150 cursor-pointer"
    >
      <div
        className="w-full max-w-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700/80 rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh] sm:max-h-[80vh] divide-y divide-slate-200 dark:divide-slate-800 text-slate-900 dark:text-slate-100 cursor-default"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search Input Bar with Mobile-Friendly Dismiss */}
        <div className="relative flex items-center px-3.5 sm:px-4 py-3 sm:py-3.5 bg-slate-50 dark:bg-slate-950/70 gap-2">
          <Search className="w-5 h-5 text-amber-500 shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setSelectedIndex(0)
            }}
            placeholder="Ketik perintah atau cari (misal: 'Trending', 'Export', 'API')..."
            className="w-full bg-transparent border-none text-slate-900 dark:text-slate-100 placeholder-slate-400 text-sm sm:text-base focus:outline-none font-sans"
          />
          {query && (
            <button
              onClick={() => {
                setQuery('')
                setSelectedIndex(0)
                inputRef.current?.focus()
              }}
              className="text-xs font-mono text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 px-1.5 py-0.5 rounded bg-slate-200 dark:bg-slate-800 shrink-0 cursor-pointer"
            >
              Hapus
            </button>
          )}
          <span className="hidden sm:inline-flex items-center gap-1 text-[11px] font-mono px-2 py-0.5 rounded bg-slate-200 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-slate-600 dark:text-slate-400 shrink-0">
            <kbd>ESC</kbd>
          </span>

          {/* Universal Close Button (Essential for Mobile touch screens) */}
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup menu pencarian"
            title="Tutup (Esc)"
            className="p-1.5 rounded-xl text-slate-400 hover:text-slate-700 dark:hover:text-slate-100 hover:bg-slate-200/80 dark:hover:bg-slate-800 transition shrink-0 cursor-pointer min-h-[38px] min-w-[38px] flex items-center justify-center border border-transparent hover:border-slate-300 dark:hover:border-slate-700"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Results List */}
        <div ref={listRef} className="overflow-y-auto p-2 space-y-1 max-h-[55vh]">
          {filteredActions.length === 0 ? (
            <div className="py-12 text-center text-slate-400 text-sm">
              <p>Tidak ada perintah yang sesuai dengan "{query}"</p>
              <p className="text-xs text-slate-500 mt-1">Coba kata kunci lain seperti 'Radar', 'Export', atau 'Bias'</p>
            </div>
          ) : (
            filteredActions.map((action, idx) => {
              const isSelected = idx === selectedIndex
              const IconComp = action.icon
              return (
                <button
                  key={action.id}
                  onClick={() => action.run()}
                  onMouseEnter={() => setSelectedIndex(idx)}
                  className={`w-full flex items-center justify-between p-3 rounded-xl text-left transition-all duration-100 group cursor-pointer ${
                    isSelected
                      ? 'bg-amber-500/10 border border-amber-500/40 text-slate-900 dark:text-slate-100 shadow-sm'
                      : 'hover:bg-slate-100 dark:hover:bg-slate-800/60 border border-transparent text-slate-700 dark:text-slate-300'
                  }`}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div
                      className={`p-2 rounded-lg shrink-0 ${
                        isSelected
                          ? 'bg-amber-500 text-slate-950 font-bold'
                          : 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 group-hover:text-amber-600 dark:group-hover:text-amber-400'
                      }`}
                    >
                      <IconComp className="w-4 h-4" />
                    </div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-xs sm:text-sm truncate">
                          {action.title}
                        </span>
                        <span className="text-[10px] font-mono uppercase px-1.5 py-0.2 rounded bg-slate-100 dark:bg-slate-800/80 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-700/60 shrink-0 font-medium">
                          {action.category}
                        </span>
                      </div>
                      <p className="text-xs text-slate-500 dark:text-slate-400 truncate mt-0.5">
                        {action.subtitle}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0 ml-3">
                    {action.shortcut && (
                      <span className="text-[10px] font-mono text-slate-500 dark:text-slate-400 bg-slate-100 dark:bg-slate-800/80 px-2 py-0.5 rounded border border-slate-200 dark:border-slate-700/50">
                        {action.shortcut}
                      </span>
                    )}
                    <ArrowRight
                      className={`w-3.5 h-3.5 transition-transform ${
                        isSelected ? 'text-amber-600 dark:text-amber-400 translate-x-0.5' : 'text-slate-400 dark:text-slate-600'
                      }`}
                    />
                  </div>
                </button>
              )
            })
          )}
        </div>

        {/* Footer Shortcut Legend & Mobile Action */}
        <div className="px-4 py-2.5 bg-slate-50 dark:bg-slate-950/90 flex items-center justify-between text-[11px] font-mono text-slate-500 dark:text-slate-400">
          <div className="hidden sm:flex items-center gap-3">
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 rounded bg-slate-200 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-slate-700 dark:text-slate-300">↑↓</kbd> Navigasi
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 rounded bg-slate-200 dark:bg-slate-800 border border-slate-300 dark:border-slate-700 text-slate-700 dark:text-slate-300">↵</kbd> Jalankan
            </span>
          </div>

          {/* Mobile Tap Dismiss Button */}
          <button
            type="button"
            onClick={onClose}
            className="sm:hidden text-amber-700 dark:text-amber-400 font-bold flex items-center gap-1 py-1 cursor-pointer"
          >
            <X className="w-3.5 h-3.5" />
            <span>Tutup Menu</span>
          </button>

          <span className="flex items-center gap-1 text-slate-400 dark:text-slate-500">
            <Command className="w-3 h-3 text-amber-500/70" /> The Dispatch
          </span>
        </div>
      </div>
    </div>
  )
}
