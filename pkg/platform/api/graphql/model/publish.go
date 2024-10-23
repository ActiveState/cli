package model

type PublishResult struct {
	ErrorResponse
	IngredientID        string `json:"ingredientID"`
	IngredientVersionID string `json:"ingredientVersionID"`
	Revision            int    `json:"revision"`
}

type PublishResponse struct {
	Result PublishResult `json:"publish"`
}
