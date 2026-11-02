import React from 'react'
import {
  Bookmark,
  X,
  Trash2,
  ExternalLink,
  Clock,
} from 'lucide-react'

export interface WatchlistItem {
  id: string
  title: string
  url: string
  source: string
  type: 'article' | 'cluster'
  summary?: string
  dateAdded: string
}

const STORAGE_KEY = 'the_dispatch_watchlist'

export function getStoredWatchlist(): WatchlistItem[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

export function saveStoredWatchlist(items: WatchlistItem[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
  } catch {
    // ignore
  }
}

interface WatchlistDrawerProps {
  isOpen: boolean
  onClose: () => void
  items: WatchlistItem[]
  onRemoveItem: (id: string) => void
  onClearAll: () => void
}

export const WatchlistDrawer: React.FC<WatchlistDrawerProps> = ({
  isOpen,
  onClose,
  items,
  onRemoveItem,
  onClearAll,
}) => {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex justify-end animate-in fade-in duration-150">
      <div
        className="w-full max-w-md bg-white dark:bg-slate-900 border-l border-slate-200 dark:border-slate-700/80 shadow-2xl h-full flex flex-col justify-between text-slate-900 dark:text-slate-100"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Drawer Header */}
        <div className="p-4 sm:p-5 border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950/80 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-500">
              <Bookmark className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-editorial text-lg font-bold text-slate-900 dark:text-slate-100">
                  Saved Watchlist
                </h3>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-amber-800 dark:text-amber-400 border border-amber-500/30 font-bold">
                  {items.length} Tersimpan
                </span>
              </div>
              <p className="text-xs text-slate-600 dark:text-slate-400 font-sans">
                Arsip artikel dan klaster sensitif yang Anda tandai
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-800 transition cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Drawer Content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {items.length === 0 ? (
            <div className="py-20 text-center text-slate-400 space-y-3">
              <div className="w-12 h-12 rounded-2xl bg-slate-100 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700 flex items-center justify-center mx-auto text-slate-400 dark:text-slate-500">
                <Bookmark className="w-6 h-6" />
              </div>
              <p className="text-sm font-semibold text-slate-700 dark:text-slate-300">Belum Ada Item Tersimpan</p>
              <p className="text-xs text-slate-500 max-w-xs mx-auto leading-relaxed">
                Tandai artikel dari Live Wire atau klaster topik dari Trending Radar dengan tombol bookmark untuk menyimpannya di sini.
              </p>
            </div>
          ) : (
            items.map((item) => (
              <div
                key={item.id}
                className="p-3.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 hover:border-slate-300 dark:hover:border-slate-700 transition space-y-2 group"
              >
                <div className="flex items-start justify-between gap-2">
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-white dark:bg-slate-800/80 text-amber-800 dark:text-amber-400 border border-slate-200 dark:border-slate-700 uppercase font-semibold">
                    {item.source || 'Sumber Berita'}
                  </span>
                  <button
                    onClick={() => onRemoveItem(item.id)}
                    className="text-slate-400 hover:text-rose-500 p-1 rounded transition opacity-0 group-hover:opacity-100 cursor-pointer"
                    title="Hapus dari Watchlist"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>

                <h4 className="font-semibold text-xs sm:text-sm text-slate-900 dark:text-slate-100 line-clamp-2 leading-snug">
                  {item.title}
                </h4>

                {item.summary && (
                  <p className="text-xs text-slate-600 dark:text-slate-400 line-clamp-2 leading-relaxed">
                    {item.summary}
                  </p>
                )}

                <div className="flex items-center justify-between pt-1 border-t border-slate-200 dark:border-slate-850 text-[11px] font-mono text-slate-500">
                  <span className="flex items-center gap-1">
                    <Clock className="w-3.5 h-3.5" /> {item.dateAdded}
                  </span>
                  {item.url && (
                    <a
                      href={item.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-amber-700 dark:text-amber-400 hover:text-amber-800 dark:hover:text-amber-300 flex items-center gap-1 font-sans"
                    >
                      Buka Sumber <ExternalLink className="w-3.5 h-3.5" />
                    </a>
                  )}
                </div>
              </div>
            ))
          )}
        </div>

        {/* Drawer Footer */}
        {items.length > 0 && (
          <div className="p-4 border-t border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950/80 flex items-center justify-between">
            <button
              onClick={onClearAll}
              className="text-xs font-mono text-rose-600 dark:text-rose-400 hover:text-rose-700 dark:hover:text-rose-300 transition flex items-center gap-1.5 cursor-pointer"
            >
              <Trash2 className="w-3.5 h-3.5" /> Kosongkan Semua
            </button>
            <button
              onClick={onClose}
              className="px-4 py-2 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-slate-700 text-xs font-mono transition cursor-pointer"
            >
              Tutup
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
