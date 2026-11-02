package scheduler

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"news-aggregator/internal/ai"
	"strings"
	"sync"
	"time"
)

// BriefingEngine orchestrates scheduled AI news syntheses and rate-limited push notifications.
type BriefingEngine struct {
	cfg          Config
	repo         BriefingRepository
	aiSvc        ai.AIService
	sender       TelegramSender
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

	return &BriefingEngine{
		cfg:          cfg,
		repo:         repo,
		aiSvc:        aiSvc,
		sender:       sender,
		lastRunDates: make(map[string]bool),
		stats: BroadcastStats{
			TotalBroadcasts: 0,
			TotalSent:       0,
			TotalFailed:     0,
			IsRunning:       false,
		},
	}
}

// Start launches the background scheduler loop.
func (e *BriefingEngine) Start(ctx context.Context) {
	if !e.cfg.Enabled {
		slog.Info("Daily Briefing Scheduler is disabled via configuration")
		return
	}

	slog.Info("Starting Daily Briefing Scheduler daemon...",
		slog.Any("daily_hours", e.cfg.DailyHours),
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

	// Mode 2: Specific Wall-Clock Daily Hours (07:00, 17:00 WIB)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Daily Briefing Scheduler stopped gracefully")
			return
		case <-ticker.C:
			now := time.Now()
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
						slog.Info("Triggering scheduled daily briefing broadcast", slog.String("scheduled_time", scheduledHour))
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

// BroadcastToTopic compiles and broadcasts a digest for a specific topic to all its active subscribers.
func (e *BriefingEngine) BroadcastToTopic(ctx context.Context, topic string, customMessage string) (*BroadcastResult, error) {
	if e.repo == nil {
		return nil, fmt.Errorf("repository is not available")
	}

	chatIDs, err := e.repo.GetSubscribersByTopic(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscribers for topic %s: %w", topic, err)
	}

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
				time.Now().Format("02 January 2006"),
			)
		} else {
			messageContent = synthesized
		}
	}

	// 2. Dispatch rate-limited broadcast
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
			result.FailedCount++
			errMsg := err.Error()
			result.Errors = append(result.Errors, fmt.Sprintf("chat %d: %s", chatID, errMsg))

			// Check if user blocked bot or chat is invalid -> auto-deactivate subscription
			lowerErr := strings.ToLower(errMsg)
			if strings.Contains(lowerErr, "blocked") || strings.Contains(lowerErr, "not found") || strings.Contains(lowerErr, "deactivated") || strings.Contains(lowerErr, "forbidden") {
				slog.Info("Auto-deactivating subscription for unreachable chat", slog.Int64("chat_id", chatID), slog.String("reason", errMsg))
				_ = e.repo.DeactivateTelegramChat(chatID)
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
		ans, _, err := e.aiSvc.AnswerQuestion(ctx, sb.String(), promptQuestion)
		if err == nil && strings.TrimSpace(ans) != "" {
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
	finalMsg.WriteString(fmt.Sprintf("📅 <i>%s</i>\n\n", time.Now().Format("02 January 2006")))
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
