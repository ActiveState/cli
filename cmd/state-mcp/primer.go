package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constraints"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/ipc"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// newPrimer creates a new primer.Values instance for use with command execution
func newPrimer() (*primer.Values, func() error, error) {
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
		OutWriter:   io.Discard, // We use Outputer.History() instead
		ErrWriter:   io.Discard, // We use Outputer.History() instead
		Colored:     false,
		Interactive: false,
		ShellName:   "",
	})
	if err != nil {
		return nil, closer, errs.Wrap(err, "Failed to create output")
	}

	// Set up conditional, which accesses a lot of primer data
	sshell := subshell.New(cfg)

	conditional := constraints.NewPrimeConditional(auth, nil, sshell.Shell())
	project.RegisterConditional(conditional)
	if err := project.RegisterExpander("mixin", project.NewMixin(auth).Expander); err != nil {
		logging.Debug("Could not register mixin expander: %v", err)
	}

	ipcClient, svcPort, err := connectToSvc()
	if err != nil {
		return nil, closer, errs.Wrap(err, "Failed to connect to service")
	}

	svcmodel := model.NewSvcModel(svcPort)

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

	return primer.New(out, auth, sshell, conditional, cfg, ipcClient, svcmodel), closer, nil
}


type stdOutput struct{}

func (s *stdOutput) Notice(msg interface{}) {
	logging.Info(fmt.Sprintf("%v", msg))
}

// connectToSvc connects to the state service and returns an IPC client
func connectToSvc() (*ipc.Client, string, error) {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return nil, "", errs.Wrap(err, "Could not get service info")
	}

	ipcClient := svcctl.NewDefaultIPCClient()
	argText := strings.Join(os.Args, " ")
	svcPort, err := svcctl.EnsureExecStartedAndLocateHTTP(ipcClient, svcExec, argText, &stdOutput{})
	if err != nil {
		return nil, "", errs.Wrap(err, "Failed to start state-svc at state tool invocation")
	}

	return ipcClient, svcPort, nil
}