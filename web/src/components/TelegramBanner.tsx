import React, { useState } from 'react'
import { Send, QrCode, ExternalLink, X, Smartphone } from 'lucide-react'

export const TelegramBanner: React.FC = () => {
  const [showQR, setShowQR] = useState(false)
  const botUsername = 'PengumpulBeritaBot'
  const botUrl = `https://t.me/${botUsername}`

  return (
    <>
      <section className="bg-gradient-to-r from-[#182032] via-[#111827] to-[#0E131F] border-b border-[#1F2937] px-4 lg:px-8 py-3.5">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center justify-between gap-3">
          <div className="flex items-start md:items-center gap-3">
            <div className="w-8 h-8 rounded-md bg-[#2563EB]/15 border border-[#2563EB]/30 flex items-center justify-center text-[#60A5FA] shrink-0">
              <Send className="w-4 h-4 -rotate-12 translate-x-0.5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-editorial text-sm lg:text-base font-semibold text-[#F3F4F6]">
                  Pengalaman Harian Penuh Tersedia di Telegram Bot
                </span>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-[#2563EB]/20 text-[#93C5FD] border border-[#2563EB]/30 hidden sm:inline-block">
                  Daily Driver
                </span>
              </div>
              <p className="text-xs text-[#9CA3AF] leading-normal">
                Dapatkan push notifikasi <i>Morning Briefing</i> harian (07:00 & 17:00 WIB), langganan topik berita, dan tanya jawab langsung di smartphone Anda.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2 self-start md:self-auto shrink-0">
            <button
              onClick={() => setShowQR(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-[#111827] hover:bg-[#182032] border border-[#1F2937] text-xs font-mono text-[#F3F4F6] transition"
              title="Lihat QR Code"
            >
              <QrCode className="w-3.5 h-3.5 text-[#9CA3AF]" />
              <span className="hidden sm:inline">QR Code</span>
            </button>
            <a
              href={botUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition shadow-sm"
            >
              <span>Buka @{botUsername}</span>
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
          </div>
        </div>
      </section>

      {/* QR Code Modal */}
      {showQR && (
        <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg max-w-sm w-full p-6 text-center space-y-4 shadow-2xl relative">
            <button
              onClick={() => setShowQR(false)}
              className="absolute top-4 right-4 text-[#9CA3AF] hover:text-[#F3F4F6] p-1"
            >
              <X className="w-5 h-5" />
            </button>

            <div className="w-12 h-12 mx-auto rounded-full bg-[#2563EB]/15 border border-[#2563EB]/30 flex items-center justify-center text-[#60A5FA]">
              <Smartphone className="w-6 h-6" />
            </div>

            <div>
              <h3 className="font-editorial text-lg font-bold text-[#F3F4F6]">
                Scan untuk Membuka Bot
              </h3>
              <p className="text-xs text-[#9CA3AF] mt-1">
                Arahkan kamera smartphone Anda untuk langsung mulai berinteraksi dengan @{botUsername}.
              </p>
            </div>

            {/* QR Code generator using quick visual SVG image service */}
            <div className="bg-white p-3 rounded-lg inline-block border border-gray-200">
              <img
                src={`https://api.qrserver.com/v1/create-qr-code/?size=180x180&data=${encodeURIComponent(botUrl)}`}
                alt="Telegram Bot QR Code"
                className="w-44 h-44 mx-auto block"
              />
            </div>

            <div className="pt-2">
              <a
                href={botUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-1.5 w-full py-2 rounded-md bg-[#D97706] hover:bg-[#B45309] text-white text-xs font-mono font-medium transition"
              >
                <span>Buka Langsung di Telegram Web / App</span>
                <ExternalLink className="w-3.5 h-3.5" />
              </a>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
