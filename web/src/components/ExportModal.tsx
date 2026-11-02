import React, { useState } from 'react'
import {
  Download,
  X,
  FileText,
  FileSpreadsheet,
  Printer,
  Copy,
  Check,
  Layers,
} from 'lucide-react'
import type { TrendingClusterItem } from '../types'

interface ExportModalProps {
  isOpen: boolean
  onClose: () => void
  clusters: TrendingClusterItem[]
}

export const ExportModal: React.FC<ExportModalProps> = ({
  isOpen,
  onClose,
  clusters,
}) => {
  const [copied, setCopied] = useState(false)

  if (!isOpen) return null

  // Generate Markdown Briefing Text
  const generateMarkdown = () => {
    const now = new Date().toLocaleString('id-ID', {
      day: 'numeric',
      month: 'long',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }) + ' WIB'

    let md = `# THE DISPATCH • EXECUTIVE INTELLIGENCE BRIEFING\n`
    md += `Tanggal Kompilasi: ${now}\n`
    md += `Platform: The Dispatch (https://thedispatch.id)\n`
    md += `Telegram Bot: @PengumpulBeritaBot\n`
    md += `Total Klaster Terverifikasi: ${clusters.length} Isu Panas\n\n`
    md += `---\n\n`

    clusters.forEach((c, idx) => {
      md += `## #${idx + 1}. ${c.title}\n`
      md += `- **Hotness Index:** ${c.hotness_score.toFixed(1)} / 100\n`
      md += `- **Volume Liputan:** ${c.article_count} artikel dari ${c.unique_outlets} redaksi media\n`
      md += `- **Portal Sumber:** ${c.outlets.join(', ')}\n`
      md += `- **Ringkasan Intelijen:** ${c.summary}\n\n`
    })

    md += `---\n`
    md += `*Laporan ini dihasilkan secara otomatis oleh The Dispatch Autonomous Topic Clustering Engine.*`
    return md
  }

  // Generate CSV text
  const generateCSV = () => {
    const headers = ['ID', 'Judul Klaster', 'Hotness Score', 'Jumlah Artikel', 'Jumlah Redaksi', 'Media Outlets', 'Ringkasan']
    const rows = clusters.map((c) => [
      c.id,
      `"${c.title.replace(/"/g, '""')}"`,
      c.hotness_score.toFixed(1),
      c.article_count,
      c.unique_outlets,
      `"${c.outlets.join(', ')}"`,
      `"${c.summary.replace(/"/g, '""')}"`,
    ])

    return [headers.join(','), ...rows.map((r) => r.join(','))].join('\n')
  }

  // Download File Utility
  const downloadFile = (content: string, filename: string, type: string) => {
    const blob = new Blob([content], { type })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
  }

  const handleDownloadMD = () => {
    const md = generateMarkdown()
    const dateStr = new Date().toISOString().split('T')[0]
    downloadFile(md, `the-dispatch-briefing-${dateStr}.md`, 'text/markdown;charset=utf-8')
  }

  const handleDownloadCSV = () => {
    const csv = generateCSV()
    const dateStr = new Date().toISOString().split('T')[0]
    downloadFile(csv, `the-dispatch-clusters-${dateStr}.csv`, 'text/csv;charset=utf-8')
  }

  const handleCopy = () => {
    const md = generateMarkdown()
    navigator.clipboard.writeText(md)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handlePrint = () => {
    window.print()
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-md flex items-center justify-center p-3 sm:p-4 animate-in fade-in duration-150">
      <div
        className="w-full max-w-2xl bg-slate-900 border border-slate-700/80 rounded-2xl shadow-2xl shadow-black/80 overflow-hidden flex flex-col max-h-[90vh]"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 sm:px-6 py-4 border-b border-slate-800 bg-slate-950/70">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400">
              <Download className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-editorial text-lg sm:text-xl font-bold text-slate-100">
                Executive Intelligence Export Suite
              </h3>
              <p className="text-xs text-slate-400 font-sans">
                Kompilasi laporan ringkasan topik dan dump dataset klaster untuk analitik eksternal
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

        {/* Content */}
        <div className="p-5 sm:p-6 overflow-y-auto space-y-6 text-sm text-slate-300 font-sans">
          {/* Quick Action Cards Grid */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4">
            {/* Markdown Report Card */}
            <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 hover:border-amber-500/40 transition group flex flex-col justify-between space-y-3">
              <div>
                <div className="flex items-center justify-between">
                  <div className="p-2 rounded-lg bg-amber-500/10 text-amber-400">
                    <FileText className="w-5 h-5" />
                  </div>
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                    Markdown (.md)
                  </span>
                </div>
                <h4 className="font-editorial text-base font-bold text-slate-100 mt-2">
                  Executive Briefing (.md)
                </h4>
                <p className="text-xs text-slate-400 mt-1 leading-relaxed">
                  Laporan naratif lengkap dengan ringkasan AI, Hotness score, dan daftar redaksi berita.
                </p>
              </div>
              <button
                type="button"
                onClick={handleDownloadMD}
                className="w-full py-2 px-3 rounded-lg bg-amber-500 hover:bg-amber-400 active:scale-95 text-slate-950 font-semibold text-xs font-mono transition flex items-center justify-center gap-1.5 shadow-sm"
              >
                <Download className="w-4 h-4" /> Unduh Markdown
              </button>
            </div>

            {/* CSV Dataset Card */}
            <div className="p-4 rounded-xl bg-slate-950 border border-slate-800 hover:border-emerald-500/40 transition group flex flex-col justify-between space-y-3">
              <div>
                <div className="flex items-center justify-between">
                  <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
                    <FileSpreadsheet className="w-5 h-5" />
                  </div>
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700">
                    Spreadsheet (.csv)
                  </span>
                </div>
                <h4 className="font-editorial text-base font-bold text-slate-100 mt-2">
                  Cluster Dataset (.csv)
                </h4>
                <p className="text-xs text-slate-400 mt-1 leading-relaxed">
                  Data tabular klaster siap impor untuk Excel, Google Sheets, Pandas, atau Tableau.
                </p>
              </div>
              <button
                type="button"
                onClick={handleDownloadCSV}
                className="w-full py-2 px-3 rounded-lg bg-emerald-500 hover:bg-emerald-400 active:scale-95 text-slate-950 font-semibold text-xs font-mono transition flex items-center justify-center gap-1.5 shadow-sm"
              >
                <Download className="w-4 h-4" /> Unduh CSV
              </button>
            </div>
          </div>

          {/* Quick Copy & Print Row */}
          <div className="flex flex-wrap items-center gap-3 pt-1">
            <button
              type="button"
              onClick={handleCopy}
              className="flex-1 py-2.5 px-4 rounded-xl bg-slate-800 hover:bg-slate-700 active:bg-slate-750 text-slate-200 border border-slate-700 transition flex items-center justify-center gap-2 text-xs font-mono"
            >
              {copied ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
              <span>{copied ? 'Tersalin ke Clipboard!' : 'Salin Teks Briefing'}</span>
            </button>

            <button
              type="button"
              onClick={handlePrint}
              className="flex-1 py-2.5 px-4 rounded-xl bg-slate-800 hover:bg-slate-700 active:bg-slate-750 text-slate-200 border border-slate-700 transition flex items-center justify-center gap-2 text-xs font-mono"
            >
              <Printer className="w-4 h-4 text-slate-400" />
              <span>Cetak / Simpan PDF</span>
            </button>
          </div>

          {/* Preview Snippet */}
          <div className="space-y-2">
            <span className="text-xs font-mono text-slate-400 flex items-center gap-1.5 font-semibold">
              <Layers className="w-4 h-4 text-amber-500" /> Pratinjau Briefing ({clusters.length} Klaster)
            </span>
            <div className="p-3.5 rounded-xl bg-slate-950 border border-slate-800 max-h-48 overflow-y-auto font-mono text-xs text-slate-300 space-y-2 leading-relaxed">
              {clusters.slice(0, 3).map((c, i) => (
                <div key={c.id} className="border-b border-slate-800/80 pb-2 last:border-0 last:pb-0">
                  <div className="flex items-center gap-2">
                    <span className="text-amber-400 font-bold">#{i + 1}</span>
                    <span className="text-slate-200 font-semibold truncate">{c.title}</span>
                  </div>
                  <p className="text-[11px] text-slate-400 line-clamp-2 mt-0.5">{c.summary}</p>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="px-5 sm:px-6 py-3.5 bg-slate-950 border-t border-slate-800 flex items-center justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-mono transition"
          >
            Tutup
          </button>
        </div>
      </div>
    </div>
  )
}
