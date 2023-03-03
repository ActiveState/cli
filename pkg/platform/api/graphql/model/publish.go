package model

type PublishResult struct {
	IngredientID        string `json:"ingredientID"`
	IngredientVersionID string `json:"ingredientVersionID"`
	Revision            int    `json:"revision"`
}
