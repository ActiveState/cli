package protocol

import (
	"fmt"
	"net/url"
	"os"

	"github.com/skratchdot/open-golang/open"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type primeable interface {
	primer.Outputer
}

type Protocol struct {
	out output.Outputer
}

type Params struct {
	URL string
}

func New(prime primeable) *Protocol {
	return &Protocol{prime.Output()}
}

func (p *Protocol) Run(params Params) error {
	logging.Debug("Execute Protocol, URL: %s", params.URL)

	parsed, err := url.Parse(params.URL)
	if err != nil {
		return locale.WrapError(err, "err_protocol_parse", "Invailid URL provided: {{.V0}}", params.URL)
	}

	// Host=platform means we're trying to open a link to the platform
	if parsed.Host == "platform" {
		query := ""
		if parsed.RawQuery != "" {
			query = "?" + parsed.RawQuery
		}
		if err := open.Run("https://platform.activestate.com" + parsed.Path + query); err != nil {
			return locale.WrapExternalError(err, "err_protocol_open", "Could not open URL: {{.V0}}", params.URL)
		}
		return nil
	}

	// Host!=platform, then host is the org and path is the project name
	namespace, err := project.ParseNamespace(parsed.Host + parsed.Path)
	if err != nil {
		return locale.WrapError(err, "err_protocol_namespace", "{{.V0}} is not a valid namespace", parsed.Host+parsed.Path)
	}

	if parsed.Fragment != "" && parsed.Fragment != "replace" {
		return locale.NewError("err_protocol_flag", "Invalid URL fragment, the only supported URL fragment is 'replace'")
	}

	// Execute state command
	exe, err := os.Executable()
	if err != nil {
		return locale.WrapError(err, "err_protocol_os_executable", "Could not detect executable path of State Tool.")
	}
	_, _, err = osutils.ExecuteAndPipeStd(exe, []string{"activate", fmt.Sprintf("--%s", parsed.Fragment), namespace.String()}, []string{})
	return err
}
