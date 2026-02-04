package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "freepass-2026/internal/sqlc/gen"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeAIClient struct {
	configured bool
	payload    []byte
	err        error
	called     bool
}

func (f *fakeAIClient) Configured() bool {
	return f.configured
}

func (f *fakeAIClient) Model() string {
	return "gemini-2.5-flash"
}

func (f *fakeAIClient) GenerateJSON(ctx context.Context, systemPrompt string, userPrompt string, schema map[string]interface{}) ([]byte, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	return f.payload, nil
}

type fakeAIQueries struct {
	canteen   db.Canteen
	menuItem  db.MenuItem
	feedbacks []db.ListFeedbackTextsByCanteenAndRangeRow
	menuAgg   []db.AggregateMenuRatingsByCanteenRow
	stats     db.CanteenOrdersStatsByRangeRow
	canteens  []db.ListCanteensBasicRow
	getErr    error
}

func (f *fakeAIQueries) GetCanteenByID(ctx context.Context, id pgtype.UUID) (db.Canteen, error) {
	return f.canteen, f.getErr
}

func (f *fakeAIQueries) GetMenuItemByID(ctx context.Context, id pgtype.UUID) (db.MenuItem, error) {
	return f.menuItem, f.getErr
}

func (f *fakeAIQueries) ListFeedbackTextsByCanteenAndRange(ctx context.Context, arg db.ListFeedbackTextsByCanteenAndRangeParams) ([]db.ListFeedbackTextsByCanteenAndRangeRow, error) {
	return f.feedbacks, f.getErr
}

func (f *fakeAIQueries) AggregateMenuRatingsByCanteen(ctx context.Context, canteenID pgtype.UUID) ([]db.AggregateMenuRatingsByCanteenRow, error) {
	return f.menuAgg, f.getErr
}

func (f *fakeAIQueries) CanteenOrdersStatsByRange(ctx context.Context, arg db.CanteenOrdersStatsByRangeParams) (db.CanteenOrdersStatsByRangeRow, error) {
	return f.stats, f.getErr
}

func (f *fakeAIQueries) ListCanteensBasic(ctx context.Context) ([]db.ListCanteensBasicRow, error) {
	return f.canteens, f.getErr
}

func TestOwnerFeedbackInsightsDaysValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ownerID := mustUUID("11111111-1111-1111-1111-111111111111")
	canteenID := mustUUID("22222222-2222-2222-2222-222222222222")
	queries := &fakeAIQueries{canteen: db.Canteen{ID: canteenID, OwnerID: ownerID}}
	client := &fakeAIClient{configured: true, payload: []byte(`{"summary":"ok","sentiment":{"label":"neutral","score":0},"top_themes":[],"action_items":[],"notable_quotes":[]}`)}
	h := NewAIHandler(queries, client, func() time.Time { return time.Unix(0, 0).UTC() })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/owner/canteens/"+canteenID.String()+"/ai/feedback-insights?days=2", nil)
	c.Params = []gin.Param{{Key: "canteenId", Value: canteenID.String()}}
	c.Set("user_role", "OWNER")
	c.Set("user_id", ownerID.String())

	h.OwnerFeedbackInsights(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOwnerFeedbackInsightsOwnershipForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ownerID := mustUUID("11111111-1111-1111-1111-111111111111")
	otherOwner := mustUUID("33333333-3333-3333-3333-333333333333")
	canteenID := mustUUID("22222222-2222-2222-2222-222222222222")
	queries := &fakeAIQueries{canteen: db.Canteen{ID: canteenID, OwnerID: otherOwner}}
	client := &fakeAIClient{configured: true, payload: []byte(`{"summary":"ok","sentiment":{"label":"neutral","score":0},"top_themes":[],"action_items":[],"notable_quotes":[]}`)}
	h := NewAIHandler(queries, client, func() time.Time { return time.Unix(0, 0).UTC() })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/owner/canteens/"+canteenID.String()+"/ai/feedback-insights?days=30", nil)
	c.Params = []gin.Param{{Key: "canteenId", Value: canteenID.String()}}
	c.Set("user_role", "OWNER")
	c.Set("user_id", ownerID.String())

	h.OwnerFeedbackInsights(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestUserRecommendationExplainInvalidMenuItemID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	canteenID := mustUUID("22222222-2222-2222-2222-222222222222")
	queries := &fakeAIQueries{canteen: db.Canteen{ID: canteenID}}
	client := &fakeAIClient{configured: true, payload: []byte(`{"explanation":"ok","signals":[]}`)}
	h := NewAIHandler(queries, client, func() time.Time { return time.Unix(0, 0).UTC() })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/canteens/"+canteenID.String()+"/ai/recommendation-explain?menu_item_id=invalid", nil)
	c.Params = []gin.Param{{Key: "canteenId", Value: canteenID.String()}}
	c.Set("user_role", "USER")

	h.UserRecommendationExplain(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdminOversightForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queries := &fakeAIQueries{}
	client := &fakeAIClient{configured: true, payload: []byte(`{"watchlist":[]}`)}
	h := NewAIHandler(queries, client, func() time.Time { return time.Unix(0, 0).UTC() })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/ai/oversight", nil)
	c.Set("user_role", "USER")

	h.AdminOversight(c)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAINotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	canteenID := mustUUID("22222222-2222-2222-2222-222222222222")
	queries := &fakeAIQueries{canteen: db.Canteen{ID: canteenID}}
	client := &fakeAIClient{configured: false}
	h := NewAIHandler(queries, client, func() time.Time { return time.Unix(0, 0).UTC() })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/canteens/"+canteenID.String()+"/ai/recommendation-explain", nil)
	c.Params = []gin.Param{{Key: "canteenId", Value: canteenID.String()}}
	c.Set("user_role", "USER")

	h.UserRecommendationExplain(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func mustUUID(value string) pgtype.UUID {
	var id pgtype.UUID
	_ = id.Scan(value)
	return id
}
