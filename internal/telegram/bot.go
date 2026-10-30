package telegram

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"news-aggregator/internal/models"
	"news-aggregator/internal/repository"
	"news-aggregator/internal/service"
	"regexp"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

// BotService manages the Telegram bot interaction, message routing, and push notifications.
type BotService struct {
	bot          *tgbotapi.BotAPI
	newsService  service.NewsService
	repo         *repository.PostgresRepository
	systemAPIKey string
	enabled      bool
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

	systemKey := "telegram-bot-system-key"
	if repo != nil {
		_, _ = repo.EnsureSystemUser(systemKey, "telegram-bot@newsaggregator.internal", 999999)
	}

	return &BotService{
		bot:          bot,
		newsService:  newsSvc,
		repo:         repo,
		systemAPIKey: systemKey,
		enabled:      true,
	}, nil
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
• 🔍 <b>/search &lt;topik&gt;:</b> Cari arsip berita berbasis kesamaan makna semantik (pgvector)
• ⚖️ <b>/compare &lt;url1&gt; &lt;url2&gt;:</b> Analisis perbandingan sudut pandang & sentimen media
• ☀️ <b>/briefing:</b> Dapatkan rangkuman intisari berita utama saat ini
• 🔔 <b>/subscribe &lt;topik&gt;:</b> Langganan notifikasi ringkasan berita harian
• 🔕 <b>/unsubscribe:</b> Berhenti menerima notifikasi harian`,
			html.EscapeString(userName),
			html.EscapeString(b.bot.Self.UserName),
		)
		b.sendMessage(chatID, reply)

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
		topic := args
		if topic == "" {
			topic = "berita umum"
		}
		if b.repo != nil {
			_ = b.repo.SubscribeTelegram(chatID, topic)
		}
		b.sendMessage(chatID, fmt.Sprintf("✅ <b>Berhasil Berlangganan!</b>\nAnda akan menerima notifikasi ringkasan berita pilihan untuk topik: <b>%s</b>.", html.EscapeString(topic)))

	case "unsubscribe":
		if b.repo != nil {
			_ = b.repo.UnsubscribeTelegram(chatID)
		}
		b.sendMessage(chatID, "🔕 <b>Berhasil Berhenti Langganan.</b> Anda tidak akan menerima notifikasi ringkasan harian lagi. Ketik <code>/subscribe</code> kapan saja jika ingin kembali berlangganan.")

	case "briefing":
		b.handleBriefing(ctx, chatID)

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
		b.sendMessage(chatID, "🔍 Tidak ditemukan artikel yang cocok dengan konteks pencarian Anda.")
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

func (b *BotService) handleBriefing(ctx context.Context, chatID int64) {
	b.sendMessage(chatID, "☀️ <i>Menyusun Executive Morning Briefing untuk Anda...</i>")

	resp, err := b.newsService.GlobalQA(ctx, b.systemAPIKey, models.GlobalQARequest{
		Question: "Apa rangkuman intisari peristiwa berita paling penting dan terkini yang perlu diketahui?",
	})
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ Gagal menyusun briefing: %s", html.EscapeString(err.Error())))
		return
	}

	reply := fmt.Sprintf("☀️ <b>Executive Morning Briefing</b>\n📅 <i>%s</i>\n\n%s\n\n<i>Dihasilkan secara otomatis oleh @%s</i>",
		time.Now().Format("02 January 2006"),
		html.EscapeString(resp.Answer),
		html.EscapeString(b.bot.Self.UserName),
	)
	b.sendMessage(chatID, reply)
}

func (b *BotService) sendMessage(chatID int64, text string) {
	if !b.IsEnabled() {
		return
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = false

	if _, err := b.bot.Send(msg); err != nil {
		// Fallback to plain text if HTML tags fail to parse
		msg.ParseMode = ""
		msg.Text = stripHTML(text)
		_, _ = b.bot.Send(msg)
	}
}

func (b *BotService) sendChatAction(chatID int64, action string) {
	if !b.IsEnabled() {
		return
	}
	actionMsg := tgbotapi.NewChatAction(chatID, action)
	_, _ = b.bot.Send(actionMsg)
}

// PushNotification broadcasts a message to all active subscribers.
func (b *BotService) PushNotification(ctx context.Context, title, body string) error {
	if !b.IsEnabled() || b.repo == nil {
		return nil
	}

	subs, err := b.repo.GetActiveTelegramSubscriptions()
	if err != nil {
		return err
	}

	messageText := fmt.Sprintf("📢 <b>PEMBERITAHUAN BERITA</b>\n\n<b>%s</b>\n\n%s", html.EscapeString(title), html.EscapeString(body))

	for _, sub := range subs {
		b.sendMessage(sub.ChatID, messageText)
	}

	return nil
}

func stripHTML(s string) string {
	re := regexp.MustCompile("<[^>]*>")
	return re.ReplaceAllString(s, "")
}
