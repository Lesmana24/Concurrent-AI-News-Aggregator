package repository

import (
	"context"
	"fmt"
	"log/slog"
	"news-aggregator/internal/models"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresRepository struct {
	db        *gorm.DB
	hasVector bool
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	repo := &PostgresRepository{db: db}
	repo.InitVectorExtension()
	repo.InitSearchIndexes()
	repo.InitTimestampIndexes()
	return repo
}

// InitTimestampIndexes creates B-tree indexes on created_at and published_at
// to accelerate recency filtering, enable BitmapOr scans, and eliminate sequential table scans.
func (r *PostgresRepository) InitTimestampIndexes() error {
	if r.db == nil {
		return nil
	}

	indexes := []struct {
		name string
		sql  string
	}{
		{
			name: "idx_articles_created_at",
			sql:  "CREATE INDEX IF NOT EXISTS idx_articles_created_at ON articles (created_at DESC);",
		},
		{
			name: "idx_articles_published_at",
			sql:  "CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles (published_at DESC);",
		},
		{
			name: "idx_articles_source_created_at",
			sql:  "CREATE INDEX IF NOT EXISTS idx_articles_source_created_at ON articles (source, created_at DESC);",
		},
	}

	for _, idx := range indexes {
		if err := r.db.Exec(idx.sql).Error; err != nil {
			slog.Warn("Failed to create timestamp index", slog.String("index", idx.name), slog.String("error", err.Error()))
		} else {
			slog.Info("Timestamp index verified or created", slog.String("index", idx.name))
		}
	}

	return nil
}

// InitSearchIndexes creates the Full-Text Search (FTS) and Trigram (pg_trgm) GIN indexes
// to accelerate keyword searches and eliminate sequential table scans.
func (r *PostgresRepository) InitSearchIndexes() error {
	if r.db == nil {
		return nil
	}

	// 1. Core PostgreSQL Full-Text Search (FTS) GIN index
	// Uses 'simple' dictionary: standard in all PG distributions, preserves Indonesian words & acronyms.
	ftsIndexSQL := `CREATE INDEX IF NOT EXISTS idx_articles_fts ON articles 
		USING gin (to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(summary, '')));`
	if err := r.db.Exec(ftsIndexSQL).Error; err != nil {
		slog.Warn("FTS GIN index could not be created", slog.String("error", err.Error()))
	} else {
		slog.Info("FTS GIN index (idx_articles_fts) verified or created")
	}

	// 2. pg_trgm extension and GIN trigram indexes for partial/substring/fuzzy matching
	if err := r.db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm;").Error; err != nil {
		slog.Warn("pg_trgm extension could not be enabled (trigram indexing will be skipped)", slog.String("error", err.Error()))
		return nil
	}

	titleTrgmSQL := `CREATE INDEX IF NOT EXISTS idx_articles_title_trgm ON articles USING gin (title gin_trgm_ops);`
	if err := r.db.Exec(titleTrgmSQL).Error; err != nil {
		slog.Warn("failed to create trigram GIN index on articles.title", slog.String("error", err.Error()))
	}

	summaryTrgmSQL := `CREATE INDEX IF NOT EXISTS idx_articles_summary_trgm ON articles USING gin (summary gin_trgm_ops);`
	if err := r.db.Exec(summaryTrgmSQL).Error; err != nil {
		slog.Warn("failed to create trigram GIN index on articles.summary", slog.String("error", err.Error()))
	}

	slog.Info("Search GIN indexes (FTS and pg_trgm) successfully initialized")
	return nil
}

// InitVectorExtension attempts to enable pgvector and auto-migrate the ArticleEmbedding schema.
func (r *PostgresRepository) InitVectorExtension() (bool, error) {
	if r.db == nil {
		return false, nil
	}

	err := r.db.Exec("CREATE EXTENSION IF NOT EXISTS vector;").Error
	if err != nil {
		slog.Warn("pgvector extension could not be enabled (semantic vector search will be disabled)", slog.String("error", err.Error()))
		r.hasVector = false
		return false, err
	}

	if err := r.db.AutoMigrate(&models.ArticleEmbedding{}); err != nil {
		slog.Warn("ArticleEmbedding migration failed", slog.String("error", err.Error()))
		r.hasVector = false
		return false, err
	}

	// Create HNSW cosine similarity index for sub-millisecond vector lookups
	_ = r.db.Exec("CREATE INDEX IF NOT EXISTS idx_article_embeddings_hnsw ON article_embeddings USING hnsw (embedding vector_cosine_ops);").Error

	r.hasVector = true
	slog.Info("pgvector extension and HNSW index successfully initialized")
	return true, nil
}

func (r *PostgresRepository) HasVectorSupport() bool {
	return r.hasVector
}

func (r *PostgresRepository) GetUserByKey(apiKey string) (*models.User, error) {
	if r.db == nil {
		return nil, nil
	}
	var user models.User
	if err := r.db.Where("api_key = ?", apiKey).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email address.
func (r *PostgresRepository) GetUserByEmail(email string) (*models.User, error) {
	if r.db == nil {
		return nil, nil
	}
	var user models.User
	if err := r.db.Where("LOWER(email) = LOWER(?)", strings.TrimSpace(email)).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user with an initial quota.
func (r *PostgresRepository) CreateUser(email, apiKey string, initialQuota int) (*models.User, error) {
	if r.db == nil {
		return nil, nil
	}
	user := models.User{
		Email:      strings.ToLower(strings.TrimSpace(email)),
		APIKey:     strings.TrimSpace(apiKey),
		Quota:      initialQuota,
		DailyLimit: initialQuota,
		IsSystem:   false,
		IsGuest:    false,
	}
	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ResetUserDailyQuota atomically resets a user's quota to dailyLimit and updates last_reset_date.
func (r *PostgresRepository) ResetUserDailyQuota(userID uint, dailyLimit int, todayDate string) error {
	if r.db == nil {
		return nil
	}
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"quota":           dailyLimit,
			"last_reset_date": todayDate,
		}).Error
}

// GetOrCreateGuestUser retrieves an existing guest user or creates a new guest account with defaultLimit tokens.
func (r *PostgresRepository) GetOrCreateGuestUser(guestIdentifier string, todayDate string, defaultLimit int) (*models.User, error) {
	if r.db == nil {
		return nil, nil
	}
	cleanID := strings.ToLower(strings.TrimSpace(guestIdentifier))
	if cleanID == "" {
		cleanID = "anon"
	}
	// Sanitize to alphanumeric + dashes
	var sb strings.Builder
	for _, ch := range cleanID {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			sb.WriteRune(ch)
		}
	}
	safeID := sb.String()
	if safeID == "" {
		safeID = "anon"
	}
	if len(safeID) > 40 {
		safeID = safeID[:40]
	}
	guestEmail := fmt.Sprintf("guest-%s@guest.dispatch", safeID)

	var user models.User
	err := r.db.Where("email = ? AND is_guest = ?", guestEmail, true).First(&user).Error
	if err == nil {
		// Existing guest found: check if daily reset is needed
		if user.LastResetDate != todayDate {
			_ = r.db.Model(&user).Updates(map[string]interface{}{
				"quota":           defaultLimit,
				"daily_limit":     defaultLimit,
				"last_reset_date": todayDate,
			}).Error
			user.Quota = defaultLimit
			user.DailyLimit = defaultLimit
			user.LastResetDate = todayDate
		}
		return &user, nil
	}

	// Create new guest user
	newGuestKey := fmt.Sprintf("guest_%s_%d", safeID, time.Now().UnixNano())
	newUser := models.User{
		Email:         guestEmail,
		APIKey:        newGuestKey,
		Quota:         defaultLimit,
		DailyLimit:    defaultLimit,
		LastResetDate: todayDate,
		IsGuest:       true,
		IsSystem:      false,
	}

	if err := r.db.Create(&newUser).Error; err != nil {
		// In case of unique constraint race condition, attempt fetch again
		var retryUser models.User
		if retryErr := r.db.Where("email = ?", guestEmail).First(&retryUser).Error; retryErr == nil {
			return &retryUser, nil
		}
		return nil, err
	}

	return &newUser, nil
}

// RotateUserAPIKey updates a user's API key atomically.
func (r *PostgresRepository) RotateUserAPIKey(userID uint, newAPIKey string) error {
	if r.db == nil {
		return nil
	}
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("api_key", strings.TrimSpace(newAPIKey)).Error
}

func (r *PostgresRepository) UpdateUserQuota(userID uint, newQuota int) error {
	if r.db == nil {
		return nil
	}
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("quota", newQuota).Error
}

// DeductUserQuota atomically decrements the user's quota only if current quota >= amount.
// Returns (true, nil) if deducted, (false, nil) if quota is insufficient.
func (r *PostgresRepository) DeductUserQuota(userID uint, amount int) (bool, error) {
	if amount <= 0 {
		return true, nil
	}
	res := r.db.Model(&models.User{}).
		Where("id = ? AND quota >= ?", userID, amount).
		Update("quota", gorm.Expr("quota - ?", amount))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// RefundUserQuota atomically increments the user's quota in case of processing failure.
func (r *PostgresRepository) RefundUserQuota(userID uint, amount int) error {
	if amount <= 0 {
		return nil
	}
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("quota", gorm.Expr("quota + ?", amount)).Error
}

func (r *PostgresRepository) BatchInsertArticles(articles []models.Article) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "content", "summary", "ai_provider"}),
	}).Create(&articles).Error
}

func (r *PostgresRepository) GetCachedArticles(url string) ([]models.Article, error) {
	var articles []models.Article
	err := r.db.Where("url = ?", url).Find(&articles).Error
	return articles, err
}

func (r *PostgresRepository) GetArticleByURL(url string) (*models.Article, error) {
	var article models.Article
	if err := r.db.Where("url = ?", url).First(&article).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

// SaveArticleEmbedding upserts dense vector embedding for an article.
func (r *PostgresRepository) SaveArticleEmbedding(articleID uint, embedding []float32) error {
	if !r.hasVector || len(embedding) == 0 {
		return nil
	}
	vec := pgvector.NewVector(embedding)
	record := models.ArticleEmbedding{
		ArticleID: articleID,
		Embedding: vec,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "article_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"embedding"}),
	}).Create(&record).Error
}

// SearchSimilarArticles queries the top-K most similar articles using cosine distance <=> in pgvector.
func (r *PostgresRepository) SearchSimilarArticles(ctx context.Context, queryVector []float32, limit int, minSimilarity float64) ([]models.SemanticSearchResult, error) {
	if !r.hasVector || len(queryVector) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}

	vecStr := pgvector.NewVector(queryVector).String()
	var results []models.SemanticSearchResult

	err := r.db.WithContext(ctx).Raw(`
		SELECT 
			a.id AS article_id,
			a.title,
			a.url,
			a.summary,
			a.source,
			(1 - (e.embedding <=> ?)) AS similarity
		FROM articles a
		JOIN article_embeddings e ON a.id = e.article_id
		WHERE (1 - (e.embedding <=> ?)) >= ?
		ORDER BY similarity DESC
		LIMIT ?
	`, vecStr, vecStr, minSimilarity, limit).Scan(&results).Error

	return results, err
}

var idStopWords = map[string]struct{}{
	"apa": {}, "siapa": {}, "kapan": {}, "dimana": {}, "dmana": {}, "bagaimana": {}, "bgmn": {},
	"mengapa": {}, "kenapa": {}, "knp": {}, "apakah": {}, "berita": {}, "kabar": {}, "terbaru": {},
	"tentang": {}, "ttg": {}, "mengenai": {}, "seputar": {}, "terkait": {}, "adalah": {}, "yaitu": {},
	"atau": {}, "dan": {}, "yang": {}, "yg": {}, "ini": {}, "itu": {}, "dari": {}, "pada": {},
	"untuk": {}, "utk": {}, "bisa": {}, "ada": {}, "oleh": {}, "para": {}, "hari": {}, "kemarin": {},
	"besok": {}, "lalu": {}, "info": {}, "informasi": {}, "tolong": {}, "coba": {}, "jelaskan": {},
	"dong": {}, "saja": {}, "kah": {}, "lah": {}, "pun": {}, "mendasari": {}, "dasar": {}, "alasan": {},
	"faktor": {}, "penyebab": {}, "akibat": {}, "soal": {}, "hal": {},
}

var validAcronyms = map[string]bool{
	"mk": true, "ma": true, "ri": true, "uu": true, "dpr": true, "kpu": true, "tni": true, "polri": true,
	"bp": true, "ai": true, "as": true, "eu": true, "ap": true, "pbb": true, "cpo": true, "bbm": true, "spbu": true,
}

func containsWholeWord(text, word string) bool {
	lowerText := strings.ToLower(text)
	lowerWord := strings.ToLower(word)
	idx := strings.Index(lowerText, lowerWord)
	for idx != -1 {
		rBefore := rune(0)
		if idx > 0 {
			rBefore, _ = utf8.DecodeLastRuneInString(lowerText[:idx])
		}
		startOk := (idx == 0) || !isAlphaNum(rBefore)

		endIdx := idx + len(lowerWord)
		rAfter := rune(0)
		if endIdx < len(lowerText) {
			rAfter, _ = utf8.DecodeRuneInString(lowerText[endIdx:])
		}
		endOk := (endIdx == len(lowerText)) || !isAlphaNum(rAfter)
		if startOk && endOk {
			return true
		}
		next := strings.Index(lowerText[endIdx:], lowerWord)
		if next == -1 {
			break
		}
		idx = endIdx + next
	}
	return false
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// sanitizeTSQueryTerm extracts alphanumeric characters from a word to prevent syntax errors in to_tsquery.
func sanitizeTSQueryTerm(word string) string {
	var sb strings.Builder
	for _, r := range word {
		if isAlphaNum(r) || r > 127 {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// buildTSQuery constructs an OR-joined prefix-matching tsquery string for PostgreSQL to_tsquery.
// e.g. ["kripto", "pajak"] -> "'kripto':* | 'pajak':*"
func buildTSQuery(words []string) string {
	seen := make(map[string]struct{}, len(words))
	var terms []string
	for _, w := range words {
		clean := sanitizeTSQueryTerm(strings.ToLower(w))
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		terms = append(terms, "'"+clean+"':*")
	}
	if len(terms) == 0 {
		return ""
	}
	return strings.Join(terms, " | ")
}

// SearchArticlesByKeyword searches articles in PostgreSQL using index-backed FTS and trigram matching.
func (r *PostgresRepository) SearchArticlesByKeyword(ctx context.Context, query string, limit int) ([]models.SemanticSearchResult, error) {
	if r.db == nil {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 0 {
		limit = 5
	}

	cleanQuery := strings.Map(func(r rune) rune {
		if strings.ContainsRune("?!.,:;()[]\"'{}<>/\\|@#$%^&*~`+=_-", r) {
			return ' '
		}
		return r
	}, strings.ToLower(strings.TrimSpace(query)))

	rawWords := strings.Fields(cleanQuery)
	if len(rawWords) == 0 {
		return nil, nil
	}

	var meaningfulWords []string
	for _, w := range rawWords {
		if _, isStop := idStopWords[w]; isStop {
			continue
		}
		if len(w) >= 3 || validAcronyms[w] {
			meaningfulWords = append(meaningfulWords, w)
			if strings.HasPrefix(w, "demonstra") {
				meaningfulWords = append(meaningfulWords, "demo")
			}
		}
	}

	if len(meaningfulWords) == 0 {
		for _, w := range rawWords {
			if len(w) >= 3 || validAcronyms[w] {
				meaningfulWords = append(meaningfulWords, w)
			}
		}
	}
	if len(meaningfulWords) == 0 {
		meaningfulWords = rawWords
	}

	// 1. Primary candidate retrieval: PostgreSQL Full-Text Search (FTS) using GIN index (idx_articles_fts)
	var candidates []models.Article
	tsQuery := buildTSQuery(meaningfulWords)
	if tsQuery != "" {
		ftsClause := "to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(summary, '')) @@ to_tsquery('simple', ?)"
		err := r.db.WithContext(ctx).Model(&models.Article{}).
			Where(ftsClause, tsQuery).
			Order("id DESC").
			Limit(40).
			Find(&candidates).Error
		if err != nil {
			slog.Debug("FTS candidate query failed, falling back to trigram search", slog.String("error", err.Error()))
			candidates = nil
		}
	}

	// 2. Fallback candidate retrieval: Trigram-accelerated ILIKE search using GIN indexes (idx_articles_title_trgm, idx_articles_summary_trgm)
	if len(candidates) == 0 && len(meaningfulWords) > 0 {
		var orConditions []string
		var orArgs []interface{}
		for _, w := range meaningfulWords {
			pattern := "%" + w + "%"
			orConditions = append(orConditions, "title ILIKE ? OR summary ILIKE ?")
			orArgs = append(orArgs, pattern, pattern)
		}

		if len(orConditions) > 0 {
			clause := strings.Join(orConditions, " OR ")
			_ = r.db.WithContext(ctx).Model(&models.Article{}).
				Where(clause, orArgs...).
				Order("id DESC").
				Limit(40).
				Find(&candidates).Error
		}
	}

	// 3. Fallback to latest articles if no matches found
	if len(candidates) == 0 {
		_ = r.db.WithContext(ctx).Model(&models.Article{}).
			Order("id DESC").
			Limit(limit).
			Find(&candidates).Error
	}

	type scoredArticle struct {
		article models.Article
		score   int
	}

	var scored []scoredArticle
	maxScore := 0

	for _, a := range candidates {
		score := 0
		distinctMatches := 0

		for _, w := range meaningfulWords {
			matchedThisWord := false

			// Title Match (Whole Word preferred)
			if containsWholeWord(a.Title, w) {
				score += 10
				matchedThisWord = true
			} else if len(w) >= 4 && strings.Contains(strings.ToLower(a.Title), w) {
				score += 5
				matchedThisWord = true
			}

			// Summary Match (Whole Word preferred)
			if containsWholeWord(a.Summary, w) {
				score += 4
				matchedThisWord = true
			} else if len(w) >= 4 && strings.Contains(strings.ToLower(a.Summary), w) {
				score += 2
				matchedThisWord = true
			}

			if matchedThisWord {
				distinctMatches++
			}
		}

		if distinctMatches > 1 {
			score += distinctMatches * 6
		}

		if score > 0 {
			scored = append(scored, scoredArticle{article: a, score: score})
			if score > maxScore {
				maxScore = score
			}
		}
	}

	// Sort by relevance score descending, then by id descending
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].article.ID > scored[j].article.ID
	})

	// Filter out noise: discard candidates with score < 35% of top score
	cutoffScore := 3
	if maxScore >= 10 {
		cutoffScore = int(float64(maxScore) * 0.35)
	}

	var finalArticles []models.Article
	for _, s := range scored {
		if s.score >= cutoffScore {
			finalArticles = append(finalArticles, s.article)
			if len(finalArticles) >= limit {
				break
			}
		}
	}

	if len(finalArticles) == 0 && len(candidates) > 0 {
		end := limit
		if len(candidates) < end {
			end = len(candidates)
		}
		finalArticles = candidates[:end]
	}

	results := make([]models.SemanticSearchResult, len(finalArticles))
	for i, a := range finalArticles {
		sim := 0.75
		if maxScore > 0 && i < len(scored) && scored[i].article.ID == a.ID {
			sim = 0.65 + (float64(scored[i].score)/float64(maxScore))*0.30
			if sim > 0.98 {
				sim = 0.98
			}
		}
		results[i] = models.SemanticSearchResult{
			ArticleID:  a.ID,
			Title:      a.Title,
			URL:        a.URL,
			Summary:    a.Summary,
			Source:     a.Source,
			Similarity: sim,
		}
	}

	return results, nil
}

// EnsureSystemUser creates or retrieves a system user for background/bot operations.
func (r *PostgresRepository) EnsureSystemUser(apiKey, email string, defaultQuota int) (*models.User, error) {
	var user models.User
	err := r.db.Where("api_key = ?", apiKey).First(&user).Error
	if err == nil {
		if !user.IsSystem {
			_ = r.db.Model(&user).Update("is_system", true).Error
			user.IsSystem = true
		}
		if user.Quota < 100 {
			_ = r.db.Model(&user).Update("quota", defaultQuota).Error
			user.Quota = defaultQuota
		}
		return &user, nil
	}

	user = models.User{
		APIKey:     apiKey,
		Email:      email,
		Quota:      defaultQuota,
		DailyLimit: defaultQuota,
		IsSystem:   true,
		IsGuest:    false,
	}
	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// SubscribeTelegram activates or adds a telegram chat subscription for a topic.
func (r *PostgresRepository) SubscribeTelegram(chatID int64, topic string) error {
	topic = strings.TrimSpace(strings.ToLower(topic))
	if topic == "" {
		topic = "berita"
	}

	var sub models.TelegramSubscription
	err := r.db.Where("chat_id = ? AND topic = ?", chatID, topic).First(&sub).Error
	if err == nil {
		return r.db.Model(&sub).Update("is_active", true).Error
	}

	sub = models.TelegramSubscription{
		ChatID:   chatID,
		Topic:    topic,
		IsActive: true,
	}
	return r.db.Create(&sub).Error
}

// UnsubscribeTelegram deactivates all subscriptions for a chat ID.
func (r *PostgresRepository) UnsubscribeTelegram(chatID int64) error {
	return r.db.Model(&models.TelegramSubscription{}).
		Where("chat_id = ?", chatID).
		Update("is_active", false).Error
}

// UnsubscribeTelegramTopic deactivates a specific subscription topic for a chat ID.
// If topic is empty or "all", it deactivates all subscriptions for that chat.
func (r *PostgresRepository) UnsubscribeTelegramTopic(chatID int64, topic string) error {
	topic = strings.TrimSpace(strings.ToLower(topic))
	if topic == "" || topic == "all" || topic == "semua" {
		return r.UnsubscribeTelegram(chatID)
	}

	return r.db.Model(&models.TelegramSubscription{}).
		Where("chat_id = ? AND topic = ?", chatID, topic).
		Update("is_active", false).Error
}

// GetChatSubscriptions retrieves active subscriptions for a specific chat ID.
func (r *PostgresRepository) GetChatSubscriptions(chatID int64) ([]models.TelegramSubscription, error) {
	var subs []models.TelegramSubscription
	err := r.db.Where("chat_id = ? AND is_active = ?", chatID, true).Order("topic ASC").Find(&subs).Error
	return subs, err
}

// GetActiveTelegramSubscriptions retrieves all active subscriptions.
func (r *PostgresRepository) GetActiveTelegramSubscriptions() ([]models.TelegramSubscription, error) {
	var subs []models.TelegramSubscription
	err := r.db.Where("is_active = ?", true).Find(&subs).Error
	return subs, err
}

// GetSubscribersByTopic retrieves unique active chat IDs subscribed to a given topic.
func (r *PostgresRepository) GetSubscribersByTopic(topic string) ([]int64, error) {
	topic = strings.TrimSpace(strings.ToLower(topic))
	var chatIDs []int64
	err := r.db.Model(&models.TelegramSubscription{}).
		Where("topic = ? AND is_active = ?", topic, true).
		Distinct("chat_id").
		Pluck("chat_id", &chatIDs).Error
	return chatIDs, err
}

// GetActiveTopics retrieves distinct active topics currently subscribed to.
func (r *PostgresRepository) GetActiveTopics() ([]string, error) {
	var topics []string
	err := r.db.Model(&models.TelegramSubscription{}).
		Where("is_active = ?", true).
		Distinct("topic").
		Pluck("topic", &topics).Error
	return topics, err
}

// DeactivateTelegramChat deactivates all subscriptions for a chat ID (e.g. when user blocked bot).
func (r *PostgresRepository) DeactivateTelegramChat(chatID int64) error {
	return r.db.Model(&models.TelegramSubscription{}).
		Where("chat_id = ?", chatID).
		Update("is_active", false).Error
}

// GetRecentArticlesByTopic retrieves recent articles (default within hours, with fallback) for a topic.
func (r *PostgresRepository) GetRecentArticlesByTopic(ctx context.Context, topic string, hours int, limit int) ([]models.Article, error) {
	if r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	if hours <= 0 {
		hours = 24
	}

	topic = strings.TrimSpace(strings.ToLower(topic))
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)

	var articles []models.Article

	isGeneric := topic == "" || topic == "berita" || topic == "umum" || topic == "berita umum" || topic == "semua"
	if isGeneric {
		// Try fetching within time window
		err := r.db.WithContext(ctx).Model(&models.Article{}).
			Where("created_at >= ? OR published_at >= ?", cutoff, cutoff).
			Order("id DESC").
			Limit(limit).
			Find(&articles).Error
		if err == nil && len(articles) > 0 {
			return articles, nil
		}

		// Fallback to latest articles regardless of window
		err = r.db.WithContext(ctx).Model(&models.Article{}).
			Order("id DESC").
			Limit(limit).
			Find(&articles).Error
		return articles, err
	}

	// Topic-specific search
	pattern := "%" + topic + "%"
	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("(LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(source) LIKE ?) AND (created_at >= ? OR published_at >= ?)", pattern, pattern, pattern, cutoff, cutoff).
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	if err == nil && len(articles) > 0 {
		return articles, nil
	}

	// Fallback without cutoff
	err = r.db.WithContext(ctx).Model(&models.Article{}).
		Where("LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(source) LIKE ?", pattern, pattern, pattern).
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	if err == nil && len(articles) > 0 {
		return articles, nil
	}

	// If still empty for niche topic, fallback to latest articles
	err = r.db.WithContext(ctx).Model(&models.Article{}).
		Order("id DESC").
		Limit(limit).
		Find(&articles).Error
	return articles, err
}

// SubscriptionStats stores statistics about active subscribers.
type SubscriptionStats struct {
	TotalActive   int64            `json:"total_active"`
	TotalInactive int64            `json:"total_inactive"`
	UniqueUsers   int64            `json:"unique_users"`
	TopicCounts   map[string]int64 `json:"topic_counts"`
}

// GetSubscriptionStats retrieves subscriber counts and breakdown by topic.
func (r *PostgresRepository) GetSubscriptionStats() (*SubscriptionStats, error) {
	var totalActive int64
	var totalInactive int64
	var uniqueUsers int64

	_ = r.db.Model(&models.TelegramSubscription{}).Where("is_active = ?", true).Count(&totalActive).Error
	_ = r.db.Model(&models.TelegramSubscription{}).Where("is_active = ?", false).Count(&totalInactive).Error
	_ = r.db.Model(&models.TelegramSubscription{}).Where("is_active = ?", true).Distinct("chat_id").Count(&uniqueUsers).Error

	type topicRow struct {
		Topic string
		Count int64
	}
	var rows []topicRow
	_ = r.db.Model(&models.TelegramSubscription{}).
		Select("topic, count(*) as count").
		Where("is_active = ?", true).
		Group("topic").
		Scan(&rows).Error

	topicCounts := make(map[string]int64)
	for _, row := range rows {
		topicCounts[row.Topic] = row.Count
	}

	return &SubscriptionStats{
		TotalActive:   totalActive,
		TotalInactive: totalInactive,
		UniqueUsers:   uniqueUsers,
		TopicCounts:   topicCounts,
	}, nil
}

// GetRecentArticles fetches articles published or created within hours, or falls back to latest articles.
func (r *PostgresRepository) GetRecentArticles(ctx context.Context, hours int) ([]models.Article, error) {
	if r.db == nil {
		return nil, nil
	}
	if hours <= 0 {
		hours = 48
	}
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)

	var articles []models.Article
	err := r.db.WithContext(ctx).Model(&models.Article{}).
		Where("created_at >= ? OR published_at >= ?", cutoff, cutoff).
		Order("id DESC").
		Limit(100).
		Find(&articles).Error
	if err == nil && len(articles) > 0 {
		return articles, nil
	}

	// Fallback to latest 50 articles
	err = r.db.WithContext(ctx).Model(&models.Article{}).
		Order("id DESC").
		Limit(50).
		Find(&articles).Error
	return articles, err
}

// GetArticleEmbeddings retrieves vector embeddings for specified article IDs.
func (r *PostgresRepository) GetArticleEmbeddings(ctx context.Context, articleIDs []uint) (map[uint][]float32, error) {
	if len(articleIDs) == 0 || !r.hasVector {
		return make(map[uint][]float32), nil
	}

	var rows []models.ArticleEmbedding
	err := r.db.WithContext(ctx).Where("article_id IN ?", articleIDs).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	res := make(map[uint][]float32, len(rows))
	for _, row := range rows {
		res[row.ArticleID] = row.Embedding.Slice()
	}
	return res, nil
}

// SaveClusters atomically updates active topic clusters and replaces old snapshots.
func (r *PostgresRepository) SaveClusters(ctx context.Context, clusters []models.TopicCluster) error {
	if len(clusters) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clean up existing clusters and associations to replace with the fresh authoritative snapshot
		_ = tx.Exec("DELETE FROM cluster_articles").Error
		_ = tx.Exec("DELETE FROM topic_clusters").Error

		// Insert new authoritative clusters
		for i := range clusters {
			clusters[i].ID = 0
			if err := tx.Create(&clusters[i]).Error; err != nil {
				return err
			}
			// Replace association
			if len(clusters[i].Articles) > 0 {
				_ = tx.Model(&clusters[i]).Association("Articles").Replace(clusters[i].Articles)
			}
		}
		return nil
	})
}

// GetTrendingClusters retrieves the top trending topic clusters ranked by hotness score.
func (r *PostgresRepository) GetTrendingClusters(ctx context.Context, limit int) ([]models.TopicCluster, error) {
	if limit <= 0 {
		limit = 5
	}
	var clusters []models.TopicCluster
	err := r.db.WithContext(ctx).
		Order("hotness_score DESC, id DESC").
		Limit(limit).
		Find(&clusters).Error
	return clusters, err
}

// GetClusterByID retrieves a single topic cluster with all associated articles.
func (r *PostgresRepository) GetClusterByID(ctx context.Context, id uint) (*models.TopicCluster, error) {
	var cluster models.TopicCluster
	err := r.db.WithContext(ctx).
		Preload("Articles").
		Where("id = ?", id).
		First(&cluster).Error
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}
