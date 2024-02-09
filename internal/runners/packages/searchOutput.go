package packages

import "github.com/ActiveState/cli/internal/output"

type searchOutput2 struct {
	out     output.Outputer
	details []*packageDetailsTable `opts:"verticalTable"`
}

type packageDetailsTable struct {
	Name            string `opts:"omitEmpty" locale:"search_package_name,[HEADING]Name[/RESET]" json:"name"`
	Description     string `opts:"omitEmpty" locale:"search_package_description,[HEADING]Description[/RESET]" json:"description"`
	Author          string `opts:"omitEmpty" locale:"search_package_author,[HEADING]Author[/RESET]" json:"author"`
	Authors         string `opts:"omitEmpty" locale:"search_package_authors,[HEADING]Authors[/RESET]" json:"authors"`
	Website         string `opts:"omitEmpty" locale:"search_package_website,[HEADING]Website[/RESET]" json:"website"`
	License         string `opts:"omitEmpty" locale:"search_package_license,[HEADING]License[/RESET]" json:"license"`
	Versions        string `opts:"omitEmpty" locale:"search_package_versions,[HEADING]Versions[/RESET]" json:"versions"`
	Vulnerabilities string `opts:"omitEmpty" locale:"search_package_vulnerabilities,[HEADING]Vulnerabilities[/RESET]" json:"vulnerabilities"`
}
