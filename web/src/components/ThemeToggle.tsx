import React, { useState, useRef, useEffect } from 'react'
import { Sun, Moon, Monitor, Check } from 'lucide-react'

export type ThemeMode = 'system' | 'light' | 'dark'

const STORAGE_KEY = 'the_dispatch_theme'

export function getInitialTheme(): ThemeMode {
  try {
    const saved = localStorage.getItem(STORAGE_KEY) as ThemeMode | null
    if (saved === 'light' || saved === 'dark' || saved === 'system') {
      return saved
    }
  } catch {
    // ignore
  }
  return 'system'
}

export function resolveEffectiveTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') {
    if (typeof window !== 'undefined' && window.matchMedia) {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    }
    return 'dark'
  }
  return mode
}

export function applyThemeToDOM(mode: ThemeMode) {
  const effective = resolveEffectiveTheme(mode)
  const root = document.documentElement
  root.setAttribute('data-theme', effective)
  if (effective === 'dark') {
    root.classList.add('dark')
    root.classList.remove('light')
  } else {
    root.classList.add('light')
    root.classList.remove('dark')
  }
}

interface ThemeToggleProps {
  theme: ThemeMode
  onThemeChange: (theme: ThemeMode) => void
}

export const ThemeToggle: React.FC<ThemeToggleProps> = ({ theme, onThemeChange }) => {
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const effectiveTheme = resolveEffectiveTheme(theme)

  // Close dropdown when clicked outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen])

  const options: { mode: ThemeMode; label: string; desc: string; icon: React.ComponentType<{ className?: string }> }[] = [
    { mode: 'system', label: 'Bawaan Sistem', desc: 'Otomatis sinkron dengan tema OS/device Anda', icon: Monitor },
    { mode: 'light', label: 'Mode Terang', desc: 'Latar putih bersih dengan kontras teks tinggi', icon: Sun },
    { mode: 'dark', label: 'Mode Gelap', desc: 'Latar obsidian nyaman untuk ruang redup', icon: Moon },
  ]

  return (
    <div className="relative" ref={menuRef}>
      {/* Trigger Button (Min 40px Touch Target) */}
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-lg bg-slate-900 hover:bg-slate-850 active:bg-slate-800 border border-slate-700/80 hover:border-amber-500/50 transition text-xs font-mono text-slate-200 min-h-[40px] touch-manipulation shadow-sm"
        title={`Tema saat ini: ${theme === 'system' ? 'Bawaan Device' : theme === 'light' ? 'Terang' : 'Gelap'}`}
        aria-label="Pilih Tema Halaman"
      >
        {effectiveTheme === 'dark' ? (
          <Moon className="w-3.5 h-3.5 text-amber-400 shrink-0" />
        ) : (
          <Sun className="w-3.5 h-3.5 text-amber-500 shrink-0" />
        )}
        <span className="hidden sm:inline capitalize">
          {theme === 'system' ? 'Device' : theme}
        </span>
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute right-0 mt-2 w-56 rounded-xl bg-slate-900 border border-slate-700/80 shadow-2xl p-1.5 z-50 animate-in fade-in zoom-in-95 duration-100">
          <div className="px-2.5 py-1.5 text-[11px] font-mono text-slate-400 border-b border-slate-800 mb-1">
            PILIH TEMA TAMPILAN
          </div>
          {options.map((opt) => {
            const Icon = opt.icon
            const isSelected = theme === opt.mode
            return (
              <button
                key={opt.mode}
                type="button"
                onClick={() => {
                  onThemeChange(opt.mode)
                  setIsOpen(false)
                }}
                className={`w-full flex items-center justify-between px-2.5 py-2 rounded-lg text-left transition text-xs font-sans group ${
                  isSelected
                    ? 'bg-amber-500/15 text-amber-400 font-bold border border-amber-500/30'
                    : 'text-slate-300 hover:bg-slate-800/80 hover:text-slate-100'
                }`}
              >
                <div className="flex items-center gap-2.5">
                  <Icon
                    className={`w-4 h-4 shrink-0 ${
                      isSelected ? 'text-amber-400' : 'text-slate-400 group-hover:text-slate-200'
                    }`}
                  />
                  <div>
                    <div className="text-xs">{opt.label}</div>
                    <div className="text-[10px] text-slate-400 font-sans leading-tight mt-0.5">
                      {opt.desc}
                    </div>
                  </div>
                </div>
                {isSelected && <Check className="w-3.5 h-3.5 text-amber-400 shrink-0 ml-1.5" />}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

