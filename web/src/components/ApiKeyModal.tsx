import React, { useState, useEffect } from 'react'
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
  RefreshCw,
  UserPlus,
  AlertCircle,
  Loader2,
} from 'lucide-react'
import type { UserMe } from '../types'
import { api, DEFAULT_DEMO_KEY } from '../services/api'

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

  // Registration state
  const [registerEmail, setRegisterEmail] = useState('')
  const [isRegistering, setIsRegistering] = useState(false)
  const [registerMessage, setRegisterMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  // Rotation state
  const [showRotateConfirm, setShowRotateConfirm] = useState(false)
  const [isRotating, setIsRotating] = useState(false)
  const [rotateMessage, setRotateMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  useEffect(() => {
    setTempKey(apiKey)
  }, [apiKey])

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

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!registerEmail.trim()) return

    setIsRegistering(true)
    setRegisterMessage(null)
    try {
      const resp = await api.registerUser(registerEmail.trim())
      onApiKeyChange(resp.api_key)
      setTempKey(resp.api_key)
      setRegisterMessage({
        type: 'success',
        text: `Registrasi berhasil! API Key baru telah diterapkan (Kuota: ${resp.quota}).`,
      })
      setRegisterEmail('')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setRegisterMessage({ type: 'error', text: msg })
    } finally {
      setIsRegistering(false)
    }
  }

  const handleRotate = async () => {
    if (isGuest) {
      setRotateMessage({ type: 'error', text: 'Sesi Guest tidak diizinkan merotasi key demi keamanan kuota harian.' })
      return
    }
    setIsRotating(true)
    setRotateMessage(null)
    try {
      const resp = await api.rotateApiKey(apiKey)
      onApiKeyChange(resp.api_key)
      setTempKey(resp.api_key)
      setShowRotateConfirm(false)
      setRotateMessage({
        type: 'success',
        text: 'API Key berhasil dirotasi! Key lama telah langsung dicabut.',
      })
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 3000)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setRotateMessage({ type: 'error', text: msg })
    } finally {
      setIsRotating(false)
    }
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

  const isGuest = Boolean(user?.is_guest ?? (!apiKey || apiKey.startsWith('guest_') || apiKey === DEFAULT_DEMO_KEY))
  const maxQuota = user?.daily_limit ?? (isGuest ? 5 : (user ? 100 : 5))
  const quotaValue = user ? user.quota : (isGuest ? 5 : 100)
  const quotaPercent = Math.min(Math.round((quotaValue / (maxQuota || 1)) * 100), 100)

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-3 sm:p-4 animate-in fade-in duration-150">
      <div
        className="w-full max-w-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700/80 rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh] text-slate-900 dark:text-slate-100"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between px-5 sm:px-6 py-4 border-b border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-950/70">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-500">
              <Key className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-900 dark:text-slate-100">
                  Developer API & Quota Meter
                </h3>
                {isGuest ? (
                  <span className="text-[10px] font-mono uppercase px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-700 dark:text-amber-400 border border-amber-500/30 font-semibold flex items-center gap-1">
                    <Zap className="w-3 h-3" /> Guest Pass (5 Req/Hari)
                  </span>
                ) : (
                  <span className="text-[10px] font-mono uppercase px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border border-emerald-500/30 font-semibold flex items-center gap-1">
                    <ShieldCheck className="w-3 h-3" /> Developer Tier
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-600 dark:text-slate-400 font-sans">
                Akses terintegrasi ke engine kecerdasan berita, radar klastering tren, dan sintesis liputan
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-100 hover:bg-slate-100 dark:hover:bg-slate-800 transition touch-manipulation cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-5 sm:p-6 overflow-y-auto space-y-6 text-sm text-slate-700 dark:text-slate-300 font-sans">
          {/* Quota & Usage Meter Card */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950/60 border border-slate-200 dark:border-slate-800 space-y-3">
            <div className="flex items-center justify-between text-xs font-mono">
              <span className="text-slate-600 dark:text-slate-400 flex items-center gap-1.5">
                <Zap className="w-3.5 h-3.5 text-amber-500" /> {isGuest ? 'Guest Daily Token Quota' : 'Daily Reasoning & Ingest Quota'}
              </span>
              <span className="text-amber-700 dark:text-amber-400 font-bold">
                {quotaValue} / {maxQuota} Req
              </span>
            </div>
            {/* Visual Progress Bar */}
            <div className="w-full bg-slate-200 dark:bg-slate-800 rounded-full h-2 overflow-hidden">
              <div
                className="bg-gradient-to-r from-amber-500 to-amber-400 h-2 rounded-full transition-all duration-500"
                style={{ width: `${quotaPercent}%` }}
              />
            </div>
            <div className="flex items-center justify-between text-[11px] font-mono text-slate-500 dark:text-slate-400">
              <span>Status: {isGuest ? 'Guest Sandbox (Auto Reset 00:00 WIB)' : (user?.status || 'Active Sandbox')}</span>
              <span>Rate Limit: {isGuest ? '20 RPM' : '60 RPM'}</span>
            </div>
          </div>

          {/* Active Key Display & Editor */}
          <div className="space-y-2">
            <label className="text-xs font-mono text-slate-600 dark:text-slate-400 flex items-center justify-between">
              <span>Active Access Key (Header: <code className="text-amber-700 dark:text-amber-400 font-bold">X-API-KEY</code>)</span>
              {saveSuccess && (
                <span className="text-emerald-700 dark:text-emerald-400 text-[11px] flex items-center gap-1 font-sans font-medium">
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
                  className="w-full px-3.5 py-2.5 rounded-xl bg-slate-50 dark:bg-slate-950 border border-slate-300 dark:border-slate-700 focus:border-amber-500 focus:outline-none text-slate-900 dark:text-slate-100 font-mono text-xs sm:text-sm tracking-wider pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowKey(!showKey)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
                  title={showKey ? 'Sembunyikan Key' : 'Tampilkan Key'}
                >
                  {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>

              <button
                type="button"
                onClick={handleCopyKey}
                className="px-3.5 py-2.5 rounded-xl bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 active:bg-slate-300 dark:active:bg-slate-750 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-slate-700 transition flex items-center gap-1.5 shrink-0 text-xs font-mono cursor-pointer"
                title="Salin Key ke Clipboard"
              >
                {copied ? <Check className="w-4 h-4 text-emerald-600 dark:text-emerald-400" /> : <Copy className="w-4 h-4" />}
                <span>{copied ? 'Tersalin' : 'Salin'}</span>
              </button>

              <button
                type="button"
                onClick={handleSave}
                className="px-4 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-400 active:scale-95 text-slate-950 font-semibold transition text-xs font-mono shrink-0 shadow-md shadow-amber-500/20 cursor-pointer"
              >
                Terapkan
              </button>
            </div>
            <div className="flex items-center justify-between pt-1">
              <p className="text-[11px] text-slate-500 dark:text-slate-400 font-sans">
                {isGuest
                  ? 'Sesi Guest Otomatis: Batas 5 token per hari, tereset setiap 00:00 WIB.'
                  : 'Gunakan key unik Anda untuk mengakses endpoint API.'}
              </p>
              {!isGuest && !showRotateConfirm && (
                <button
                  type="button"
                  onClick={() => {
                    setShowRotateConfirm(true)
                    setRotateMessage(null)
                  }}
                  className="px-2.5 py-1 text-xs font-mono text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 hover:bg-amber-500/10 rounded-lg transition flex items-center gap-1.5 cursor-pointer shrink-0"
                >
                  <RefreshCw className="w-3.5 h-3.5" /> Rotasi Key
                </button>
              )}
            </div>

            {/* Guest Anti-Abuse Notification */}
            {isGuest && (
              <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/30 text-xs text-amber-800 dark:text-amber-300 space-y-1.5 mt-2">
                <div className="flex items-center gap-1.5 font-bold">
                  <AlertCircle className="w-4 h-4 shrink-0 text-amber-600 dark:text-amber-400" />
                  <span>Proteksi Kuota Tamu & Anti-Manipulasi:</span>
                </div>
                <p className="text-[11px] text-slate-600 dark:text-slate-300 font-sans leading-relaxed">
                  Setiap pengunjung website mendapatkan <strong>5 token harian</strong> yang otomatis kembali menjadi 5 pada pukul <strong>00:00 WIB</strong>. Rotasi / pencabutan key dinonaktifkan pada sesi tamu untuk mencegah bypass kuota. Untuk batas 100 token harian dan kemampuan rotasi key, daftarkan email Anda di bawah.
                </p>
              </div>
            )}

            {/* Rotate Confirmation dialog/banner */}
            {showRotateConfirm && (
              <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/30 text-xs space-y-2 mt-2">
                <div className="flex items-start gap-2 text-amber-700 dark:text-amber-400">
                  <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
                  <p>
                    <strong>Konfirmasi Rotasi:</strong> API key saat ini akan langsung dicabut (revoked) dan digantikan dengan key baru. Tindakan ini tidak dapat dibatalkan.
                  </p>
                </div>
                <div className="flex justify-end gap-2 pt-1">
                  <button
                    type="button"
                    disabled={isRotating}
                    onClick={() => setShowRotateConfirm(false)}
                    className="px-3 py-1 rounded-lg border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-600 dark:text-slate-400 font-mono text-xs cursor-pointer"
                  >
                    Batal
                  </button>
                  <button
                    type="button"
                    disabled={isRotating}
                    onClick={handleRotate}
                    className="px-3 py-1 rounded-lg bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold font-mono text-xs flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
                  >
                    {isRotating ? (
                      <>
                        <Loader2 className="w-3.5 h-3.5 animate-spin" /> Merotasi...
                      </>
                    ) : (
                      <>
                        <RefreshCw className="w-3.5 h-3.5" /> Ya, Rotasi Sekarang
                      </>
                    )}
                  </button>
                </div>
              </div>
            )}

            {rotateMessage && (
              <div
                className={`p-2.5 rounded-lg text-xs flex items-center gap-2 mt-2 ${
                  rotateMessage.type === 'success'
                    ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-500/20'
                    : 'bg-red-500/10 text-red-700 dark:text-red-400 border border-red-500/20'
                }`}
              >
                {rotateMessage.type === 'success' ? <Check className="w-4 h-4 shrink-0" /> : <AlertCircle className="w-4 h-4 shrink-0" />}
                <span>{rotateMessage.text}</span>
              </div>
            )}
          </div>

          {/* Register New Account Section */}
          <div className="p-4 rounded-xl bg-slate-50 dark:bg-slate-950/40 border border-slate-200 dark:border-slate-800 space-y-3">
            <div className="flex items-center gap-2">
              <UserPlus className="w-4 h-4 text-amber-500" />
              <h4 className="text-xs font-mono font-semibold uppercase tracking-wider text-slate-700 dark:text-slate-300">
                Registrasi Akun Developer Baru
              </h4>
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Daftarkan email Anda untuk mendapatkan API Key unik dengan 100 kuota harian gratis.
            </p>
            <form onSubmit={handleRegister} className="flex flex-col sm:flex-row gap-2">
              <input
                type="email"
                value={registerEmail}
                onChange={(e) => setRegisterEmail(e.target.value)}
                placeholder="nama@email.com"
                className="flex-1 px-3.5 py-2 rounded-xl bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 focus:border-amber-500 focus:outline-none text-slate-900 dark:text-slate-100 font-mono text-xs"
              />
              <button
                type="submit"
                disabled={isRegistering || !registerEmail.trim()}
                className="px-4 py-2 rounded-xl bg-slate-900 dark:bg-slate-100 hover:bg-slate-800 dark:hover:bg-white text-white dark:text-slate-950 font-semibold font-mono text-xs transition flex items-center justify-center gap-1.5 shrink-0 disabled:opacity-50 cursor-pointer shadow-sm"
              >
                {isRegistering ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" /> Mendaftar...
                  </>
                ) : (
                  <>
                    <UserPlus className="w-3.5 h-3.5" /> Daftar API Key
                  </>
                )}
              </button>
            </form>
            {registerMessage && (
              <div
                className={`p-2.5 rounded-lg text-xs flex items-center gap-2 ${
                  registerMessage.type === 'success'
                    ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border border-emerald-500/20'
                    : 'bg-red-500/10 text-red-700 dark:text-red-400 border border-red-500/20'
                }`}
              >
                {registerMessage.type === 'success' ? <Check className="w-4 h-4 shrink-0" /> : <AlertCircle className="w-4 h-4 shrink-0" />}
                <span>{registerMessage.text}</span>
              </div>
            )}
          </div>

          {/* Developer Quick-Start Code Snippets */}
          <div className="space-y-2 pt-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-mono text-slate-600 dark:text-slate-400 flex items-center gap-1.5 font-semibold">
                <Code2 className="w-4 h-4 text-amber-500" /> Developer Integration Snippet
              </span>
              <div className="flex items-center rounded-lg bg-slate-100 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 p-0.5 text-xs font-mono">
                <button
                  type="button"
                  onClick={() => setActiveSnippetTab('curl')}
                  className={`px-2.5 py-1 rounded-md transition cursor-pointer ${
                    activeSnippetTab === 'curl'
                      ? 'bg-amber-500 text-slate-950 font-bold'
                      : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
                  }`}
                >
                  cURL
                </button>
                <button
                  type="button"
                  onClick={() => setActiveSnippetTab('ts')}
                  className={`px-2.5 py-1 rounded-md transition cursor-pointer ${
                    activeSnippetTab === 'ts'
                      ? 'bg-amber-500 text-slate-950 font-bold'
                      : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
                  }`}
                >
                  TypeScript
                </button>
                <button
                  type="button"
                  onClick={() => setActiveSnippetTab('python')}
                  className={`px-2.5 py-1 rounded-md transition cursor-pointer ${
                    activeSnippetTab === 'python'
                      ? 'bg-amber-500 text-slate-950 font-bold'
                      : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
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
        <div className="px-5 sm:px-6 py-3.5 bg-slate-50 dark:bg-slate-950 border-t border-slate-200 dark:border-slate-800 flex items-center justify-between text-xs font-mono text-slate-600 dark:text-slate-400">
          <a
            href="/swagger/index.html"
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-amber-600 dark:hover:text-amber-400 transition flex items-center gap-1"
          >
            <Terminal className="w-3.5 h-3.5" /> Buka Dokumentasi API Reference →
          </a>
          <button
            onClick={onClose}
            className="px-3.5 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 border border-slate-200 dark:border-slate-700 transition cursor-pointer"
          >
            Tutup
          </button>
        </div>
      </div>
    </div>
  )
}
