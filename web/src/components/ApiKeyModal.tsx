import React, { useState } from 'react'
import {
  Key,
  X,
  Copy,
  Check,
  Eye,
  EyeOff,
  Terminal,
  Code2,
  ShieldCheck,
  Zap,
} from 'lucide-react'
import type { UserMe } from '../types'
import { DEFAULT_DEMO_KEY } from '../services/api'

interface ApiKeyModalProps {
  isOpen: boolean
  onClose: () => void
  apiKey: string
  onApiKeyChange: (newKey: string) => void
  user: UserMe | null
}

export const ApiKeyModal: React.FC<ApiKeyModalProps> = ({
  isOpen,
  onClose,
  apiKey,
  onApiKeyChange,
  user,
}) => {
  const [tempKey, setTempKey] = useState(apiKey)
  const [showKey, setShowKey] = useState(false)
  const [copied, setCopied] = useState(false)
  const [activeSnippetTab, setActiveSnippetTab] = useState<'curl' | 'ts' | 'python'>('curl')
  const [saveSuccess, setSaveSuccess] = useState(false)

  if (!isOpen) return null

  const handleCopyKey = () => {
    navigator.clipboard.writeText(apiKey)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleSave = () => {
    const cleaned = tempKey.trim() || DEFAULT_DEMO_KEY
    onApiKeyChange(cleaned)
    setSaveSuccess(true)
    setTimeout(() => setSaveSuccess(false), 2000)
  }

  const snippets = {
    curl: `curl -X GET "http://localhost:8080/api/v1/news/trending?limit=6" \\
  -H "X-API-KEY: ${apiKey}"`,
    ts: `// TypeScript / Browser Fetch
const response = await fetch('/api/v1/news/trending?limit=6', {
  headers: {
    'X-API-KEY': '${apiKey}',
  },
});
const { topics } = await response.json();
console.log('Top Clusters:', topics);`,
    python: `# Python 3 (requests)
import requests

url = "http://localhost:8080/api/v1/news/trending"
headers = {"X-API-KEY": "${apiKey}"}
params = {"limit": 6}

response = requests.get(url, headers=headers, params=params)
data = response.json()
print("Total Clusters:", data.get("total_clusters"))`,
  }

  const quotaValue = user?.quota ?? 50
  const maxQuota = 1000
  const quotaPercent = Math.min(Math.round((quotaValue / maxQuota) * 100), 100)

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-3 sm:p-4 animate-in fade-in duration-150">
      <div
        className="w-full max-w-2xl bg-slate-900 border border-slate-700/80 rounded-2xl shadow-2xl shadow-black/80 overflow-hidden flex flex-col max-h-[90vh]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between px-5 sm:px-6 py-4 border-b border-slate-800 bg-slate-950/70">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400">
              <Key className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-100">
                  Developer API & Quota Meter
                </h3>
                <span className="text-[10px] font-mono uppercase px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 font-semibold flex items-center gap-1">
                  <ShieldCheck className="w-3 h-3" /> Enterprise Tier
                </span>
              </div>
              <p className="text-xs text-slate-400 font-sans">
                Akses programatik ke pgvector semantic engine, clustering radar, dan RAG AI
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-100 hover:bg-slate-800 transition touch-manipulation"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-5 sm:p-6 overflow-y-auto space-y-6 text-sm text-slate-300 font-sans">
          {/* Quota & Usage Meter Card */}
          <div className="p-4 rounded-xl bg-slate-950/60 border border-slate-800 space-y-3">
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-slate-400 flex items-center gap-1.5">
                <Zap className="w-3.5 h-3.5 text-amber-400" /> Daily Reasoning & Ingest Quota
              </span>
              <span className="text-amber-400 font-bold">
                {quotaValue} / {maxQuota} Req
              </span>
            </div>
            {/* Visual Progress Bar */}
            <div className="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
              <div
                className="bg-gradient-to-r from-amber-500 to-amber-300 h-2 rounded-full transition-all duration-500"
                style={{ width: `${quotaPercent}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] font-mono text-slate-400">
              <span>Status: {user?.status || 'Active Sandbox'}</span>
              <span>Rate Limit: 60 RPM</span>
            </div>
          </div>

          {/* Active Key Display & Editor */}
          <div className="space-y-2">
            <label className="text-xs font-mono text-slate-400 flex items-center justify-between">
              <span>Active Access Key (Header: <code className="text-amber-400">X-API-KEY</code>)</span>
              {saveSuccess && (
                <span className="text-emerald-400 text-[11px] flex items-center gap-1 font-sans">
                  <Check className="w-3 h-3" /> Berhasil disimpan
                </span>
              )}
            </label>
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <input
                  type={showKey ? 'text' : 'password'}
                  value={tempKey}
                  onChange={(e) => setTempKey(e.target.value)}
                  placeholder="Masukkan API Key (misal: test-key-123)"
                  className="w-full px-3.5 py-2.5 rounded-xl bg-slate-950 border border-slate-700 focus:border-amber-500 focus:outline-none text-slate-100 font-mono text-xs sm:text-sm tracking-wider pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowKey(!showKey)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-200"
                  title={showKey ? 'Sembunyikan Key' : 'Tampilkan Key'}
                >
                  {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>

              <button
                type="button"
                onClick={handleCopyKey}
                className="px-3.5 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 active:bg-slate-750 text-slate-200 border border-slate-700 transition flex items-center gap-1.5 shrink-0 text-xs font-mono"
                title="Salin Key ke Clipboard"
              >
                {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
                <span>{copied ? 'Tersalin' : 'Salin'}</span>
              </button>

              <button
                type="button"
                onClick={handleSave}
                className="px-4 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-400 active:scale-95 text-slate-950 font-semibold transition text-xs font-mono shrink-0 shadow-md shadow-amber-500/20"
              >
                Terapkan
              </button>
            </div>
            <p className="text-[11px] text-slate-400 font-sans">
              Gunakan key default <code className="text-slate-300 font-mono">test-key-123</code> untuk pengujian lokal tanpa batasan sandbox.
            </p>
          </div>

          {/* Developer Quick-Start Code Snippets */}
          <div className="space-y-2 pt-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-mono text-slate-400 flex items-center gap-1.5 font-semibold">
                <Code2 className="w-4 h-4 text-amber-500" /> Developer Integration Snippet
              </span>
              <div className="flex items-center rounded-lg bg-slate-950 border border-slate-800 p-0.5 text-xs font-mono">
                <button
                  type="button"
                  onClick={() => setActiveSnippetTab('curl')}
                  className={`px-2.5 py-1 rounded-md transition ${
                    activeSnippetTab === 'curl'
                      ? 'bg-amber-500 text-slate-950 font-bold'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  cURL
                </button>
                <button
                  type="button"
                  onClick={() => setActiveSnippetTab('ts')}
                  className={`px-2.5 py-1 rounded-md transition ${
                    activeSnippetTab === 'ts'
                      ? 'bg-amber-500 text-slate-950 font-bold'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  TypeScript
                </button>
                <button
                  type="button"
                  onClick={() => setActiveSnippetTab('python')}
                  className={`px-2.5 py-1 rounded-md transition ${
                    activeSnippetTab === 'python'
                      ? 'bg-amber-500 text-slate-950 font-bold'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  Python
                </button>
              </div>
            </div>

            <div className="relative rounded-xl bg-slate-950 border border-slate-800 overflow-hidden">
              <pre className="p-3.5 text-xs font-mono text-slate-200 overflow-x-auto selection:bg-amber-500/30">
                <code>{snippets[activeSnippetTab]}</code>
              </pre>
            </div>
          </div>
        </div>

        {/* Modal Footer */}
        <div className="px-5 sm:px-6 py-3.5 bg-slate-950 border-t border-slate-800 flex items-center justify-between text-xs font-mono text-slate-400">
          <a
            href="/swagger/index.html"
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-amber-400 transition flex items-center gap-1"
          >
            <Terminal className="w-3.5 h-3.5" /> Buka Dokumentasi OpenAPI Swagger →
          </a>
          <button
            onClick={onClose}
            className="px-3.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 transition"
          >
            Tutup
          </button>
        </div>
      </div>
    </div>
  )
}
