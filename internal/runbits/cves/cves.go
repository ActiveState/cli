package cves

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	configMediator "github.com/ActiveState/cli/internal/mediators/config"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/buildplan"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
	"github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/request"
	"github.com/ActiveState/cli/pkg/platform/model"
)

func init() {
	configMediator.RegisterOption(constants.SecurityPromptConfig, configMediator.Bool, true)
	configMediator.RegisterOption(constants.SecurityPromptLevelConfig, configMediator.String, vulnModel.SeverityCritical)
}

type primeable interface {
	primer.Outputer
	primer.Prompter
	primer.Auther
	primer.Configurer
}

type CveReport struct {
	prime primeable
}

func NewCveReport(prime primeable) *CveReport {
	return &CveReport{prime}
}

func (c *CveReport) Report(newBuildPlan *buildplan.BuildPlan, oldBuildPlan *buildplan.BuildPlan, names ...string) error {
	changeset := newBuildPlan.DiffArtifacts(oldBuildPlan, false)
	if c.shouldSkipReporting(changeset) {
		logging.Debug("Skipping CVE reporting")
		return nil
	}

	var ingredients []*request.Ingredient
	for _, artifact := range changeset.Added {
		for _, ing := range artifact.Ingredients {
			ingredients = append(ingredients, &request.Ingredient{
				Namespace: ing.Namespace,
				Name:      ing.Name,
				Version:   ing.Version,
			})
		}
	}

	for _, change := range changeset.Updated {
		if !change.VersionsChanged() {
			continue // For CVE reporting we only care about ingredient changes
		}

		for _, ing := range change.To.Ingredients {
			ingredients = append(ingredients, &request.Ingredient{
				Namespace: ing.Namespace,
				Name:      ing.Name,
				Version:   ing.Version,
			})
		}
	}

	if len(names) == 0 {
		for _, ing := range ingredients {
			names = append(names, ing.Name)
		}
	}
	pg := output.StartSpinner(c.prime.Output(), locale.Tr("progress_cve_search", strings.Join(names, ", ")), constants.TerminalAnimationInterval)

	ingredientVulnerabilities, err := model.FetchVulnerabilitiesForIngredients(c.prime.Auth(), ingredients)
	if err != nil {
		return errs.Wrap(err, "Failed to retrieve vulnerabilities")
	}

	// No vulnerabilities, nothing further to do here
	if len(ingredientVulnerabilities) == 0 {
		logging.Debug("No vulnerabilities found for ingredients")
		pg.Stop(locale.T("progress_safe"))
		pg = nil
		return nil
	}

	pg.Stop(locale.T("progress_unsafe"))
	pg = nil

	vulnerabilities := model.CombineVulnerabilities(ingredientVulnerabilities, names...)
	c.summarizeCVEs(vulnerabilities)

	if c.prime.Prompt() != nil && c.shouldPromptForSecurity(vulnerabilities) {
		cont, err := c.promptForSecurity()
		if err != nil {
			return errs.Wrap(err, "Failed to prompt for security")
		}

		if !cont {
			if !c.prime.Prompt().IsInteractive() {
				return errs.AddTips(
					locale.NewInputError("err_pkgop_security_prompt", "Operation aborted due to security prompt"),
					locale.Tl("more_info_prompt", "To disable security prompting run: [ACTIONABLE]state config set security.prompt.enabled false[/RESET]"),
				)
			}
			return locale.NewInputError("err_pkgop_security_prompt", "Operation aborted due to security prompt")
		}
	}

	return nil
}

func (c *CveReport) shouldSkipReporting(changeset buildplan.ArtifactChangeset) bool {
	if !c.prime.Auth().Authenticated() {
		return true
	}

	return len(changeset.Added) == 0 && len(changeset.Updated) == 0
}

func (c *CveReport) shouldPromptForSecurity(vulnerabilities model.VulnerableIngredientsByLevels) bool {
	if !c.prime.Config().GetBool(constants.SecurityPromptConfig) || vulnerabilities.Count == 0 {
		return false
	}

	promptLevel := c.prime.Config().GetString(constants.SecurityPromptLevelConfig)

	logging.Debug("Prompt level: ", promptLevel)
	switch promptLevel {
	case vulnModel.SeverityCritical:
		return vulnerabilities.Critical.Count > 0
	case vulnModel.SeverityHigh:
		return vulnerabilities.Critical.Count > 0 ||
			vulnerabilities.High.Count > 0
	case vulnModel.SeverityMedium:
		return vulnerabilities.Critical.Count > 0 ||
			vulnerabilities.High.Count > 0 ||
			vulnerabilities.Medium.Count > 0
	case vulnModel.SeverityLow:
		return vulnerabilities.Critical.Count > 0 ||
			vulnerabilities.High.Count > 0 ||
			vulnerabilities.Medium.Count > 0 ||
			vulnerabilities.Low.Count > 0
	}

	return false
}

func (c *CveReport) summarizeCVEs(vulnerabilities model.VulnerableIngredientsByLevels) {
	out := c.prime.Output()
	out.Print("")

	switch {
	case vulnerabilities.CountPrimary == 0:
		out.Print("   " + locale.Tr("warning_vulnerable_indirectonly", strconv.Itoa(vulnerabilities.Count)))
	case vulnerabilities.CountPrimary == vulnerabilities.Count:
		out.Print("   " + locale.Tr("warning_vulnerable_directonly", strconv.Itoa(vulnerabilities.Count)))
	default:
		out.Print("   " + locale.Tr("warning_vulnerable", strconv.Itoa(vulnerabilities.CountPrimary), strconv.Itoa(vulnerabilities.Count-vulnerabilities.CountPrimary)))
	}

	printVulnerabilities := func(vulnerableIngredients model.VulnerableIngredientsByLevel, name, color string) {
		if vulnerableIngredients.Count > 0 {
			ings := []string{}
			for _, vulns := range vulnerableIngredients.Ingredients {
				prefix := ""
				if vulnerabilities.Count > vulnerabilities.CountPrimary {
					prefix = fmt.Sprintf("%s@%s: ", vulns.IngredientName, vulns.IngredientVersion)
				}
				ings = append(ings, fmt.Sprintf("%s[CYAN]%s[/RESET]", prefix, strings.Join(vulns.CVEIDs, ", ")))
			}
			out.Print(fmt.Sprintf("    • [%s]%d %s:[/RESET] %s", color, vulnerableIngredients.Count, name, strings.Join(ings, ", ")))
		}
	}

	printVulnerabilities(vulnerabilities.Critical, locale.Tl("cve_critical", "Critical"), "RED")
	printVulnerabilities(vulnerabilities.High, locale.Tl("cve_high", "High"), "ORANGE")
	printVulnerabilities(vulnerabilities.Medium, locale.Tl("cve_medium", "Medium"), "YELLOW")
	printVulnerabilities(vulnerabilities.Low, locale.Tl("cve_low", "Low"), "MAGENTA")

	out.Print("")
	out.Print("   " + locale.T("more_info_vulnerabilities"))
	out.Print("   " + locale.T("disable_prompting_vulnerabilities"))
}

func (c *CveReport) promptForSecurity() (bool, error) {
	confirm, err := c.prime.Prompt().Confirm("", locale.Tr("prompt_continue_pkg_operation"), ptr.To(false))
	if err != nil {
		return false, locale.WrapError(err, "err_pkgop_confirm", "Need a confirmation.")
	}

	return confirm, nil
}