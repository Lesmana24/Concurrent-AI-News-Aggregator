package scheduler

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"news-aggregator/internal/ai"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var floodWaitRegex = regexp.MustCompile(`(?i)(?:retry after|flood_wait_?)\s*(\d+)`)

// parseFloodWait extracts the flood wait duration in seconds from an error message if it is a 429/flood limit.
func parseFloodWait(errMsg string) (int, bool) {
	lower := strings.ToLower(errMsg)
	if strings.Contains(lower, "too many requests") || strings.Contains(lower, "retry after") || strings.Contains(lower, "flood") || strings.Contains(lower, "429") {
		matches := floodWaitRegex.FindStringSubmatch(errMsg)
		if len(matches) > 1 {
			if sec, err := strconv.Atoi(matches[1]); err == nil && sec > 0 {
				return sec, true
			}
		}
		return 3, true // Default safe minimum backoff
	}
	return 0, false
}

// indonesianMonths provides Indonesian translation for month names.
var indonesianMonths = [...]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// FormatIndonesianDate formats a time.Time into an idiomatic Indonesian date string (e.g. "09 November 2026").
func FormatIndonesianDate(t time.Time) string {
	m := int(t.Month())
	if m >= 1 && m <= 12 {
		return fmt.Sprintf("%02d %s %d", t.Day(), indonesianMonths[m], t.Year())
	}
	return t.Format("02 January 2006")
}

// ResolveLocation resolves a timezone string into a *time.Location,
// with robust fallbacks for Indonesian timezones (WIB, WITA, WIT) and UTC offsets
// even on minimal container environments lacking tzdata.
func ResolveLocation(tz string) *time.Location {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	// 1. Standard IANA timezone lookup
	loc, err := time.LoadLocation(tz)
	if err == nil && loc != nil {
		return loc
	}

	// 2. Fallbacks for Indonesian timezones if tzdata is not available on host/container
	switch strings.ToUpper(tz) {
	case "ASIA/JAKARTA", "WIB", "UTC+7", "+07:00", "+0700", "+07":
		return time.FixedZone("WIB", 7*3600)
	case "ASIA/MAKASSAR", "WITA", "UTC+8", "+08:00", "+0800", "+08":
		return time.FixedZone("WITA", 8*3600)
	case "ASIA/JAYAPURA", "WIT", "UTC+9", "+09:00", "+0900", "+09":
		return time.FixedZone("WIT", 9*3600)
	case "UTC", "GMT", "Z":
		return time.UTC
	}

	// Default fallback to WIB (UTC+7) for Indonesian national news
	return time.FixedZone("WIB", 7*3600)
}

// BriefingEngine orchestrates scheduled AI news syntheses and rate-limited push notifications.
type BriefingEngine struct {
	cfg          Config
	repo         BriefingRepository
	aiSvc        ai.AIService
	sender       TelegramSender
	location     *time.Location
	stats        BroadcastStats
	running      bool
	lastRunDates map[string]bool
	mu           sync.RWMutex
}

// NewBriefingEngine instantiates a new BriefingEngine.
func NewBriefingEngine(cfg Config, repo BriefingRepository, aiSvc ai.AIService, sender TelegramSender) *BriefingEngine {
	if cfg.MaxArticlesPerDigest <= 0 {
		cfg.MaxArticlesPerDigest = 5
	}
	if cfg.BroadcastRateLimitPerSec <= 0 || cfg.BroadcastRateLimitPerSec > 30 {
		cfg.BroadcastRateLimitPerSec = 25
	}
	if len(cfg.DailyHours) == 0 {
		cfg.DailyHours = []string{"07:00", "17:00"}
	}
	if strings.TrimSpace(cfg.Timezone) == "" {
		cfg.Timezone = "Asia/Jakarta"
	}

	loc := ResolveLocation(cfg.Timezone)

	return &BriefingEngine{
		cfg:          cfg,
		repo:         repo,
		aiSvc:        aiSvc,
		sender:       sender,
		location:     loc,
		lastRunDates: make(map[string]bool),
		stats: BroadcastStats{
			TotalBroadcasts: 0,
			TotalSent:       0,
			TotalFailed:     0,
			IsRunning:       false,
		},
	}
}

// Location returns the configured time.Location for the briefing engine.
func (e *BriefingEngine) Location() *time.Location {
	if e.location == nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return e.location
}

func (e *BriefingEngine) now() time.Time {
	return time.Now().In(e.Location())
}

// Start launches the background scheduler loop.
func (e *BriefingEngine) Start(ctx context.Context) {
	if !e.cfg.Enabled {
		slog.Info("Daily Briefing Scheduler is disabled via configuration")
		return
	}

	slog.Info("Starting Daily Briefing Scheduler daemon...",
		slog.Any("daily_hours", e.cfg.DailyHours),
		slog.String("timezone", e.Location().String()),
		slog.Int("interval_minutes", e.cfg.IntervalMinutes),
		slog.Int("rate_limit_per_sec", e.cfg.BroadcastRateLimitPerSec),
	)

	// Mode 1: Interval-based polling (primarily for dev/testing)
	if e.cfg.IntervalMinutes > 0 {
		ticker := time.NewTicker(time.Duration(e.cfg.IntervalMinutes) * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Daily Briefing Scheduler stopped gracefully")
				return
			case <-ticker.C:
				_, _ = e.TriggerAll(ctx, "", "")
			}
		}
	}

	// Mode 2: Specific Wall-Clock Daily Hours in configured timezone (e.g. 07:00, 17:00 WIB)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Daily Briefing Scheduler stopped gracefully")
			return
		case <-ticker.C:
			now := e.now()
			currentHourMinute := now.Format("15:04")
			todayDate := now.Format("2006-01-02")

			for _, scheduledHour := range e.cfg.DailyHours {
				if currentHourMinute == scheduledHour {
					runKey := fmt.Sprintf("%s_%s", todayDate, scheduledHour)

					e.mu.Lock()
					alreadyRan := e.lastRunDates[runKey]
					if !alreadyRan {
						e.lastRunDates[runKey] = true
					}
					e.mu.Unlock()

					if !alreadyRan {
						slog.Info("Triggering scheduled daily briefing broadcast",
							slog.String("scheduled_time", scheduledHour),
							slog.String("timezone", e.Location().String()),
						)
						go func() {
							_, _ = e.TriggerAll(context.Background(), "", "")
						}()
					}
				}
			}
		}
	}
}

// TriggerAll triggers an on-demand briefing broadcast across active topics or a specific filtered topic.
func (e *BriefingEngine) TriggerAll(ctx context.Context, topicFilter string, customMessage string) ([]BroadcastResult, error) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil, fmt.Errorf("broadcast cycle is already in progress")
	}
	e.running = true
	e.stats.IsRunning = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.stats.IsRunning = false
		e.mu.Unlock()
	}()

	// If custom announcement is provided without topicFilter, broadcast to all unique subscribers once
	if customMessage != "" && topicFilter == "" {
		return e.broadcastGlobalMessage(ctx, customMessage)
	}

	var topics []string
	if topicFilter != "" {
		topics = []string{strings.ToLower(strings.TrimSpace(topicFilter))}
	} else if e.repo != nil {
		activeTopics, err := e.repo.GetActiveTopics()
		if err == nil && len(activeTopics) > 0 {
			topics = activeTopics
		}
	}

	if len(topics) == 0 {
		topics = []string{"berita"}
	}

	var results []BroadcastResult
	startTime := time.Now()

	for _, topic := range topics {
		res, err := e.BroadcastToTopic(ctx, topic, customMessage)
		if err != nil {
			slog.Warn("Failed broadcast to topic", slog.String("topic", topic), slog.String("error", err.Error()))
		}
		if res != nil {
			results = append(results, *res)
		}
	}

	totalDuration := time.Since(startTime)

	// Update cumulative stats
	e.mu.Lock()
	e.stats.TotalBroadcasts++
	e.stats.LastRunTime = startTime
	e.stats.LastDurationMs = totalDuration.Milliseconds()
	for _, r := range results {
		e.stats.TotalSent += r.SuccessCount
		e.stats.TotalFailed += r.FailedCount
	}
	e.mu.Unlock()

	return results, nil
}

// broadcastGlobalMessage sends a custom announcement to all unique active subscribers exactly once.
func (e *BriefingEngine) broadcastGlobalMessage(ctx context.Context, customMessage string) ([]BroadcastResult, error) {
	if e.repo == nil {
		return nil, fmt.Errorf("repository is not available")
	}

	startTime := time.Now()
	topics, err := e.repo.GetActiveTopics()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active topics: %w", err)
	}

	seen := make(map[int64]struct{})
	var uniqueChatIDs []int64
	for _, t := range topics {
		ids, err := e.repo.GetSubscribersByTopic(t)
		if err != nil {
			continue
		}
		for _, id := range ids {
			if _, exists := seen[id]; !exists {
				seen[id] = struct{}{}
				uniqueChatIDs = append(uniqueChatIDs, id)
			}
		}
	}

	res, err := e.dispatchToSubscribers(ctx, "all", uniqueChatIDs, customMessage)
	if err != nil {
		return nil, err
	}

	totalDuration := time.Since(startTime)

	// Update cumulative stats
	e.mu.Lock()
	e.stats.TotalBroadcasts++
	e.stats.LastRunTime = startTime
	e.stats.LastDurationMs = totalDuration.Milliseconds()
	e.stats.TotalSent += res.SuccessCount
	e.stats.TotalFailed += res.FailedCount
	e.mu.Unlock()

	return []BroadcastResult{*res}, nil
}

// BroadcastToTopic compiles and broadcasts a digest for a specific topic to all its active subscribers.
func (e *BriefingEngine) BroadcastToTopic(ctx context.Context, topic string, customMessage string) (*BroadcastResult, error) {
	if e.repo == nil {
		return nil, fmt.Errorf("repository is not available")
	}

	chatIDs, err := e.repo.GetSubscribersByTopic(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscribers for topic %s: %w", topic, err)
	}

	// Deduplicate chat IDs within this topic
	seen := make(map[int64]struct{}, len(chatIDs))
	var uniqueChatIDs []int64
	for _, id := range chatIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			uniqueChatIDs = append(uniqueChatIDs, id)
		}
	}

	// 1. Synthesize briefing content
	var messageContent string
	if customMessage != "" {
		messageContent = customMessage
	} else {
		synthesized, err := e.SynthesizeBriefing(ctx, topic)
		if err != nil {
			slog.Warn("Failed to synthesize briefing, using fallback notice", slog.String("topic", topic), slog.String("error", err.Error()))
			messageContent = fmt.Sprintf("☀️ <b>Executive Briefing: %s</b>\n📅 <i>%s</i>\n\nMaaf, sistem sedang menyegarkan arsip berita untuk topik ini. Silakan cek kembali nanti atau ketik /briefing untuk mencoba lagi.",
				html.EscapeString(strings.ToUpper(topic)),
				FormatIndonesianDate(e.now()),
			)
		} else {
			messageContent = synthesized
		}
	}

	// 2. Dispatch rate-limited and flood-controlled broadcast
	return e.dispatchToSubscribers(ctx, topic, uniqueChatIDs, messageContent)
}

// dispatchToSubscribers handles the rate-limited, flood-controlled dispatch loop to a list of unique chat IDs.
func (e *BriefingEngine) dispatchToSubscribers(ctx context.Context, topic string, chatIDs []int64, messageContent string) (*BroadcastResult, error) {
	result := &BroadcastResult{
		TriggeredAt:      time.Now(),
		Topic:            topic,
		TotalSubscribers: len(chatIDs),
		SuccessCount:     0,
		FailedCount:      0,
	}

	if len(chatIDs) == 0 {
		result.Duration = time.Since(result.TriggeredAt)
		return result, nil
	}

	if e.sender == nil {
		result.Duration = time.Since(result.TriggeredAt)
		result.Errors = append(result.Errors, "telegram sender is disabled or nil")
		return result, nil
	}

	delayMs := 1000 / e.cfg.BroadcastRateLimitPerSec
	if delayMs <= 0 {
		delayMs = 40
	}

	for _, chatID := range chatIDs {
		// Respect context cancellation
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, "broadcast interrupted: context cancelled")
			result.Duration = time.Since(result.TriggeredAt)
			return result, ctx.Err()
		default:
		}

		err := e.sender.SendNotification(chatID, messageContent)
		if err != nil {
			errMsg := err.Error()
			lowerErr := strings.ToLower(errMsg)

			// Auto-deactivate subscription if user blocked bot or chat is invalid
			if strings.Contains(lowerErr, "blocked") || strings.Contains(lowerErr, "not found") || strings.Contains(lowerErr, "deactivated") || strings.Contains(lowerErr, "forbidden") {
				slog.Info("Auto-deactivating subscription for unreachable chat", slog.Int64("chat_id", chatID), slog.String("reason", errMsg))
				_ = e.repo.DeactivateTelegramChat(chatID)
				result.FailedCount++
				result.Errors = append(result.Errors, fmt.Sprintf("chat %d: %s", chatID, errMsg))
			} else if waitSec, isFlood := parseFloodWait(errMsg); isFlood {
				// Telegram Flood Control: pause dispatch loop and retry once for this chat
				slog.Warn("Telegram flood limit encountered during broadcast dispatch, pausing",
					slog.Int("wait_seconds", waitSec),
					slog.Int64("chat_id", chatID),
				)
				select {
				case <-time.After(time.Duration(waitSec) * time.Second):
					// Retry dispatch once for this subscriber
					retryErr := e.sender.SendNotification(chatID, messageContent)
					if retryErr == nil {
						result.SuccessCount++
					} else {
						result.FailedCount++
						result.Errors = append(result.Errors, fmt.Sprintf("chat %d: %s (after retry)", chatID, retryErr.Error()))
					}
				case <-ctx.Done():
					result.Errors = append(result.Errors, "broadcast interrupted during flood wait: context cancelled")
					result.Duration = time.Since(result.TriggeredAt)
					return result, ctx.Err()
				}
			} else {
				result.FailedCount++
				result.Errors = append(result.Errors, fmt.Sprintf("chat %d: %s", chatID, errMsg))
			}
		} else {
			result.SuccessCount++
		}

		// Safe throttling between messages
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	result.Duration = time.Since(result.TriggeredAt)
	return result, nil
}

// SynthesizeBriefing fetches top recent articles for a topic and generates an executive digest using AI.
func (e *BriefingEngine) SynthesizeBriefing(ctx context.Context, topic string) (string, error) {
	if e.repo == nil {
		return "", fmt.Errorf("repository is not configured")
	}

	articles, err := e.repo.GetRecentArticlesByTopic(ctx, topic, 24, e.cfg.MaxArticlesPerDigest)
	if err != nil || len(articles) == 0 {
		return "", fmt.Errorf("no recent articles found for topic: %s", topic)
	}

	// Build factual RAG context
	var sb strings.Builder
	for i, a := range articles {
		sb.WriteString(fmt.Sprintf("\n[Berita %d] %s (Sumber: %s)\nRingkasan: %s\n", i+1, a.Title, a.Source, a.Summary))
	}

	promptQuestion := fmt.Sprintf(
		"Buatlah 'Executive Daily Briefing' profesional berbahasa Indonesia untuk topik '%s' berdasarkan berita-berita berikut. "+
			"Format jawaban menjadi:\n"+
			"1. Intisari 2-4 poin utama peristiwa terpenting.\n"+
			"2. Kesimpulan singkat mengenai tren atau implikasi peristiwa hari ini.\n"+
			"Gunakan bahasa jurnalistik yang lugas, padat, dan faktual.",
		topic,
	)

	aiAnswer := ""
	if e.aiSvc != nil {
		ans, provider, err := e.aiSvc.AnswerQuestion(ctx, sb.String(), promptQuestion)
		if err == nil && !ai.IsLocalFallback(provider) && !strings.Contains(ans, "[Pemberitahuan Sistem]") && strings.TrimSpace(ans) != "" {
			aiAnswer = ans
		}
	}

	if aiAnswer == "" {
		// Fallback formatting directly from article summaries
		var fallbackBuilder strings.Builder
		fallbackBuilder.WriteString("Berikut adalah perkembangan berita penting hari ini:\n\n")
		for _, a := range articles {
			fallbackBuilder.WriteString(fmt.Sprintf("• <b>%s</b> (%s)\n  %s\n\n", html.EscapeString(a.Title), html.EscapeString(a.Source), html.EscapeString(a.Summary)))
		}
		aiAnswer = fallbackBuilder.String()
	}

	// Compile Final Telegram HTML Output
	var finalMsg strings.Builder
	finalMsg.WriteString(fmt.Sprintf("☀️ <b>Executive Briefing: %s</b>\n", html.EscapeString(strings.ToUpper(topic))))
	finalMsg.WriteString(fmt.Sprintf("📅 <i>%s</i>\n\n", FormatIndonesianDate(e.now())))
	finalMsg.WriteString(html.EscapeString(aiAnswer))
	finalMsg.WriteString("\n\n📰 <b>Referensi Berita:</b>\n")

	for i, a := range articles {
		source := a.Source
		if source == "" {
			source = "Media Nasional"
		}
		finalMsg.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a> (<i>%s</i>)\n", i+1, a.URL, html.EscapeString(a.Title), html.EscapeString(source)))
	}

	finalMsg.WriteString("\n<i>💡 Otomatis dikirim oleh News Aggregator Bot. Ketik /subscriptions untuk melihat langganan Anda.</i>")

	return finalMsg.String(), nil
}

// GetStats returns cumulative performance and run statistics.
func (e *BriefingEngine) GetStats() BroadcastStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}
