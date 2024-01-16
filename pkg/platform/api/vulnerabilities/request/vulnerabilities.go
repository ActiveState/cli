package request

type Ingredient struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

type vulnerabilities struct {
	vars map[string]interface{}
}

func VulnerabilitiesByIngredients(ingredients []*Ingredient) *vulnerabilities {
	return &vulnerabilities{vars: map[string]interface{}{
		"ingredients": ingredients,
	}}
}

func (p *vulnerabilities) Query() string {
	return `query q($ingredients: jsonb) {
vulnerabilities: vulnerable_ingredients_filter(
  args: {ingredient_versions: $ingredients}
  ) {
    name
    primary_namespace
    version
    vulnerability {
      severity
      cve_identifier
      source
    }
    vulnerability_id
  }
}`
}

func (p *vulnerabilities) Vars() (map[string]interface{}, error) {
	return p.vars, nil
}
