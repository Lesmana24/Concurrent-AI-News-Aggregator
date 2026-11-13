import React, { useState } from 'react'
import { AlertTriangle, Clock, UserPlus, ArrowRight, X } from 'lucide-react'
import type { UserMe } from '../types'
import { useWibCountdown } from '../utils/time'

interface QuotaExhaustedBannerProps {
  user: UserMe | null
  onOpenApiKeyModal: () => void
}

export const QuotaExhaustedBanner: React.FC<QuotaExhaustedBannerProps> = ({
  user,
  onOpenApiKeyModal,
}) => {
  const [dismissed, setDismissed] = useState(false)
  const countdown = useWibCountdown()

  // Only show when user is loaded and quota is exhausted
  if (!user || user.quota > 0 || dismissed) {
    return null
  }

  const isGuest = user.is_guest ?? false

  return (
    <div className="relative mb-6 rounded-2xl bg-gradient-to-r from-amber-500/15 via-orange-500/10 to-amber-500/15 border border-amber-500/40 p-4 sm:p-5 shadow-lg shadow-amber-500/5 backdrop-blur-md animate-in fade-in slide-in-from-top-2 duration-300">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        {/* Left Information */}
        <div className="flex items-start gap-3.5">
          <div className="p-2.5 rounded-xl bg-amber-500/20 border border-amber-500/30 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5">
            <AlertTriangle className="w-5 h-5 animate-pulse" />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-2 flex-wrap">
              <h4 className="font-editorial text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100">
                {isGuest
                  ? 'Batas Kuota Tamu Harian Telah Habis (0 / 5 Token)'
                  : 'Batas Kuota Harian Anda Telah Habis'}
              </h4>
              <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded-md bg-amber-500/20 text-amber-800 dark:text-amber-300 border border-amber-500/30 flex items-center gap-1">
                <Clock className="w-3 h-3" /> Reset dalam: {countdown}
              </span>
            </div>
            <p className="text-xs sm:text-sm text-slate-700 dark:text-slate-300 font-sans leading-relaxed">
              {isGuest
                ? 'Kuota 5 token tamu Anda akan otomatis tereset kembali pada pukul 00:00 WIB. Butuh komputasi berita sekarang tanpa menunggu besok? Daftarkan email developer gratis untuk langsung mendapatkan 100 token harian.'
                : 'Kuota harian Anda akan otomatis kembali penuh pada pukul 00:00 WIB.'}
            </p>
          </div>
        </div>

        {/* Action Button & Dismiss */}
        <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
          {isGuest && (
            <button
              type="button"
              onClick={onOpenApiKeyModal}
              className="px-4 py-2.5 rounded-xl bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-400 hover:to-amber-500 active:scale-95 text-slate-950 font-mono font-bold text-xs sm:text-sm transition flex items-center gap-1.5 shadow-md shadow-amber-500/20 cursor-pointer"
            >
              <UserPlus className="w-4 h-4" />
              <span>Daftar Developer (100 Token)</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </button>
          )}

          <button
            type="button"
            onClick={() => setDismissed(true)}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-200/50 dark:hover:bg-slate-800/50 transition cursor-pointer"
            title="Tutup pemberitahuan ini"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  )
}
