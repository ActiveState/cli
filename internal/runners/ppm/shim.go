package ppm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Shim struct {
	out         output.Outputer
	intercepted string
	replaced    string
	localeID    string
}

func NewShim(out output.Outputer) *Shim {
	return &Shim{
		out: out,
	}
}

func (s *Shim) RunPPM(args ...string) error {
	s.intercepted = "ppm"
	s.replaced = "packages"
	s.localeID = "ppm_print_forward"
	return s.shim(args...)
}

func (s *Shim) RunInstall(args ...string) error {
	s.intercepted = "install"
	s.replaced = "packages add"
	s.localeID = "ppm_print_forward_failure"
	return s.shim(args...)
}

func (s *Shim) RunUpgrade(args ...string) error {
	s.intercepted = "upgrade"
	s.replaced = "packages update"
	s.localeID = "ppm_print_forward_failure"
	return s.shim(args...)
}

func (s *Shim) RunRemove(args ...string) error {
	s.intercepted = "remove"
	s.replaced = "packages remove"
	s.localeID = "ppm_print_forward_failure"
	return s.shim(args...)
}

func (s *Shim) RunList(args ...string) error {
	s.intercepted = "list"
	s.replaced = "packages"
	s.localeID = "ppm_print_forward"
	return s.shim(args...)
}

func (s *Shim) shim(args ...string) error {
	err := s.executeShim(args...)
	if err != nil {
		analytics.EventWithLabel(analytics.CatPPMShimCmd, "error", errs.Join(err, " :: ").Error())
	} else {
		analytics.EventWithLabel(analytics.CatPPMShimCmd, "success", fmt.Sprintf("intercepted=%s, replaced=%s", s.intercepted, s.replaced))
	}
	return err
}

func (s *Shim) executeShim(args ...string) error {
	pj, fail := projectfile.GetSafe()
	if fail != nil && !fail.Type.Matches(projectfile.FailNoProject) {
		return locale.WrapError(
			fail.ToError(),
			"err_ppm_get_projectfile",
			"Encountered unexpected error parsing your activestate.yaml, please manually correct the yaml or if the issue is out of your hands consider letting us know about your issue at {{.V0}}",
			constants.ForumsURL,
		)
	}
	if pj == nil {
		// TODO: Replace this function call when conversion flow is complete
		analytics.Event(analytics.CatPPMShimCmd, "tutorial")
		return tutorial()
	}

	commands := strings.Split(s.replaced, " ")
	replacedArgs := args
	if len(commands) > 1 {
		s.replaced = commands[0]
		replacedArgs = commands[1:]
		replacedArgs = append(replacedArgs, args...)
	}

	forwarded := []string{"state", s.replaced}
	forwarded = append(forwarded, replacedArgs...)
	s.out.Print(locale.Tr(s.localeID, strings.Join(forwarded, " "), s.intercepted))
	return runbits.Invoke(s.out, forwarded[1:]...)
}

func invoke(command string, args ...string) error {
	executable, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_invoke_executable", "Could not find State Tool executable")
	}

	commandArgs := []string{command}
	commandArgs = append(commandArgs, args...)
	cmd := exec.Command(executable, commandArgs...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdout
	return cmd.Run()
}

func tutorial() error {
	// Placeholder until conversion flow is complete
	return nil
}
