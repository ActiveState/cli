package model

type PublishResult struct {
	ErrorResponse
	Publish struct {
		IngredientID        string `json:"ingredientID"`
		IngredientVersionID string `json:"ingredientVersionID"`
		Revision            int    `json:"revision"`
	} `json:"publish"`
}
