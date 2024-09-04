package request

import (
	"fmt"
	"strings"
	"time"
)

func SearchIngredients(namespaces []string, name string, exact bool, time *time.Time, limit, offset int) *searchIngredients {
	return &searchIngredients{map[string]interface{}{
		"namespaces": fmt.Sprintf("{%s}", strings.Join(namespaces, ",")), // API requires enclosure in {}
		"name":       name,
		"exact":      exact,
		"time":       time,
		"limit":      limit,
		"offset":     offset,
	}}
}

type searchIngredients struct {
	vars map[string]interface{}
}

func (s *searchIngredients) Query() string {
	return `
query ($namespaces: _non_empty_citext, $name: non_empty_citext, $exact: Boolean!, $time: timestamptz, $limit: Int!, $offset: Int!) {
  search_ingredients(
    args: {namespaces: $namespaces, name_: $name, exact: $exact, timestamp_: $time, limit_: $limit, offset_: $offset}
  ) {
    name
    normalized_name
    namespace {
      namespace
    }
		ingredient_id
		description
		website
    versions(order_by:{sortable_version:desc}) {
			version
			ingredient_version_id
			license_expression
		}
  }
}`
}

func (s *searchIngredients) Vars() (map[string]interface{}, error) {
	return s.vars, nil
}

func (s *searchIngredients) SetOffset(offset int) {
	s.vars["offset"] = offset
}
