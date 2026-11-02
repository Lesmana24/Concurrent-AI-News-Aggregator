import React, { useState } from 'react'
import { Key, ShieldCheck, X } from 'lucide-react'
import type { UserMe } from '../types'
import { DEFAULT_DEMO_KEY } from '../services/api'

interface HeaderProps {
  user: UserMe | null
  apiKey: string
  onApiKeyChange: (key: string) => void
  isOnline: boolean
}

export const Header: React.FC<HeaderProps> = ({ user, apiKey, onApiKeyChange, isOnline }) => {
  const [showKeyModal, setShowKeyModal] = useState(false)
  const [tempKey, setTempKey] = useState(apiKey)

  const handleSaveKey = () => {
    onApiKeyChange(tempKey.trim() || DEFAULT_DEMO_KEY)
    setShowKeyModal(false)
  }

  return (
    <>
      <header className="border-b border-slate-800 bg-slate-950/90 backdrop-blur-md sticky top-0 z-40 px-3 sm:px-6 lg:px-8 py-3 flex items-center justify-between gap-2 sm:gap-4">
        {/* Brand & Editorial Title */}
        <div className="flex items-center gap-2.5 sm:gap-3 shrink-0">
          <div className="w-8 h-8 sm:w-9 sm:h-9 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-center text-amber-400 font-editorial font-bold text-lg sm:text-xl select-none shadow-sm">
            D
          </div>
          <div>
            <div className="flex items-center gap-1.5 sm:gap-2">
              <span className="font-editorial text-base sm:text-xl font-bold tracking-tight text-slate-100">
                THE DISPATCH
              </span>
              <span className="text-[9px] sm:text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
                PLAYGROUND
              </span>
            </div>
            <p className="text-xs text-slate-400 font-sans hidden md:block">
              Newsroom Intelligence & Multi-Perspective Radar Terminal
            </p>
          </div>
        </div>

        {/* Telemetry Status & Quota Controls (Mobile Touch Optimized) */}
        <div className="flex items-center gap-2 shrink-0">
          {/* Health Status Indicator */}
          <div
            className="flex items-center gap-1.5 px-2 sm:px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono text-slate-300"
            title={isOnline ? 'Sistem Berjalan Normal' : 'Menghubungkan ke Server'}
          >
            <span className={`w-2 h-2 rounded-full ${isOnline ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'}`} />
            <span className="hidden sm:inline">{isOnline ? 'System Live' : 'Connecting'}</span>
          </div>

          {/* Quota Meter Button (Min 44px Touch Target) */}
          <button
            onClick={() => setShowKeyModal(true)}
            className="flex items-center gap-1.5 sm:gap-2 px-3 py-2 rounded-lg bg-slate-900 hover:bg-slate-850 active:bg-slate-800 border border-slate-700/80 hover:border-amber-500/50 transition text-xs font-mono text-slate-200 min-h-[44px] touch-manipulation shadow-sm"
            title="Klik untuk konfigurasi API Key"
          >
            <Key className="w-3.5 h-3.5 text-amber-500 shrink-0" />
            <span className="hidden xs:inline text-slate-400">Quota:</span>
            <span className="text-amber-400 font-bold">{user ? user.quota : '50'}</span>
          </button>
        </div>
      </header>

      {/* Modal API Key Settings (Touch & Mobile Ergonomic) */}
      {showKeyModal && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-3 sm:p-4 animate-in fade-in duration-150">
          <div className="bg-slate-900 border border-slate-700 rounded-2xl max-w-md w-full p-5 sm:p-6 shadow-2xl space-y-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between border-b border-slate-800 pb-3">
              <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-100 flex items-center gap-2">
                <Key className="w-5 h-5 text-amber-500" />
                <span>API Key & Sandbox Access</span>
              </h3>
              <button
                onClick={() => setShowKeyModal(false)}
                className="w-10 h-10 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-white flex items-center justify-center transition shrink-0"
                aria-label="Tutup Modal"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <p className="text-xs sm:text-sm text-slate-300 leading-relaxed font-sans">
              Playground ini telah dilengkapi dengan <b>Demo Key bawaan</b> (<code className="text-amber-400 font-mono">test-key-123</code>). Anda dapat menguji seluruh fitur atau menggantinya dengan API Key kustom Anda kapan saja.
            </p>

            <div className="space-y-1.5">
              <label className="text-xs font-mono font-semibold text-slate-300">Active API Key</label>
              <input
                type="text"
                value={tempKey}
                onChange={(e) => setTempKey(e.target.value)}
                placeholder="test-key-123"
                className="w-full bg-slate-950 border border-slate-700 focus:border-amber-500 rounded-xl px-3.5 py-3 text-base sm:text-sm text-slate-100 font-mono focus:outline-none transition min-h-[44px]"
              />
            </div>

            <div className="bg-slate-950 p-3.5 rounded-xl border border-slate-800 flex items-center justify-between text-xs font-mono">
              <span className="text-slate-400">Status Akses:</span>
              <span className="text-emerald-400 flex items-center gap-1.5 font-bold">
                <ShieldCheck className="w-4 h-4 text-emerald-400" />
                <span>Sandbox Mode (Aktif)</span>
              </span>
            </div>

            <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-slate-800">
              <button
                type="button"
                onClick={() => setShowKeyModal(false)}
                className="px-4 py-2.5 rounded-xl text-slate-300 hover:text-white text-xs font-mono font-medium min-h-[44px] flex items-center justify-center transition"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={handleSaveKey}
                className="px-5 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-white text-xs font-mono font-semibold min-h-[44px] flex items-center justify-center shadow-lg transition"
              >
                Simpan Konfigurasi
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
