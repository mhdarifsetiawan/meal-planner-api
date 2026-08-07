package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meal-planner-api/internal/auth"
	"meal-planner-api/internal/model"
	"meal-planner-api/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type MockPriceWatchRepository struct {
	campaigns map[int]*model.PriceWatchCampaign
	items     map[int]*model.PriceWatchItem
	nextID    int
}

func NewMockPriceWatchRepository() *MockPriceWatchRepository {
	return &MockPriceWatchRepository{
		campaigns: make(map[int]*model.PriceWatchCampaign),
		items:     make(map[int]*model.PriceWatchItem),
		nextID:    1,
	}
}

func (m *MockPriceWatchRepository) CreateCampaign(ctx context.Context, campaign *model.PriceWatchCampaign) error {
	campaign.ID = m.nextID
	m.nextID++
	m.campaigns[campaign.ID] = campaign
	return nil
}

func (m *MockPriceWatchRepository) GetCampaigns(ctx context.Context, includeInactive bool) ([]model.PriceWatchCampaign, error) {
	var list []model.PriceWatchCampaign
	for _, c := range m.campaigns {
		if includeInactive || c.IsActive {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (m *MockPriceWatchRepository) GetCampaignByID(ctx context.Context, id int) (*model.PriceWatchCampaign, error) {
	c, ok := m.campaigns[id]
	if !ok {
		return nil, repository.ErrCampaignNotFound
	}
	return c, nil
}

func (m *MockPriceWatchRepository) UpdateCampaign(ctx context.Context, campaign *model.PriceWatchCampaign) error {
	if _, ok := m.campaigns[campaign.ID]; !ok {
		return repository.ErrCampaignNotFound
	}
	m.campaigns[campaign.ID] = campaign
	return nil
}

func (m *MockPriceWatchRepository) DeleteCampaign(ctx context.Context, id int) error {
	if _, ok := m.campaigns[id]; !ok {
		return repository.ErrCampaignNotFound
	}
	delete(m.campaigns, id)
	return nil
}

func (m *MockPriceWatchRepository) CreateItem(ctx context.Context, item *model.PriceWatchItem) error {
	item.ID = m.nextID
	m.nextID++
	m.items[item.ID] = item
	return nil
}

func (m *MockPriceWatchRepository) UpdateItem(ctx context.Context, item *model.PriceWatchItem) error {
	if _, ok := m.items[item.ID]; !ok {
		return repository.ErrItemNotFound
	}
	m.items[item.ID] = item
	return nil
}

func (m *MockPriceWatchRepository) GetItemByID(ctx context.Context, id int) (*model.PriceWatchItem, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, repository.ErrItemNotFound
	}
	return item, nil
}

func (m *MockPriceWatchRepository) DeleteItem(ctx context.Context, id int) error {
	if _, ok := m.items[id]; !ok {
		return repository.ErrItemNotFound
	}
	delete(m.items, id)
	return nil
}

func (m *MockPriceWatchRepository) CreateSubmission(ctx context.Context, sub *model.PriceSubmission) error {
	sub.ID = m.nextID
	m.nextID++
	sub.Status = "pending"
	return nil
}

func (m *MockPriceWatchRepository) GetActiveCampaignsWithItems(ctx context.Context) ([]model.PriceWatchCampaign, error) {
	return m.GetCampaigns(ctx, false)
}

func (m *MockPriceWatchRepository) GetUserSubmissions(ctx context.Context, userID string) ([]model.UserPriceSubmissionDetail, error) {
	return []model.UserPriceSubmissionDetail{}, nil
}

func setupAdminPriceWatchTestApp(repo *MockPriceWatchRepository) *fiber.App {
	app := fiber.New()
	h := NewAdminPriceWatchHandler(repo, nil)

	app.Use(func(c *fiber.Ctx) error {
		userID := c.Get("Test-User-ID")
		if userID != "" {
			c.Locals(auth.LocalUserIDKey, userID)
		}
		return c.Next()
	})

	api := app.Group("/api/v1/admin/price-watch")
	api.Post("/campaigns", h.HandleCreateCampaign)
	api.Get("/campaigns", h.HandleGetCampaigns)
	api.Get("/campaigns/:id", h.HandleGetCampaignByID)
	api.Put("/campaigns/:id", h.HandleUpdateCampaign)
	api.Delete("/campaigns/:id", h.HandleDeleteCampaign)
	api.Post("/campaigns/:id/items", h.HandleCreateItem)
	api.Put("/items/:id", h.HandleUpdateItem)
	api.Delete("/items/:id", h.HandleDeleteItem)

	return app
}

func TestAdminPriceWatchHandler_CampaignCRUD(t *testing.T) {
	repo := NewMockPriceWatchRepository()
	app := setupAdminPriceWatchTestApp(repo)

	// 1. Create Campaign
	createBody := CreateCampaignRequest{
		Title:       "Pantau Beras Ramadan",
		Description: "Monitoring beras harian",
	}
	jsonBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest("POST", "/api/v1/admin/price-watch/campaigns", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Test-User-ID", "admin-uuid-1")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201 Created, got %d", resp.StatusCode)
	}

	// 2. Get Campaigns List
	reqList := httptest.NewRequest("GET", "/api/v1/admin/price-watch/campaigns", nil)
	reqList.Header.Set("Test-User-ID", "admin-uuid-1")

	respList, err := app.Test(reqList)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if respList.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", respList.StatusCode)
	}

	// 3. Create Item
	itemBody := CreateItemRequest{
		IngredientName: "Beras SPHP",
		Unit:           "kg",
		DisplayOrder:   1,
	}
	itemBytes, _ := json.Marshal(itemBody)

	reqItem := httptest.NewRequest("POST", "/api/v1/admin/price-watch/campaigns/1/items", bytes.NewReader(itemBytes))
	reqItem.Header.Set("Content-Type", "application/json")
	reqItem.Header.Set("Test-User-ID", "admin-uuid-1")

	respItem, err := app.Test(reqItem)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if respItem.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201 Created, got %d", respItem.StatusCode)
	}
}
