import React, { useState } from 'react'
import { QrCode, ExternalLink, X, Smartphone } from 'lucide-react'

export const TelegramBanner: React.FC = () => {
  const [showQR, setShowQR] = useState(false)
  const [isDismissed, setIsDismissed] = useState<boolean>(() => {
    return localStorage.getItem('the_dispatch_hide_telegram_banner') === 'true'
  })

  const handleDismiss = () => {
    setIsDismissed(true)
    localStorage.setItem('the_dispatch_hide_telegram_banner', 'true')
  }

  if (isDismissed) return null

  const botUsername = 'PengumpulBeritaBot'
  const botUrl = `https://t.me/${botUsername}`

  return (
    <>
      <section className="bg-gradient-to-r from-slate-100/80 via-white to-slate-50 dark:from-slate-900 dark:via-slate-900/90 dark:to-slate-950 border-b border-slate-200 dark:border-slate-800 px-3 sm:px-6 lg:px-8 py-3 transition-colors relative">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center justify-between gap-3 pr-8 md:pr-0">
          <div className="flex items-start sm:items-center gap-3">
            <div className="relative shrink-0 mt-0.5 sm:mt-0">
              <img
                src="/botpic.jpg"
                alt="@PengumpulBeritaBot Profile"
                className="w-9 h-9 sm:w-10 sm:h-10 rounded-full object-cover border-2 border-blue-500/60 shadow-md ring-1 ring-blue-500/30"
                onError={(e) => {
                  ;(e.target as HTMLElement).style.display = 'none'
                }}
              />
              <div
                className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-blue-500 text-white flex items-center justify-center text-[8px] font-bold border-2 border-white dark:border-slate-900 shadow-sm"
                title="Official Telegram Bot"
              >
                ✓
              </div>
            </div>
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-editorial text-sm sm:text-base font-bold text-slate-900 dark:text-slate-100">
                  Daily Driver Penuh di Telegram Bot
                </span>
                <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded-full bg-blue-500/15 text-blue-700 dark:text-blue-300 border border-blue-500/40 flex items-center gap-1 shadow-sm">
                  <span className="w-1.5 h-1.5 rounded-full bg-blue-500 dark:bg-blue-400 animate-pulse" />
                  <span>Official Bot</span>
                </span>
              </div>
              <p className="text-xs text-slate-600 dark:text-slate-300 font-sans mt-0.5 leading-relaxed">
                Notifikasi <i>Morning Briefing</i> otomatis (07:00 & 17:00 WIB), langganan topik, dan tanya-jawab AI interaktif langsung di smartphone Anda.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2 w-full md:w-auto shrink-0 pt-1 md:pt-0">
            {/* QR Code button only shown on tablet & desktop */}
            <button
              onClick={() => setShowQR(true)}
              className="hidden sm:flex items-center gap-1.5 px-3 py-2 rounded-xl bg-white dark:bg-slate-900 hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700 text-xs font-mono font-medium text-slate-700 dark:text-slate-200 min-h-[40px] transition shadow-xs cursor-pointer"
              title="Lihat QR Code untuk Smartphone"
            >
              <QrCode className="w-4 h-4 text-slate-500 dark:text-slate-400" />
              <span>QR Code</span>
            </button>

            {/* Main Action Button */}
            <a
              href={botUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex-1 md:flex-initial flex items-center justify-center gap-2 px-3.5 py-2 rounded-xl bg-amber-600 hover:bg-amber-500 active:bg-amber-700 text-white text-xs font-mono font-bold min-h-[40px] transition shadow-md touch-manipulation cursor-pointer"
            >
              <span>Buka @{botUsername}</span>
              <ExternalLink className="w-3.5 h-3.5" />
            </a>

            {/* Dismiss Banner Button (Desktop & Mobile) */}
            <button
              onClick={handleDismiss}
              aria-label="Tutup banner Telegram"
              title="Tutup banner ini agar layar lebih lega"
              className="p-2 rounded-xl text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-slate-200/60 dark:hover:bg-slate-800 transition cursor-pointer min-h-[40px] min-w-[40px] flex items-center justify-center"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>
      </section>

      {/* QR Code Modal for Desktop users */}
      {showQR && (
        <div 
          className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in duration-150"
          onClick={() => setShowQR(false)}
        >
          <div 
            className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700/80 rounded-2xl max-w-sm w-full p-6 text-center space-y-4 shadow-2xl relative text-slate-900 dark:text-slate-100"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              onClick={() => setShowQR(false)}
              className="w-10 h-10 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 absolute top-3 right-3 flex items-center justify-center transition cursor-pointer"
              aria-label="Tutup QR Code"
            >
              <X className="w-5 h-5" />
            </button>

            <div className="w-12 h-12 rounded-xl bg-blue-500/15 border border-blue-500/30 text-blue-600 dark:text-blue-400 flex items-center justify-center mx-auto">
              <Smartphone className="w-6 h-6" />
            </div>

            <div>
              <h3 className="font-editorial text-lg font-bold text-slate-900 dark:text-slate-100">
                Pindai untuk Membuka Bot
              </h3>
              <p className="text-xs text-slate-600 dark:text-slate-300 mt-1 font-sans">
                Arahkan kamera smartphone Anda ke QR Code di bawah untuk langsung membuka chat di aplikasi Telegram:
              </p>
            </div>

            {/* Generated QR Code Image */}
            <div className="p-3 bg-white rounded-xl inline-block shadow-md border border-slate-200 dark:border-slate-700 mx-auto">
              <img
                src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(botUrl)}`}
                alt="QR Code Telegram Bot"
                width={180}
                height={180}
                className="w-44 h-44"
              />
            </div>

            <div className="pt-1">
              <span className="text-xs font-mono font-bold text-amber-700 dark:text-amber-400">
                @{botUsername}
              </span>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
