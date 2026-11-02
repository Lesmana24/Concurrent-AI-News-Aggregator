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
      <header className="border-b border-[#1F2937] bg-[#0E131F]/90 backdrop-blur-md sticky top-0 z-40 px-4 lg:px-8 py-3.5 flex flex-wrap items-center justify-between gap-4">
        {/* Brand & Editorial Title */}
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-md bg-[#D97706]/10 border border-[#D97706]/30 flex items-center justify-center text-[#F59E0B] font-editorial font-bold text-xl select-none">
            D
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-editorial text-lg lg:text-xl font-bold tracking-tight text-[#F3F4F6]">
                THE DISPATCH
              </span>
              <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-[#1F2937] text-[#9CA3AF] border border-[#374151]">
                PLAYGROUND
              </span>
            </div>
            <p className="text-xs text-[#9CA3AF] font-sans hidden sm:block">
              Newsroom Intelligence & Multi-Perspective Radar Terminal
            </p>
          </div>
        </div>

        {/* Telemetry Status & Quota Controls */}
        <div className="flex items-center gap-3">
          {/* Health Status */}
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-[#111827] border border-[#1F2937] text-xs font-mono text-[#9CA3AF]">
            <span className={`w-2 h-2 rounded-full ${isOnline ? 'bg-[#10B981] animate-pulse' : 'bg-[#EF4444]'}`} />
            <span>{isOnline ? 'System Live' : 'Connecting'}</span>
          </div>

          {/* Quota Meter Badge */}
          <button
            onClick={() => setShowKeyModal(true)}
            className="flex items-center gap-2 px-3 py-1 rounded-md bg-[#111827] hover:bg-[#182032] border border-[#1F2937] hover:border-[#D97706]/40 transition text-xs font-mono text-[#F3F4F6]"
            title="Klik untuk konfigurasi API Key"
          >
            <Key className="w-3.5 h-3.5 text-[#D97706]" />
            <span>Quota:</span>
            <span className="text-[#F59E0B] font-semibold">{user ? user.quota : '50'} reqs</span>
          </button>
        </div>
      </header>

      {/* Modal API Key Settings */}
      {showKeyModal && (
        <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg max-w-md w-full p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="font-editorial text-lg font-bold text-[#F3F4F6] flex items-center gap-2">
                <Key className="w-5 h-5 text-[#D97706]" />
                API Key & Sandbox Access
              </h3>
              <button
                onClick={() => setShowKeyModal(false)}
                className="text-[#9CA3AF] hover:text-[#F3F4F6] p-1 rounded-md"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <p className="text-xs text-[#9CA3AF] leading-relaxed">
              Playground ini telah dilengkapi dengan <b>Demo Key bawaan</b> (<code className="text-[#F59E0B]">test-key-123</code>). Anda dapat menggantinya dengan API Key kustom Anda kapan saja.
            </p>

            <div className="space-y-1.5">
              <label className="text-xs font-mono text-[#9CA3AF]">Active API Key</label>
              <input
                type="text"
                value={tempKey}
                onChange={(e) => setTempKey(e.target.value)}
                placeholder="test-key-123"
                className="w-full bg-[#080C14] border border-[#1F2937] rounded-md px-3 py-2 text-sm text-[#F3F4F6] font-mono focus:outline-none focus:border-[#D97706]"
              />
            </div>

            <div className="bg-[#080C14] p-3 rounded-md border border-[#1F2937] flex items-center justify-between text-xs font-mono">
              <span className="text-[#9CA3AF]">Status Akses:</span>
              <span className="text-[#10B981] flex items-center gap-1">
                <ShieldCheck className="w-4 h-4" /> Terverifikasi
              </span>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => {
                  setTempKey(DEFAULT_DEMO_KEY)
                  onApiKeyChange(DEFAULT_DEMO_KEY)
                  setShowKeyModal(false)
                }}
                className="px-3 py-1.5 rounded-md text-xs font-mono text-[#9CA3AF] hover:bg-[#1F2937]"
              >
                Reset Default
              </button>
              <button
                onClick={handleSaveKey}
                className="px-4 py-1.5 rounded-md text-xs font-mono bg-[#D97706] hover:bg-[#B45309] text-white font-medium"
              >
                Simpan Kunci
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
