package main

import (
	"context"
	"io"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// newPrimer creates a new primer.Values instance for use with command execution
func (t *mcpServerHandler) newPrimer(projectDir string, o io.Writer) (*primer.Values, func() error, error) {
	closers := []func() error{}
	closer := func() error {
		for _, c := range closers {
			if err := c(); err != nil {
				return err
			}
		}
		return nil
	}

	cfg, err := config.New()
	if err != nil {
		return nil, closer, errs.Wrap(err, "Failed to create config")
	}
	closers = append(closers, cfg.Close)

	auth := authentication.New(cfg)
	closers = append(closers, auth.Close)

	out, err := output.New(string(output.SimpleFormatName), &output.Config{
		OutWriter:   o,
		ErrWriter:   o,
		Colored:     false,
		Interactive: false,
		ShellName:   "",
	})
	if err != nil {
		return nil, closer, errs.Wrap(err, "Failed to create output")
	}

	var pj *project.Project
	if projectDir != "" {
		pjf, err := projectfile.FromPath(projectDir)
		if err != nil {
			return nil, closer, errs.Wrap(err, "Failed to create projectfile")
		}
		pj, err = project.New(pjf, out)
		if err != nil {
			return nil, closer, errs.Wrap(err, "Failed to create project")
		}
	}

	// Set up conditional, which accesses a lot of primer data
	sshell := subshell.New(cfg)

	conditional := constraints.NewPrimeConditional(auth, pj, sshell.Shell())
	project.RegisterConditional(conditional)
	if err := project.RegisterExpander("mixin", project.NewMixin(auth).Expander); err != nil {
		logging.Debug("Could not register mixin expander: %v", err)
	}

	svcmodel := model.NewSvcModel(t.svcPort)

	if auth.AvailableAPIToken() != "" {
		jwt, err := svcmodel.GetJWT(context.Background())
		if err != nil {
			multilog.Critical("Could not get JWT: %v", errs.JoinMessage(err))
		}
		if err != nil || jwt == nil {
			// Could not authenticate; user got logged out
			auth.Logout()
		} else {
			auth.UpdateSession(jwt)
		}
	}

	return primer.New(pj, out, auth, sshell, conditional, cfg, t.ipcClient, svcmodel), closer, nil
} 