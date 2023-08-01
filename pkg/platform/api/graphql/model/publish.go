package model

type PublishResult struct {
	Publish struct {
		ErrorResponse
		IngredientID        string `json:"ingredientID"`
		IngredientVersionID string `json:"ingredientVersionID"`
		Revision            int    `json:"revision"`
	} `json:"publish"`
}
