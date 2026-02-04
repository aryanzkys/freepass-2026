package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"freepass-2026/internal/ai"
	httpcontext "freepass-2026/internal/http/context"
	"freepass-2026/internal/http/response"
	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AIClient interface {
	Configured() bool
	Model() string
	GenerateJSON(ctx context.Context, systemPrompt string, userPrompt string, schema map[string]interface{}) ([]byte, error)
}

type AIQueries interface {
	GetCanteenByID(ctx context.Context, id pgtype.UUID) (db.Canteen, error)
	GetMenuItemByID(ctx context.Context, id pgtype.UUID) (db.MenuItem, error)
	ListFeedbackTextsByCanteenAndRange(ctx context.Context, arg db.ListFeedbackTextsByCanteenAndRangeParams) ([]db.ListFeedbackTextsByCanteenAndRangeRow, error)
	AggregateMenuRatingsByCanteen(ctx context.Context, canteenID pgtype.UUID) ([]db.AggregateMenuRatingsByCanteenRow, error)
	CanteenOrdersStatsByRange(ctx context.Context, arg db.CanteenOrdersStatsByRangeParams) (db.CanteenOrdersStatsByRangeRow, error)
	ListCanteensBasic(ctx context.Context) ([]db.ListCanteensBasicRow, error)
}

type AIHandler struct {
	Queries AIQueries
	AI      AIClient
	Clock   func() time.Time
}

type sentimentResponse struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type themeCountResponse struct {
	Theme string `json:"theme"`
	Count int32  `json:"count"`
}

type ownerFeedbackInsightsResponse struct {
	CanteenID     string               `json:"canteen_id"`
	RangeDays     int                  `json:"range_days"`
	Summary       string               `json:"summary"`
	Sentiment     sentimentResponse    `json:"sentiment"`
	TopThemes     []themeCountResponse `json:"top_themes"`
	ActionItems   []string             `json:"action_items"`
	NotableQuotes []string             `json:"notable_quotes"`
	GeneratedAt   string               `json:"generated_at"`
	Model         string               `json:"model"`
}

type recommendationSignal struct {
	Signal string `json:"signal"`
	Value  string `json:"value"`
}

type userRecommendationExplainResponse struct {
	CanteenID   string                 `json:"canteen_id"`
	MenuItemID  *string                `json:"menu_item_id"`
	Explanation string                 `json:"explanation"`
	Signals     []recommendationSignal `json:"signals"`
	GeneratedAt string                 `json:"generated_at"`
	Model       string                 `json:"model"`
}

type adminWatchlistItem struct {
	CanteenID          string   `json:"canteen_id"`
	CanteenName        string   `json:"canteen_name"`
	RiskSummary        string   `json:"risk_summary"`
	TopComplaints      []string `json:"top_complaints"`
	SuggestedFollowups []string `json:"suggested_followups"`
}

type adminOversightResponse struct {
	RangeDays   int                  `json:"range_days"`
	Watchlist   []adminWatchlistItem `json:"watchlist"`
	GeneratedAt string               `json:"generated_at"`
	Model       string               `json:"model"`
}

func NewAIHandler(queries AIQueries, client AIClient, clock func() time.Time) *AIHandler {
	if clock == nil {
		clock = time.Now
	}
	return &AIHandler{Queries: queries, AI: client, Clock: clock}
}

func (h *AIHandler) OwnerFeedbackInsights(c *gin.Context) {
	role, ok := httpcontext.UserRole(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	if role != "OWNER" && role != "ADMIN" {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}
	userID, ok := httpcontext.UserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil || !userUUID.Valid {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	if !h.ensureAIConfigured(c) {
		return
	}
	days, ok := parseDays(c)
	if !ok {
		return
	}
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	ctx := c.Request.Context()
	canteen, err := h.Queries.GetCanteenByID(ctx, canteenUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	if role == "OWNER" && canteen.OwnerID != userUUID {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}

	endAt := h.Clock().UTC()
	startAt := endAt.AddDate(0, 0, -days)
	feedbacks, err := h.Queries.ListFeedbackTextsByCanteenAndRange(ctx, db.ListFeedbackTextsByCanteenAndRangeParams{
		CanteenID: canteenUUID,
		StartAt:   pgtype.Timestamptz{Time: startAt, Valid: true},
		EndAt:     pgtype.Timestamptz{Time: endAt, Valid: true},
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	stats, err := h.Queries.CanteenOrdersStatsByRange(ctx, db.CanteenOrdersStatsByRangeParams{
		CanteenID: canteenUUID,
		StartAt:   pgtype.Timestamptz{Time: startAt, Valid: true},
		EndAt:     pgtype.Timestamptz{Time: endAt, Valid: true},
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	metrics := calculateFeedbackMetrics(feedbacks)
	feedbackSamples := sampleFeedbackTexts(feedbacks, 25)

	input := map[string]interface{}{
		"canteen_id": canteenUUID.String(),
		"range_days": days,
		"stats": map[string]interface{}{
			"total_orders":      stats.TotalOrders,
			"completed_orders":  stats.CompletedOrders,
			"rejected_payments": stats.RejectedPayments,
			"refunded_payments": stats.RefundedPayments,
			"cash_orders":       stats.CashOrders,
			"cashless_orders":   stats.CashlessOrders,
			"feedback_count":    metrics.Count,
			"rating_count":      metrics.RatingCount,
			"avg_rating":        metrics.AvgRating,
			"positive_count":    metrics.Positive,
			"neutral_count":     metrics.Neutral,
			"negative_count":    metrics.Negative,
		},
		"feedback_samples": feedbackSamples,
	}
	userPrompt := mustJSON(input)
	systemPrompt := "Return JSON only. Use the data to summarize feedback, infer sentiment, extract themes, and propose action items. Do not include user identifiers or personal data. Notable quotes must be short excerpts up to 120 characters."
	schema := ownerInsightsSchema()
	payload, err := h.AI.GenerateJSON(ctx, systemPrompt, userPrompt, schema)
	if err != nil {
		handleAIError(c, err)
		return
	}
	var aiResult struct {
		Summary       string               `json:"summary"`
		Sentiment     sentimentResponse    `json:"sentiment"`
		TopThemes     []themeCountResponse `json:"top_themes"`
		ActionItems   []string             `json:"action_items"`
		NotableQuotes []string             `json:"notable_quotes"`
	}
	if err := json.Unmarshal(payload, &aiResult); err != nil {
		response.Error(c, http.StatusBadGateway, "ai_invalid_response", map[string]string{"error": err.Error()})
		return
	}
	aiResult.NotableQuotes = trimQuotes(aiResult.NotableQuotes, 120)

	resp := ownerFeedbackInsightsResponse{
		CanteenID:     canteenUUID.String(),
		RangeDays:     days,
		Summary:       aiResult.Summary,
		Sentiment:     aiResult.Sentiment,
		TopThemes:     aiResult.TopThemes,
		ActionItems:   aiResult.ActionItems,
		NotableQuotes: aiResult.NotableQuotes,
		GeneratedAt:   timeToString(pgtype.Timestamptz{Time: endAt, Valid: true}),
		Model:         h.AI.Model(),
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AIHandler) UserRecommendationExplain(c *gin.Context) {
	role, ok := httpcontext.UserRole(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	if role != "USER" && role != "ADMIN" && role != "OWNER" {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}
	if !h.ensureAIConfigured(c) {
		return
	}
	days, ok := parseDays(c)
	if !ok {
		return
	}
	canteenUUID, ok := parseUUIDParam(c, "canteenId", "canteen_id")
	if !ok {
		return
	}
	ctx := c.Request.Context()
	_, err := h.Queries.GetCanteenByID(ctx, canteenUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	var menuItemID *string
	var menuItem *db.MenuItem
	if value := c.Query("menu_item_id"); value != "" {
		var menuUUID pgtype.UUID
		if err := menuUUID.Scan(value); err != nil || !menuUUID.Valid {
			details := map[string]string{"menu_item_id": "invalid_uuid"}
			response.Error(c, http.StatusBadRequest, "validation_error", details)
			return
		}
		item, err := h.Queries.GetMenuItemByID(ctx, menuUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(c, http.StatusNotFound, "not_found", nil)
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		if item.CanteenID != canteenUUID {
			response.Error(c, http.StatusNotFound, "not_found", nil)
			return
		}
		menuItemID = &value
		menuItem = &item
	}

	endAt := h.Clock().UTC()
	startAt := endAt.AddDate(0, 0, -days)
	feedbacks, err := h.Queries.ListFeedbackTextsByCanteenAndRange(ctx, db.ListFeedbackTextsByCanteenAndRangeParams{
		CanteenID: canteenUUID,
		StartAt:   pgtype.Timestamptz{Time: startAt, Valid: true},
		EndAt:     pgtype.Timestamptz{Time: endAt, Valid: true},
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	metrics := calculateFeedbackMetrics(feedbacks)
	menuAgg, err := h.Queries.AggregateMenuRatingsByCanteen(ctx, canteenUUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}

	menuStats := buildMenuStats(menuAgg)
	menuContext := map[string]interface{}{}
	if menuItem != nil {
		menuContext = map[string]interface{}{
			"menu_item_id":      menuItem.ID.String(),
			"menu_name":         menuItem.Name,
			"menu_price":        menuItem.Price,
			"menu_available":    menuItem.IsAvailable,
			"menu_avg_rating":   menuStats.ItemAvg[menuItem.ID.String()],
			"menu_rating_count": menuStats.ItemCount[menuItem.ID.String()],
		}
	}

	input := map[string]interface{}{
		"canteen_id": canteenUUID.String(),
		"range_days": days,
		"menu_item":  menuContext,
		"stats": map[string]interface{}{
			"feedback_count":    metrics.Count,
			"rating_count":      metrics.RatingCount,
			"avg_rating":        metrics.AvgRating,
			"positive_count":    metrics.Positive,
			"neutral_count":     metrics.Neutral,
			"negative_count":    metrics.Negative,
			"menu_rating_count": menuStats.TotalCount,
			"menu_avg_rating":   menuStats.OverallAvg,
			"top_menu_items":    menuStats.TopItems,
		},
	}
	userPrompt := mustJSON(input)
	systemPrompt := "Return JSON only. Explain why the canteen or menu item is recommended using aggregate signals. Do not include personal data or long quotes."
	schema := recommendationSchema()
	payload, err := h.AI.GenerateJSON(ctx, systemPrompt, userPrompt, schema)
	if err != nil {
		handleAIError(c, err)
		return
	}
	var aiResult struct {
		Explanation string                 `json:"explanation"`
		Signals     []recommendationSignal `json:"signals"`
	}
	if err := json.Unmarshal(payload, &aiResult); err != nil {
		response.Error(c, http.StatusBadGateway, "ai_invalid_response", map[string]string{"error": err.Error()})
		return
	}

	resp := userRecommendationExplainResponse{
		CanteenID:   canteenUUID.String(),
		MenuItemID:  menuItemID,
		Explanation: aiResult.Explanation,
		Signals:     aiResult.Signals,
		GeneratedAt: timeToString(pgtype.Timestamptz{Time: endAt, Valid: true}),
		Model:       h.AI.Model(),
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AIHandler) AdminOversight(c *gin.Context) {
	role, ok := httpcontext.UserRole(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	if role != "ADMIN" {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}
	if !h.ensureAIConfigured(c) {
		return
	}
	days, ok := parseDays(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	canteens, err := h.Queries.ListCanteensBasic(ctx)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "internal_error", nil)
		return
	}
	endAt := h.Clock().UTC()
	startAt := endAt.AddDate(0, 0, -days)
	prevStart := startAt.AddDate(0, 0, -days)

	candidates := make([]map[string]interface{}, 0)
	nameByID := map[string]string{}
	for _, canteen := range canteens {
		nameByID[canteen.ID.String()] = canteen.Name
		current, err := h.Queries.ListFeedbackTextsByCanteenAndRange(ctx, db.ListFeedbackTextsByCanteenAndRangeParams{
			CanteenID: canteen.ID,
			StartAt:   pgtype.Timestamptz{Time: startAt, Valid: true},
			EndAt:     pgtype.Timestamptz{Time: endAt, Valid: true},
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		if len(current) == 0 {
			continue
		}
		previous, err := h.Queries.ListFeedbackTextsByCanteenAndRange(ctx, db.ListFeedbackTextsByCanteenAndRangeParams{
			CanteenID: canteen.ID,
			StartAt:   pgtype.Timestamptz{Time: prevStart, Valid: true},
			EndAt:     pgtype.Timestamptz{Time: startAt, Valid: true},
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		currentMetrics := calculateFeedbackMetrics(current)
		if currentMetrics.RatingCount < 3 {
			continue
		}
		previousMetrics := calculateFeedbackMetrics(previous)
		currentRate := negativeRate(currentMetrics)
		previousRate := negativeRate(previousMetrics)
		delta := currentRate - previousRate
		if currentRate < 0.35 && delta < 0.2 {
			continue
		}
		stats, err := h.Queries.CanteenOrdersStatsByRange(ctx, db.CanteenOrdersStatsByRangeParams{
			CanteenID: canteen.ID,
			StartAt:   pgtype.Timestamptz{Time: startAt, Valid: true},
			EndAt:     pgtype.Timestamptz{Time: endAt, Valid: true},
		})
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal_error", nil)
			return
		}
		candidates = append(candidates, map[string]interface{}{
			"canteen_id":          canteen.ID.String(),
			"canteen_name":        canteen.Name,
			"feedback_count":      currentMetrics.Count,
			"rating_count":        currentMetrics.RatingCount,
			"avg_rating":          currentMetrics.AvgRating,
			"negative_rate":       currentRate,
			"negative_rate_delta": delta,
			"rejected_payments":   stats.RejectedPayments,
			"refunded_payments":   stats.RefundedPayments,
			"samples":             sampleFeedbackTexts(current, 15),
		})
	}
	if len(candidates) == 0 {
		resp := adminOversightResponse{
			RangeDays:   days,
			Watchlist:   []adminWatchlistItem{},
			GeneratedAt: timeToString(pgtype.Timestamptz{Time: endAt, Valid: true}),
			Model:       h.AI.Model(),
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	input := map[string]interface{}{
		"range_days": days,
		"candidates": candidates,
	}
	userPrompt := mustJSON(input)
	systemPrompt := "Return JSON only. Summarize risk signals without accusations. Use aggregate signals and short neutral phrasing. Do not include personal data."
	schema := adminOversightSchema()
	payload, err := h.AI.GenerateJSON(ctx, systemPrompt, userPrompt, schema)
	if err != nil {
		handleAIError(c, err)
		return
	}
	var aiResult struct {
		Watchlist []adminWatchlistItem `json:"watchlist"`
	}
	if err := json.Unmarshal(payload, &aiResult); err != nil {
		response.Error(c, http.StatusBadGateway, "ai_invalid_response", map[string]string{"error": err.Error()})
		return
	}
	for i := range aiResult.Watchlist {
		if aiResult.Watchlist[i].CanteenName == "" {
			aiResult.Watchlist[i].CanteenName = nameByID[aiResult.Watchlist[i].CanteenID]
		}
	}

	resp := adminOversightResponse{
		RangeDays:   days,
		Watchlist:   aiResult.Watchlist,
		GeneratedAt: timeToString(pgtype.Timestamptz{Time: endAt, Valid: true}),
		Model:       h.AI.Model(),
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AIHandler) ensureAIConfigured(c *gin.Context) bool {
	if h.AI == nil || !h.AI.Configured() {
		response.Error(c, http.StatusServiceUnavailable, "ai_not_configured", nil)
		return false
	}
	return true
}

func parseDays(c *gin.Context) (int, bool) {
	value := c.Query("days")
	if value == "" {
		return 30, true
	}
	days, err := strconv.Atoi(value)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "validation_error", map[string]string{"days": "invalid"})
		return 0, false
	}
	if days < 7 || days > 90 {
		response.Error(c, http.StatusBadRequest, "validation_error", map[string]string{"days": "out_of_range"})
		return 0, false
	}
	return days, true
}

func mustJSON(value interface{}) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func sampleFeedbackTexts(rows []db.ListFeedbackTextsByCanteenAndRangeRow, limit int) []string {
	items := make([]string, 0, limit)
	for _, row := range rows {
		if !row.Comment.Valid {
			continue
		}
		text := compactText(row.Comment.String)
		if text == "" {
			continue
		}
		if len(text) > 120 {
			text = text[:120]
		}
		items = append(items, text)
		if len(items) >= limit {
			break
		}
	}
	return items
}

func compactText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

type feedbackMetrics struct {
	Count       int32
	RatingCount int32
	Positive    int32
	Neutral     int32
	Negative    int32
	AvgRating   float64
}

func calculateFeedbackMetrics(rows []db.ListFeedbackTextsByCanteenAndRangeRow) feedbackMetrics {
	metrics := feedbackMetrics{Count: int32(len(rows))}
	var total int64
	for _, row := range rows {
		if !row.Rating.Valid {
			continue
		}
		metrics.RatingCount++
		value := row.Rating.Int32
		if value <= 2 {
			metrics.Negative++
		} else if value == 3 {
			metrics.Neutral++
		} else {
			metrics.Positive++
		}
		total += int64(value)
	}
	if metrics.RatingCount > 0 {
		metrics.AvgRating = float64(total) / float64(metrics.RatingCount)
	}
	return metrics
}

func negativeRate(metrics feedbackMetrics) float64 {
	if metrics.RatingCount == 0 {
		return 0
	}
	return float64(metrics.Negative) / float64(metrics.RatingCount)
}

type menuStats struct {
	OverallAvg float64
	TotalCount int32
	ItemAvg    map[string]float64
	ItemCount  map[string]int32
	TopItems   []map[string]interface{}
}

func buildMenuStats(rows []db.AggregateMenuRatingsByCanteenRow) menuStats {
	stats := menuStats{ItemAvg: map[string]float64{}, ItemCount: map[string]int32{}}
	var total int64
	var count int64
	for _, row := range rows {
		stats.ItemAvg[row.MenuItemID.String()] = row.AvgRating
		stats.ItemCount[row.MenuItemID.String()] = row.RatingCount
		total += int64(row.AvgRating * float64(row.RatingCount))
		count += int64(row.RatingCount)
	}
	if count > 0 {
		stats.OverallAvg = float64(total) / float64(count)
		stats.TotalCount = int32(count)
	}
	stats.TopItems = topMenuItems(rows)
	return stats
}

func topMenuItems(rows []db.AggregateMenuRatingsByCanteenRow) []map[string]interface{} {
	items := make([]map[string]interface{}, 0, 3)
	for _, row := range rows {
		items = append(items, map[string]interface{}{
			"menu_item_id": row.MenuItemID.String(),
			"avg_rating":   row.AvgRating,
			"rating_count": row.RatingCount,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		ai := items[i]
		aj := items[j]
		if ai["avg_rating"].(float64) == aj["avg_rating"].(float64) {
			return ai["rating_count"].(int32) > aj["rating_count"].(int32)
		}
		return ai["avg_rating"].(float64) > aj["avg_rating"].(float64)
	})
	if len(items) > 3 {
		items = items[:3]
	}
	return items
}

func trimQuotes(items []string, limit int) []string {
	trimmed := make([]string, 0, len(items))
	for _, item := range items {
		text := compactText(item)
		if text == "" {
			continue
		}
		if len(text) > limit {
			text = text[:limit]
		}
		trimmed = append(trimmed, text)
	}
	return trimmed
}

func ownerInsightsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"summary": map[string]interface{}{"type": "string"},
			"sentiment": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"label": map[string]interface{}{"type": "string"},
					"score": map[string]interface{}{"type": "number"},
				},
				"required": []string{"label", "score"},
			},
			"top_themes": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"theme": map[string]interface{}{"type": "string"},
						"count": map[string]interface{}{"type": "integer"},
					},
					"required": []string{"theme", "count"},
				},
			},
			"action_items": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"notable_quotes": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
		"required": []string{"summary", "sentiment", "top_themes", "action_items", "notable_quotes"},
	}
}

func recommendationSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"explanation": map[string]interface{}{"type": "string"},
			"signals": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"signal": map[string]interface{}{"type": "string"},
						"value":  map[string]interface{}{"type": "string"},
					},
					"required": []string{"signal", "value"},
				},
			},
		},
		"required": []string{"explanation", "signals"},
	}
}

func adminOversightSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"watchlist": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"canteen_id":   map[string]interface{}{"type": "string"},
						"canteen_name": map[string]interface{}{"type": "string"},
						"risk_summary": map[string]interface{}{"type": "string"},
						"top_complaints": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"type": "string"},
						},
						"suggested_followups": map[string]interface{}{
							"type":  "array",
							"items": map[string]interface{}{"type": "string"},
						},
					},
					"required": []string{"canteen_id", "canteen_name", "risk_summary", "top_complaints", "suggested_followups"},
				},
			},
		},
		"required": []string{"watchlist"},
	}
}

func handleAIError(c *gin.Context, err error) {
	var invalid ai.InvalidResponseError
	if errors.As(err, &invalid) {
		response.Error(c, http.StatusBadGateway, "ai_invalid_response", map[string]string{"error": invalid.Message})
		return
	}
	if err.Error() == "ai_not_configured" {
		response.Error(c, http.StatusServiceUnavailable, "ai_not_configured", nil)
		return
	}
	response.Error(c, http.StatusBadGateway, "ai_upstream_error", nil)
}
