package ppm

import (
	"fmt"
	"strings"

	analytics2 "github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Shim struct {
	out       output.Outputer
	project   *projectfile.Project
	analytics analytics2.Dispatcher
}

func NewShim(prime *primer.Values) *Shim {
	return &Shim{
		out:       prime.Output(),
		project:   prime.Projectfile(),
		analytics: prime.Analytics(),
	}
}

func (s *Shim) RunPPM(converted bool, args ...string) error {
	if converted {
		s.printForwardInfo("ppm", "packages", "ppm_print_forward_after_convert")
		return nil
	} else {
		s.printForwardInfo("ppm", "packages", "ppm_print_forward")
		return s.shim("ppm", "packages", args...)
	}
}

func (s *Shim) RunInstall(converted bool, args ...string) error {
	if converted {
		s.printForwardInfo("ppm install", "install", "ppm_print_forward_after_convert")
		return nil
	} else {
		s.printForwardInfo("ppm install", "install", "ppm_print_forward_failure")
		return s.shim("install", "install", args...)
	}
}

func (s *Shim) RunUpgrade(converted bool, args ...string) error {
	if converted {
		s.printForwardInfo("ppm upgrade", "install", "ppm_print_forward_after_convert")
		return nil
	} else {
		s.printForwardInfo("ppm upgrade", "install", "ppm_print_forward_failure")
		return s.shim("upgrade", "install", args...)
	}
}

func (s *Shim) RunRemove(converted bool, args ...string) error {
	if converted {
		s.printForwardInfo("ppm remove", "uninstall", "ppm_print_forward_after_convert")
		return nil
	} else {
		s.printForwardInfo("ppm remove", "uninstall", "ppm_print_forward_failure")
		return s.shim("remove", "uninstall", args...)
	}
}

func (s *Shim) RunList(converted bool, args ...string) error {
	if converted {
		s.printForwardInfo("ppm list", "packages", "ppm_print_forward_after_convert")
		return nil
	} else {
		s.printForwardInfo("ppm list", "packages", "ppm_print_forward")
		return s.shim("list", "packages", args...)
	}
}

func (s *Shim) shim(intercepted, replaced string, args ...string) error {
	err := s.executeShim(intercepted, replaced, args...)
	if err != nil {
		s.analytics.EventWithLabel(constants.CatPPMShimCmd, intercepted, fmt.Sprintf("error: %v", errs.Join(err, " :: ").Error()))
	} else {
		s.analytics.EventWithLabel(constants.CatPPMShimCmd, intercepted, "success")
	}
	return err
}

func (s *Shim) executeShim(intercepted, replaced string, args ...string) error {
	if s.project == nil {
		// TODO: Replace this function call when conversion flow is complete
		s.analytics.Event(constants.CatPPMShimCmd, "tutorial")
		return tutorial()
	}

	commands := strings.Split(replaced, " ")
	replacedArgs := args
	if len(commands) > 1 {
		replaced = commands[0]
		replacedArgs = commands[1:]
		replacedArgs = append(replacedArgs, args...)
	}

	forwarded := []string{replaced}
	forwarded = append(forwarded, replacedArgs...)
	return runbits.Invoke(s.out, forwarded...)
}

func tutorial() error {
	// Placeholder until conversion flow is complete
	return nil
}

func (s *Shim) printForwardInfo(intercepted, replaced, localeID string) {
	forwarded := []string{"state", replaced}
	s.out.Print(locale.Tr(localeID, strings.Join(forwarded, " "), intercepted))
}

func (s *Shim) PrintSuggestion(ppmIntent, newCommand, docLink string) error {
	s.out.Print(locale.Tr("ppm_print_suggestion", ppmIntent, newCommand, docLink))
	return nil
}

func (s *Shim) PrintDefault() error {
	s.out.Print(strings.TrimSpace(locale.T("ppm_header_message")))
	return nil
}

func (s *Shim) PrintMain() error {
	s.out.Print(locale.T("ppm_print_main"))
	return nil
}
