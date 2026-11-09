package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"news-aggregator/internal/clustering"
	"news-aggregator/internal/crawler"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"news-aggregator/internal/scheduler"
	"news-aggregator/internal/service"
	"news-aggregator/internal/strutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	urlRegex       = regexp.MustCompile(`https?://[^\s]+`)
	htmlTagRegex   = regexp.MustCompile(`<[^>]*>`)
	floodWaitRegex = regexp.MustCompile(`(?i)(?:retry after|flood_wait_?)\s*(\d+)`)
)

// ParseTelegramFloodWait determines if an error is a Telegram 429 Flood Limit / Too Many Requests,
// and extracts the recommended backoff duration in seconds.
func ParseTelegramFloodWait(err error) (int, bool) {
	if err == nil {
		return 0, false
	}

	// 1. Direct type assertion / errors.As to *tgbotapi.Error
	var tgErr *tgbotapi.Error
	if errors.As(err, &tgErr) {
		if tgErr.Code == 429 || tgErr.Code == 420 || tgErr.RetryAfter > 0 {
			waitSec := tgErr.RetryAfter
			if waitSec <= 0 {
				waitSec = extractSecondsFromMessage(tgErr.Message)
			}
			if waitSec <= 0 {
				waitSec = 3 // Safe minimum fallback backoff
			}
			return waitSec, true
		}
	}

	// 2. Fallback check from error message text
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "too many requests") || strings.Contains(errMsg, "retry after") || strings.Contains(errMsg, "flood") {
		waitSec := extractSecondsFromMessage(err.Error())
		if waitSec <= 0 {
			waitSec = 3
		}
		return waitSec, true
	}

	return 0, false
}

func extractSecondsFromMessage(msg string) int {
	matches := floodWaitRegex.FindStringSubmatch(msg)
	if len(matches) > 1 {
		if sec, err := strconv.Atoi(matches[1]); err == nil && sec > 0 {
			return sec
		}
	}
	return 0
}

// BotService manages the Telegram bot interaction, message routing, and push notifications.
type BotService struct {
	bot          *tgbotapi.BotAPI
	newsService  service.NewsService
	repo         *repository.PostgresRepository
	crawler      *crawler.HarvesterEngine
	scheduler    *scheduler.BriefingEngine
	clustering   *clustering.ClusterEngine
	systemAPIKey string
	enabled      bool
	broadcastMu  sync.Mutex
}

// NewBotService initializes the Telegram bot client. If token is empty, returns a disabled service gracefully.
func NewBotService(token string, newsSvc service.NewsService, repo *repository.PostgresRepository) (*BotService, error) {
	if strings.TrimSpace(token) == "" {
		slog.Info("Telegram Bot token not provided, Telegram service disabled")
		return &BotService{enabled: false}, nil
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		slog.Warn("Failed to initialize Telegram Bot API", slog.String("error", err.Error()))
		return nil, fmt.Errorf("telegram bot initialization error: %w", err)
	}

	slog.Info("Telegram Bot successfully authorized", slog.String("username", bot.Self.UserName))

	systemKey := strings.TrimSpace(os.Getenv("TELEGRAM_SYSTEM_KEY"))
	if systemKey == "" {
		systemKey = strings.TrimSpace(os.Getenv("TELEGRAM_BOT_SYSTEM_KEY"))
	}
	if systemKey == "" {
		systemKey = generateSecureToken("telegram")
	}

	if repo != nil {
		_, _ = repo.EnsureSystemUser(systemKey, "telegram-bot@newsaggregator.internal", 999999)
	}

	botSvc := &BotService{
		bot:          bot,
		newsService:  newsSvc,
		repo:         repo,
		systemAPIKey: systemKey,
		enabled:      true,
	}

	botSvc.registerBotCommands()

	return botSvc, nil
}

// registerBotCommands sets the command autocomplete list and menu button in Telegram.
func (b *BotService) registerBotCommands() {
	if !b.IsEnabled() {
		return
	}

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Mulai bot & panduan fitur"},
		{Command: "help", Description: "Bantuan & daftar perintah"},
		{Command: "trending", Description: "Topik berita paling hangat hari ini (Radar Isu)"},
		{Command: "cluster", Description: "Rincian kumpulan artikel dalam satu isu"},
		{Command: "sync", Description: "Sinkronisasi berita terbaru dari RSS feeds"},
		{Command: "search", Description: "Cari berita arsip (pgvector / kata kunci)"},
		{Command: "briefing", Description: "Rangkuman berita harian (Executive Digest)"},
		{Command: "compare", Description: "Bandingkan sudut pandang & sentimen media"},
		{Command: "subscribe", Description: "Langganan ringkasan harian (07:00 & 17:00 WIB)"},
		{Command: "subscriptions", Description: "Cek daftar topik langganan aktif Anda"},
		{Command: "unsubscribe", Description: "Berhenti langganan notifikasi harian"},
	}

	cmdConfig := tgbotapi.NewSetMyCommands(commands...)
	if _, err := b.bot.Request(cmdConfig); err != nil {
		slog.Warn("Failed to register Telegram bot command menu", slog.String("error", err.Error()))
	} else {
		slog.Info("Telegram bot command menu successfully registered")
	}
}

// WithCrawler injects the HarvesterEngine reference to allow /sync command.
func (b *BotService) WithCrawler(c *crawler.HarvesterEngine) *BotService {
	b.crawler = c
	return b
}

// WithScheduler injects the BriefingEngine reference for scheduled broadcasts and on-demand briefings.
func (b *BotService) WithScheduler(s *scheduler.BriefingEngine) *BotService {
	b.scheduler = s
	return b
}

// WithClustering injects the ClusterEngine reference for trending news detection.
func (b *BotService) WithClustering(c *clustering.ClusterEngine) *BotService {
	b.clustering = c
	return b
}

// sendWithRetry attempts to dispatch a message with automatic backoff retry on Telegram 429 flood wait errors.
func (b *BotService) sendWithRetry(ctx context.Context, c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	const maxRetries = 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		msg, err := b.bot.Send(c)
		if err == nil {
			return msg, nil
		}

		waitSec, isFlood := ParseTelegramFloodWait(err)
		if isFlood && attempt < maxRetries {
			if waitSec > 60 {
				waitSec = 60 // Cap backoff at 60 seconds
			}
			slog.Warn("Telegram API flood limit encountered, backing off before retry",
				slog.Int("attempt", attempt+1),
				slog.Int("wait_seconds", waitSec),
				slog.String("error", err.Error()),
			)
			select {
			case <-time.After(time.Duration(waitSec) * time.Second):
				continue
			case <-ctx.Done():
				return msg, ctx.Err()
			}
		}

		return msg, err
	}
	return tgbotapi.Message{}, fmt.Errorf("failed after retries")
}

// SendNotification dispatches an HTML formatted notification to a specific Telegram chat ID with chunking and flood control.
func (b *BotService) SendNotification(chatID int64, htmlContent string) error {
	if !b.IsEnabled() {
		return fmt.Errorf("telegram bot is not enabled")
	}

	chunks := SplitTelegramMessage(htmlContent, DefaultMaxTelegramMessageRunes)
	for i, chunk := range chunks {
		msg := tgbotapi.NewMessage(chatID, chunk)
		msg.ParseMode = "HTML"
		msg.DisableWebPagePreview = false

		if _, err := b.sendWithRetry(context.Background(), msg); err != nil {
			if _, isFlood := ParseTelegramFloodWait(err); isFlood {
				return err
			}
			// Fallback to plain text for this chunk if HTML entity parsing fails
			msg.ParseMode = ""
			msg.Text = stripHTML(chunk)
			if _, plainErr := b.sendWithRetry(context.Background(), msg); plainErr != nil {
				return plainErr
			}
		}

		if i < len(chunks)-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}
	return nil
}

// IsEnabled returns true if the Telegram bot service is active.
func (b *BotService) IsEnabled() bool {
	return b != nil && b.enabled && b.bot != nil
}

// Start launches the Telegram long polling listener in a non-blocking background goroutine.
func (b *BotService) Start(ctx context.Context) {
	if !b.IsEnabled() {
		return
	}

	slog.Info("Starting Telegram Bot long-polling listener...", slog.String("bot", b.bot.Self.UserName))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Stopping Telegram Bot listener...")
				b.bot.StopReceivingUpdates()
				return
			case update, ok := <-updates:
				if !ok {
					return
				}
				if update.Message == nil {
					continue
				}
				go b.handleMessage(ctx, update.Message)
			}
		}
	}()
}

func (b *BotService) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return
	}

	// Send typing action indicator
	b.sendChatAction(chatID, tgbotapi.ChatTyping)

	// 1. Handle Commands
	if msg.IsCommand() {
		b.handleCommand(ctx, msg)
		return
	}

	// 2. Handle News URLs in message
	urls := urlRegex.FindAllString(text, -1)
	if len(urls) > 0 {
		b.handleURLs(ctx, chatID, urls)
		return
	}

	// 3. Freeform Natural Language Question (Global RAG Q&A)
	b.handleGlobalQA(ctx, chatID, text)
}

func (b *BotService) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	command := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())

	switch command {
	case "start", "help":
		userName := msg.From.FirstName
		if userName == "" {
			userName = "Sahabat Berita"
		}
		reply := fmt.Sprintf(`👋 <b>Halo, %s!</b> Selamat datang di <b>@%s</b>.

Saya adalah asisten AI berita cerdas berbasis <b>Concurrent AI News Aggregator</b>.

✨ <b>Fitur yang dapat Anda gunakan langsung:</b>
• 💬 <b>Tanya Bebas:</b> Ketik pertanyaan seputar berita apa saja (contoh: <i>"Apa saja persiapan pengamanan demo di Jakarta?"</i>)
• 📰 <b>Ringkas Link Berita:</b> Tempelkan langsung tautan artikel berita apa saja
• 🔥 <b>/trending:</b> Radar isu & topik berita paling hangat hari ini
• 📌 <b>/cluster &lt;id&gt;:</b> Baca rincian kumpulan artikel dalam satu kluster isu
• 🔄 <b>/sync:</b> Sinkronisasi dan arsipkan berita terkini otomatis dari Antara, CNN, CNBC, dll.
• 🔍 <b>/search &lt;topik&gt;:</b> Cari arsip berita berbasis kesamaan makna semantik (pgvector)
• ⚖️ <b>/compare &lt;url1&gt; &lt;url2&gt;:</b> Analisis perbandingan sudut pandang & sentimen media
• ☀️ <b>/briefing [topik]:</b> Dapatkan rangkuman intisari berita utama saat ini
• 🔔 <b>/subscribe &lt;topik&gt;:</b> Langganan notifikasi ringkasan berita harian (contoh: <code>/subscribe teknologi</code>)
• 📋 <b>/subscriptions:</b> Cek daftar topik langganan aktif Anda
• 🔕 <b>/unsubscribe [topik]:</b> Berhenti menerima notifikasi harian`,
			html.EscapeString(userName),
			html.EscapeString(b.bot.Self.UserName),
		)
		b.sendMessage(chatID, reply)

	case "trending":
		b.handleTrending(ctx, chatID)

	case "cluster":
		b.handleCluster(ctx, chatID, args)

	case "sync", "crawl":
		if b.crawler == nil {
			b.sendMessage(chatID, "⚠️ <b>Fitur sinkronisasi berita belum diaktifkan.</b>")
			return
		}
		b.sendMessage(chatID, "⏳ <i>Sedang mengambil dan mengindeks berita terbaru dari berbagai media terkemuka...</i>")
		go func() {
			res, err := b.crawler.CrawlAll(context.Background())
			if err != nil {
				b.sendMessage(chatID, fmt.Sprintf("❌ <b>Gagal sinkronisasi:</b> %s", html.EscapeString(err.Error())))
				return
			}
			msg := fmt.Sprintf("✅ <b>Sinkronisasi Berita Selesai!</b>\n\n"+
				"📥 <b>Berita Diperiksa:</b> %d artikel\n"+
				"🆕 <b>Berita Baru Diarsip:</b> %d artikel\n"+
				"⏭️ <b>Dilewati (Sudah Ada):</b> %d artikel\n"+
				"⏱️ <b>Waktu Proses:</b> %s\n\n"+
				"💡 <i>Gunakan <code>/search &lt;kata kunci&gt;</code> atau ajukan pertanyaan bebas seputar berita tersebut langsung di sini!</i>",
				res.Fetched, res.Indexed, res.Skipped, res.Duration.Round(time.Millisecond))
			b.sendMessage(chatID, msg)
		}()

	case "search":
		if args == "" {
			b.sendMessage(chatID, "⚠️ <b>Format salah.</b> Gunakan: <code>/search &lt;kata kunci&gt;</code>\nContoh: <code>/search pengamanan demo jakarta</code>")
			return
		}
		b.handleSearch(ctx, chatID, args)

	case "compare":
		urls := urlRegex.FindAllString(args, -1)
		if len(urls) < 2 {
			b.sendMessage(chatID, "⚠️ <b>Format salah.</b> Masukkan minimal 2 URL berita untuk dibandingkan.\nContoh:\n<code>/compare https://kompas.com/... https://detik.com/...</code>")
			return
		}
		b.handleCompare(ctx, chatID, urls)

	case "subscribe":
		topic := strings.TrimSpace(strings.ToLower(args))
		if topic == "" {
			topic = "berita"
		}
		if b.repo != nil {
			_ = b.repo.SubscribeTelegram(chatID, topic)
		}
		b.sendMessage(chatID, fmt.Sprintf("✅ <b>Berhasil Berlangganan!</b>\nAnda akan menerima notifikasi ringkasan berita berkala (07:00 & 17:00 WIB) untuk topik: <b>%s</b>.\n\nKetik <code>/subscriptions</code> untuk melihat daftar topik yang Anda ikuti.", html.EscapeString(topic)))

	case "subscriptions", "status", "mystatus":
		b.handleSubscriptions(chatID)

	case "unsubscribe":
		if b.repo == nil {
			b.sendMessage(chatID, "⚠️ Layanan database langganan tidak tersedia.")
			return
		}
		topic := strings.TrimSpace(strings.ToLower(args))
		if topic == "" || topic == "all" || topic == "semua" {
			_ = b.repo.UnsubscribeTelegram(chatID)
			b.sendMessage(chatID, "🔕 <b>Berhasil Berhenti Langganan.</b> Anda tidak akan menerima notifikasi ringkasan harian lagi. Ketik <code>/subscribe &lt;topik&gt;</code> kapan saja jika ingin kembali berlangganan.")
		} else {
			_ = b.repo.UnsubscribeTelegramTopic(chatID, topic)
			b.sendMessage(chatID, fmt.Sprintf("🔕 <b>Berhasil Berhenti Langganan Topik:</b> <code>%s</code>.\nKetik <code>/subscriptions</code> untuk melihat sisa topik aktif Anda.", html.EscapeString(topic)))
		}

	case "briefing":
		b.handleBriefing(ctx, chatID, args)

	default:
		b.sendMessage(chatID, "❓ Perintah tidak dikenali. Ketik <code>/help</code> untuk melihat daftar perintah yang tersedia.")
	}
}

func (b *BotService) handleURLs(ctx context.Context, chatID int64, urls []string) {
	if len(urls) == 1 {
		b.sendMessage(chatID, "⏳ <i>Sedang membaca dan merangkum artikel berita...</i>")

		results, err := b.newsService.SummarizeArticles(ctx, b.systemAPIKey, []string{urls[0]})
		if err != nil {
			b.sendMessage(chatID, fmt.Sprintf("❌ <b>Gagal merangkum berita:</b> %s", html.EscapeString(err.Error())))
			return
		}

		if len(results) == 0 {
			b.sendMessage(chatID, "❌ Tidak dapat mengekstrak isi artikel dari tautan yang diberikan.")
			return
		}

		res := results[0]
		reply := fmt.Sprintf("📰 <b>%s</b>\n\n🌐 <b>Sumber:</b> %s\n\n📝 <b>Ringkasan AI (3 Kalimat):</b>\n%s\n\n🔗 <a href=\"%s\">Buka Artikel Asli</a>",
			html.EscapeString(res.Title),
			html.EscapeString(res.Source),
			html.EscapeString(res.Summary),
			res.URL,
		)
		b.sendMessage(chatID, reply)
		return
	}

	// Multiple URLs detected
	b.sendMessage(chatID, "⏳ <i>Sedang membandingkan sudut pandang media dari beberapa tautan...</i>")
	b.handleCompare(ctx, chatID, urls)
}

func (b *BotService) handleGlobalQA(ctx context.Context, chatID int64, question string) {
	resp, err := b.newsService.GlobalQA(ctx, b.systemAPIKey, models.GlobalQARequest{Question: question})
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ <b>Kendala Pemrosesan:</b> %s", html.EscapeString(err.Error())))
		return
	}

	var sb strings.Builder
	sb.WriteString("💡 <b>Jawaban AI:</b>\n")
	sb.WriteString(html.EscapeString(resp.Answer))
	sb.WriteString("\n")

	if len(resp.Sources) > 0 {
		sb.WriteString("\n📚 <b>Sumber Berita Terkait:</b>\n")
		for i, src := range resp.Sources {
			sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a> <i>(Relevansi: %.0f%%)</i>\n",
				i+1,
				src.URL,
				html.EscapeString(src.Title),
				src.Similarity*100,
			))
		}
	} else {
		sb.WriteString("\n💡 <i>Tips: Tempelkan (paste) tautan berita terkait (Kompas, Detik, dll.) atau ketik <code>/sync</code> untuk mengarsipkan berita terbaru secara otomatis!</i>\n")
	}

	sb.WriteString(fmt.Sprintf("\n<i>Penyedia AI: %s</i>", html.EscapeString(resp.Provider)))
	b.sendMessage(chatID, sb.String())
}

func (b *BotService) handleSearch(ctx context.Context, chatID int64, query string) {
	results, err := b.newsService.SemanticSearch(ctx, query, 3)
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ <b>Gagal mencari berita:</b> %s", html.EscapeString(err.Error())))
		return
	}

	if len(results) == 0 {
		b.sendMessage(chatID, "🔍 <b>Tidak ditemukan artikel yang cocok di arsip berita.</b>\n\n💡 <i>Tips: Ketik <code>/sync</code> untuk mengarsipkan berita terkini dari berbagai media atau tempelkan link berita langsung ke chat ini!</i>")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 <b>Hasil Pencarian Semantik untuk:</b> <i>%s</i>\n\n", html.EscapeString(query)))
	for i, res := range results {
		sb.WriteString(fmt.Sprintf("<b>%d. %s</b>\n", i+1, html.EscapeString(res.Title)))
		sb.WriteString(fmt.Sprintf("🌐 Sumber: %s | Relevansi: %.1f%%\n", html.EscapeString(res.Source), res.Similarity*100))
		sb.WriteString(fmt.Sprintf("📝 %s\n", html.EscapeString(res.Summary)))
		sb.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">Baca Selengkapnya</a>\n\n", res.URL))
	}

	b.sendMessage(chatID, sb.String())
}

func (b *BotService) handleCompare(ctx context.Context, chatID int64, urls []string) {
	report, err := b.newsService.CompareArticles(ctx, b.systemAPIKey, models.NewsCompareRequest{URLs: urls})
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ <b>Gagal membandingkan berita:</b> %s", html.EscapeString(err.Error())))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚖️ <b>Analisis Bias & Perbandingan Media</b>\n\n📌 <b>Topik:</b> %s\n\n", html.EscapeString(report.Topic)))

	if len(report.ConsensusFacts) > 0 {
		sb.WriteString("🤝 <b>Fakta Konsensus (Disepakati Bersama):</b>\n")
		for _, fact := range report.ConsensusFacts {
			sb.WriteString(fmt.Sprintf("• %s\n", html.EscapeString(fact)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("📊 <b>Sudut Pandang per Media:</b>\n")
	for _, m := range report.MediaComparisons {
		sb.WriteString(fmt.Sprintf("• <b>%s</b> (Sentimen: <i>%s</i>, Skor: <code>%.2f</code>)\n", html.EscapeString(m.OutletName), html.EscapeString(m.Sentiment), m.SentimentScore))
		sb.WriteString(fmt.Sprintf("  <i>Penekanan:</i> %s\n", html.EscapeString(m.DominantFraming)))
	}
	sb.WriteString("\n")

	if len(report.Divergences) > 0 {
		sb.WriteString("⚡ <b>Perbedaan / Kontradiksi Narasi:</b>\n")
		for _, d := range report.Divergences {
			sb.WriteString(fmt.Sprintf("• %s\n", html.EscapeString(d)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("📝 <b>Sintesis Objektif:</b>\n%s\n\n", html.EscapeString(report.ObjectiveSummary)))
	sb.WriteString(fmt.Sprintf("<i>Analisis oleh: %s</i>", html.EscapeString(report.Provider)))

	b.sendMessage(chatID, sb.String())
}

func (b *BotService) handleSubscriptions(chatID int64) {
	if b.repo == nil {
		b.sendMessage(chatID, "⚠️ Layanan database langganan tidak tersedia.")
		return
	}

	subs, err := b.repo.GetChatSubscriptions(chatID)
	if err != nil || len(subs) == 0 {
		b.sendMessage(chatID, "📭 <b>Anda belum memiliki langganan aktif.</b>\n\n"+
			"Ketik <code>/subscribe &lt;topik&gt;</code> untuk mulai menerima ringkasan berita harian otomatis!\n\n"+
			"💡 <b>Rekomendasi Topik:</b>\n"+
			"• <code>/subscribe teknologi</code>\n"+
			"• <code>/subscribe ekonomi</code>\n"+
			"• <code>/subscribe politik</code>\n"+
			"• <code>/subscribe berita</code> (semua berita)",
		)
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 <b>Daftar Topik Langganan Aktif Anda:</b>\n\n")
	for i, s := range subs {
		sb.WriteString(fmt.Sprintf("%d. 🔔 <b>%s</b>\n", i+1, html.EscapeString(strings.ToUpper(s.Topic))))
	}
	sb.WriteString("\n⏰ <i>Jadwal Push Notification Otomatis: 07:00 & 17:00 WIB.</i>\n\n")
	sb.WriteString("💡 <i>Ketik <code>/unsubscribe &lt;topik&gt;</code> untuk berhenti langganan topik tertentu, atau <code>/unsubscribe</code> untuk berhenti semua.</i>")

	b.sendMessage(chatID, sb.String())
}

func (b *BotService) handleBriefing(ctx context.Context, chatID int64, topic string) {
	topic = strings.TrimSpace(topic)
	if topic != "" && b.scheduler != nil {
		b.sendMessage(chatID, fmt.Sprintf("☀️ <i>Menyusun Executive Briefing topik: %s...</i>", html.EscapeString(topic)))
		msg, err := b.scheduler.SynthesizeBriefing(ctx, topic)
		if err == nil && msg != "" {
			b.sendMessage(chatID, msg)
			return
		}
	}

	b.sendMessage(chatID, "☀️ <i>Menyusun Executive Morning Briefing untuk Anda...</i>")

	question := "Apa rangkuman intisari peristiwa berita paling penting dan terkini yang perlu diketahui?"
	if topic != "" {
		question = fmt.Sprintf("Apa rangkuman intisari peristiwa berita terkini seputar %s?", topic)
	}

	resp, err := b.newsService.GlobalQA(ctx, b.systemAPIKey, models.GlobalQARequest{
		Question: question,
	})
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ Gagal menyusun briefing: %s", html.EscapeString(err.Error())))
		return
	}

	loc := time.FixedZone("WIB", 7*3600)
	if b.scheduler != nil && b.scheduler.Location() != nil {
		loc = b.scheduler.Location()
	}
	formattedDate := scheduler.FormatIndonesianDate(time.Now().In(loc))

	reply := fmt.Sprintf("☀️ <b>Executive Morning Briefing</b>\n📅 <i>%s</i>\n\n%s\n\n<i>Dihasilkan secara otomatis oleh @%s</i>",
		formattedDate,
		html.EscapeString(resp.Answer),
		html.EscapeString(b.bot.Self.UserName),
	)
	b.sendMessage(chatID, reply)
}

func (b *BotService) handleTrending(ctx context.Context, chatID int64) {
	if b.clustering == nil {
		b.sendMessage(chatID, "⚠️ <b>Fitur Topic Clustering & Trending Radar belum diaktifkan.</b>")
		return
	}

	b.sendChatAction(chatID, tgbotapi.ChatTyping)

	items, err := b.clustering.GetTrending(ctx, 5)
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ <b>Gagal mengambil radar tren:</b> %s", html.EscapeString(err.Error())))
		return
	}

	if len(items) == 0 {
		b.sendMessage(chatID, "ℹ️ <b>Belum ada kluster topik tren saat ini.</b>\nSilakan jalankan perintah <code>/sync</code> untuk mengumpulkan berita terbaru terlebih dahulu.")
		return
	}

	var sb strings.Builder
	sb.WriteString("🔥 <b>Radar Topik Berita Paling Tren Saat Ini</b>\n")
	sb.WriteString("<i>Dianalisis secara objektif dari liputan lintas media massa</i>\n\n")

	for i, item := range items {
		outletsStr := strings.Join(item.Outlets, ", ")
		if outletsStr == "" {
			outletsStr = "Media Terkemuka"
		}

		shortSummary := item.Summary
		if utf8.RuneCountInString(shortSummary) > 120 {
			shortSummary = strutil.TruncateRunes(shortSummary, 117) + "..."
		}

		sb.WriteString(fmt.Sprintf("<b>%d. %s</b>\n", i+1, html.EscapeString(item.Title)))
		sb.WriteString(fmt.Sprintf("   📊 <b>Liputan:</b> %d artikel dari %d media (<i>%s</i>)\n", item.ArticleCount, item.UniqueOutlets, html.EscapeString(outletsStr)))
		sb.WriteString(fmt.Sprintf("   🔥 <b>Skor Tren:</b> <code>%.1f</code>\n", item.HotnessScore))
		if shortSummary != "" {
			sb.WriteString(fmt.Sprintf("   💡 <i>\"%s\"</i>\n", html.EscapeString(shortSummary)))
		}
		sb.WriteString(fmt.Sprintf("   👉 Baca detail kluster: <code>/cluster %d</code>\n\n", item.ID))
	}

	sb.WriteString("💡 <i>Ketik <code>/cluster &lt;id&gt;</code> untuk melihat seluruh sumber artikel dan analisis konsensus isu terkait.</i>")

	b.sendMessage(chatID, sb.String())
}

func (b *BotService) handleCluster(ctx context.Context, chatID int64, args string) {
	if b.clustering == nil {
		b.sendMessage(chatID, "⚠️ <b>Fitur Topic Clustering & Trending Radar belum diaktifkan.</b>")
		return
	}

	args = strings.TrimSpace(args)
	if args == "" {
		b.sendMessage(chatID, "ℹ️ <b>Mohon sertakan ID kluster topik.</b>\nContoh: <code>/cluster 1</code>\n\nKetik <code>/trending</code> untuk melihat daftar topik yang sedang tren saat ini.")
		return
	}

	id, err := strconv.ParseUint(args, 10, 64)
	if err != nil {
		b.sendMessage(chatID, "❌ <b>ID kluster harus berupa angka bulat positif.</b>\nContoh: <code>/cluster 1</code>")
		return
	}

	b.sendChatAction(chatID, tgbotapi.ChatTyping)

	detail, err := b.clustering.GetClusterDetail(ctx, uint(id))
	if err != nil || detail == nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ <b>Kluster dengan ID #%d tidak ditemukan atau sudah kedaluwarsa.</b>\nKetik <code>/trending</code> untuk melihat topik yang sedang aktif.", id))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📌 <b>Kluster Isu Berita #%d</b>\n\n", detail.ID))
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n\n", html.EscapeString(detail.Title)))
	sb.WriteString(fmt.Sprintf("📊 <b>Cakupan Liputan:</b> %d artikel dari %d media\n", detail.ArticleCount, detail.UniqueOutlets))
	sb.WriteString(fmt.Sprintf("📰 <b>Media Meliput:</b> %s\n", html.EscapeString(strings.Join(detail.Outlets, ", "))))
	sb.WriteString(fmt.Sprintf("🔥 <b>Skor Tren:</b> <code>%.1f</code> | 🕒 <b>Terakhir:</b> %s\n\n", detail.HotnessScore, detail.LastUpdated.Format("02 Jan 2006 15:04")))

	if detail.Summary != "" {
		sb.WriteString("💡 <b>Konsensus & Intisari Isu:</b>\n")
		sb.WriteString(fmt.Sprintf("<i>%s</i>\n\n", html.EscapeString(detail.Summary)))
	}

	sb.WriteString("📚 <b>Daftar Artikel Terkait:</b>\n")
	for i, art := range detail.Articles {
		src := art.Source
		if src == "" {
			src = "Sumber Berita"
		}
		sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a> (<i>%s</i>)\n", i+1, art.URL, html.EscapeString(art.Title), html.EscapeString(src)))
	}

	if len(detail.Articles) >= 2 {
		sb.WriteString(fmt.Sprintf("\n⚖️ <b>Bandingkan Sudut Pandang Media:</b>\n<code>/compare %s %s</code>", detail.Articles[0].URL, detail.Articles[1].URL))
	}

	b.sendMessage(chatID, sb.String())
}

func (b *BotService) sendMessage(chatID int64, text string) {
	if !b.IsEnabled() {
		return
	}

	chunks := SplitTelegramMessage(text, DefaultMaxTelegramMessageRunes)
	for i, chunk := range chunks {
		msg := tgbotapi.NewMessage(chatID, chunk)
		msg.ParseMode = "HTML"
		msg.DisableWebPagePreview = false

		if _, err := b.sendWithRetry(context.Background(), msg); err != nil {
			// If error is flood limit, avoid hammering Telegram with repeated fallback messages
			if _, isFlood := ParseTelegramFloodWait(err); isFlood {
				slog.Warn("Failed to send Telegram message due to persistent flood limit",
					slog.Int64("chat_id", chatID),
					slog.String("error", err.Error()),
				)
				return
			}

			// Fallback to plain text for this chunk if HTML tags fail to parse
			msg.ParseMode = ""
			msg.Text = stripHTML(chunk)
			_, _ = b.sendWithRetry(context.Background(), msg)
		}

		if i < len(chunks)-1 {
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (b *BotService) sendChatAction(chatID int64, action string) {
	if !b.IsEnabled() {
		return
	}
	actionMsg := tgbotapi.NewChatAction(chatID, action)
	_, _ = b.sendWithRetry(context.Background(), actionMsg)
}

// PushNotification broadcasts a message to all active subscribers with deduplication, rate limiting, and flood control.
func (b *BotService) PushNotification(ctx context.Context, title, body string) error {
	if !b.IsEnabled() || b.repo == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	b.broadcastMu.Lock()
	defer b.broadcastMu.Unlock()

	subs, err := b.repo.GetActiveTelegramSubscriptions()
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		return nil
	}

	// 1. Deduplicate by ChatID to avoid multiple messages to multi-topic subscribers
	seenChats := make(map[int64]struct{}, len(subs))
	var uniqueChatIDs []int64
	for _, sub := range subs {
		if _, seen := seenChats[sub.ChatID]; !seen {
			seenChats[sub.ChatID] = struct{}{}
			uniqueChatIDs = append(uniqueChatIDs, sub.ChatID)
		}
	}

	messageText := fmt.Sprintf("📢 <b>PEMBERITAHUAN BERITA</b>\n\n<b>%s</b>\n\n%s", html.EscapeString(title), html.EscapeString(body))

	// 2. Safe rate-limited dispatch (25 messages/second = 40ms interval, well below Telegram's 30/sec limit)
	const delayBetweenMessages = 40 * time.Millisecond

	for _, chatID := range uniqueChatIDs {
		// Respect context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := b.SendNotification(chatID, messageText)
		if err != nil {
			errMsg := err.Error()
			lowerErr := strings.ToLower(errMsg)

			// Auto-deactivate unreachable or blocked chats
			if strings.Contains(lowerErr, "blocked") || strings.Contains(lowerErr, "not found") || strings.Contains(lowerErr, "deactivated") || strings.Contains(lowerErr, "forbidden") {
				slog.Info("Auto-deactivating subscription for unreachable chat", slog.Int64("chat_id", chatID), slog.String("reason", errMsg))
				_ = b.repo.DeactivateTelegramChat(chatID)
			} else if waitSec, isFlood := ParseTelegramFloodWait(err); isFlood {
				slog.Warn("Telegram flood wait in PushNotification, pausing broadcast", slog.Int("wait_seconds", waitSec), slog.Int64("chat_id", chatID))
				select {
				case <-time.After(time.Duration(waitSec) * time.Second):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}

		time.Sleep(delayBetweenMessages)
	}

	return nil
}

func stripHTML(s string) string {
	return htmlTagRegex.ReplaceAllString(s, "")
}

func generateSecureToken(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}
