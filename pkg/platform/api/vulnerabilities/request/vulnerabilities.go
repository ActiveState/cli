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
		"f": ingredients,
	}}
}

func (p *vulnerabilities) Query() string {
	return `query GetVulnerabilities($f: jsonb) {
  vulnerable_ingredients_filter(args: { ingredient_versions: $f }) {
    primary_namespace
    name
    version
    vulnerability {
      cve_identifier
      alt_identifiers
      severity
    }
  }
}`
}

func (p *vulnerabilities) Vars() (map[string]interface{}, error) {
	return p.vars, nil
}
