package model

import "time"

type Recipe struct {
	ID                  int       `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	EstimatedTotalPrice int       `json:"estimated_total_price"`
	CreatedAt           time.Time `json:"created_at"`
}

type RecipeIngredient struct {
	ID             int    `json:"id"`
	RecipeID       int    `json:"recipe_id"`
	Name           string `json:"name"`
	Quantity       string `json:"quantity"`
	Unit           string `json:"unit"`
	EstimatedPrice int    `json:"estimated_price"`
}

type UserMenu struct {
	ID         int       `json:"id"`
	UserID     string    `json:"user_id"`
	RecipeID   int       `json:"recipe_id"`
	SelectedAt time.Time `json:"selected_at"`
	Status     string    `json:"status"` // "active" | "completed" | "archived"
}

type ShoppingList struct {
	ID                  int       `json:"id"`
	UserMenuID          int       `json:"user_menu_id"`
	UserID              string    `json:"user_id"`
	TotalEstimatedPrice int       `json:"total_estimated_price"`
	CreatedAt           time.Time `json:"created_at"`
}

type ShoppingListItem struct {
	ID             int        `json:"id"`
	ShoppingListID int        `json:"shopping_list_id"`
	IngredientName string     `json:"ingredient_name"`
	Quantity       string     `json:"quantity"`
	Unit           string     `json:"unit"`
	EstimatedPrice int        `json:"estimated_price"`
	IsChecked      bool       `json:"is_checked"`
	CheckedAt      *time.Time `json:"checked_at,omitempty"`
}

type SelectMenuResult struct {
	UserMenuID          int       `json:"user_menu_id"`
	ShoppingListID      int       `json:"shopping_list_id"`
	RecipeName          string    `json:"recipe_name"`
	TotalEstimatedPrice int       `json:"total_estimated_price"`
	ItemsCount          int       `json:"items_count"`
	CreatedAt           time.Time `json:"created_at"`
}

type ShoppingItem struct {
	IngredientName string `json:"ingredient_name"`
	Quantity       string `json:"quantity"`
	Unit           string `json:"unit"`
	EstimatedPrice int    `json:"estimated_price"`
	IsChecked      bool   `json:"is_checked"`
}

type ShoppingListDetail struct {
	ID                  int            `json:"id"`
	MealSelectionID     int            `json:"meal_selection_id"`
	RecipeName          string         `json:"recipe_name"`
	Items               []ShoppingItem `json:"items"`
	TotalEstimatedPrice int            `json:"total_estimated_price"`
	CreatedAt           time.Time      `json:"created_at"`
}

type HistoryItem struct {
	ID                  int       `json:"id"`
	MealSelectionID     int       `json:"meal_selection_id"`
	ShoppingListID      *int      `json:"shopping_list_id,omitempty"`
	RecipeID            int       `json:"recipe_id"`
	RecipeName          string    `json:"recipe_name"`
	Description         string    `json:"description"`
	SelectedDate        time.Time `json:"selected_date"`
	TotalEstimatedPrice int       `json:"total_estimated_price"`
	CreatedAt           time.Time `json:"created_at"`
}
