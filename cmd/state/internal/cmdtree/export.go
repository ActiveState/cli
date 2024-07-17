package cmdtree

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runners/export"
	"github.com/ActiveState/cli/internal/runners/export/config"
	"github.com/ActiveState/cli/internal/runners/export/docs"
	"github.com/ActiveState/cli/internal/runners/export/ghactions"
	"github.com/ActiveState/cli/pkg/project"
)

func newExportCommand(prime *primer.Values) *captain.Command {
	runner := export.NewExport()

	return captain.NewCommand(
		"export",
		locale.Tl("export_title", "Exporting Information"),
		locale.T("export_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return runner.Run(ccmd)
		}).SetGroup(UtilsGroup).SetSupportsStructuredOutput()
}

func newJWTCommand(prime *primer.Values) *captain.Command {
	jwt := export.NewJWT(prime)

	params := export.JWTParams{}

	return captain.NewCommand(
		"jwt",
		locale.Tl("export_jwt_title", "Exporting Credentials"),
		locale.T("export_jwt_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return jwt.Run(&params)
		}).SetSupportsStructuredOutput()
}

func newPrivateKeyCommand(prime *primer.Values) *captain.Command {
	privateKey := export.NewPrivateKey(prime)

	params := export.PrivateKeyParams{}

	return captain.NewCommand(
		"private-key",
		locale.Tl("export_privkey_title", "Exporting Private Key"),
		locale.T("export_privkey_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, args []string) error {
			return privateKey.Run(&params)
		}).SetSupportsStructuredOutput()
}

func newAPIKeyCommand(prime *primer.Values) *captain.Command {
	apikey := export.NewAPIKey(prime)
	params := export.APIKeyRunParams{}

	return captain.NewCommand(
		"new-api-key",
		locale.Tl("export_new_api_key_title", "Exporting New API Key"),
		locale.T("export_new_api_key_cmd_description"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        locale.T("export_new_api_key_arg_name"),
				Description: locale.T("export_new_api_key_arg_name_description"),
				Value:       &params.Name,
				Required:    true,
			},
		},
		func(ccmd *captain.Command, args []string) error {
			params.IsAuthed = prime.Auth().Authenticated
			return apikey.Run(params)
		}).SetSupportsStructuredOutput()
}

func newExportConfigCommand(prime *primer.Values) *captain.Command {
	runner := config.New(prime)
	params := config.ConfigParams{}

	return captain.NewCommand(
		"config",
		locale.Tl("export_config_title", "Exporting Configuration Data"),
		locale.T("export_config_cmd_description"),
		prime,
		[]*captain.Flag{
			{
				Name: "filter",
				Description: locale.Tr(
					"export_config_flag_filter_description",
					strings.Join(config.SupportedFilters(), ", "),
				),
				Value: &params.Filter,
			},
		},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(ccmd, &params)
		}).SetSupportsStructuredOutput().SetUnstable(true)
}

func newExportGithubActionCommand(prime *primer.Values) *captain.Command {
	runner := ghactions.New(prime)
	params := ghactions.Params{}

	return captain.NewCommand(
		"github-actions",
		locale.Tl("export_ghactions_title", "Exporting Github Action Workflow"),
		locale.Tl("export_ghactions_description", "Create a github action workflow for your project"),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params)
		}).SetUnstable(true)
}

func newExportDocsCommand(prime *primer.Values) *captain.Command {
	runner := docs.New(prime)
	params := docs.Params{}

	cmd := captain.NewCommand(
		"_docs",
		locale.Tl("export_docs_title", "Export state tool command reference in markdown format"),
		locale.Tl("export_docs_description", ""),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(&params, ccmd)
		})

	cmd.SetHidden(true)

	return cmd
}

func newExportEnvCommand(prime *primer.Values) *captain.Command {
	runner := export.NewEnv(prime)

	cmd := captain.NewCommand(
		"env",
		locale.Tl("env_docs_title", "Exporting environment"),
		locale.Tl("env_docs_description", "Export the environment variables associated with your runtime."),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run()
		})

	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)

	return cmd
}

func newExportLogCommand(prime *primer.Values) *captain.Command {
	runner := export.NewLog(prime)
	params := &export.LogParams{}

	cmd := captain.NewCommand(
		"log",
		locale.Tl("export_log_title", "Show Log File"),
		locale.Tl("export_log_description", "Show the path to a State Tool log file"),
		prime,
		[]*captain.Flag{
			{
				Name:        "index",
				Shorthand:   "i",
				Description: locale.Tl("flag_export_log_index", "The 0-based index of the log file to show, starting with the newest"),
				Value:       &params.Index,
			},
		},
		[]*captain.Argument{
			{
				Name:        "prefix",
				Description: locale.Tl("arg_export_log_prefix", "The prefix of the log file to show (e.g. state or state-svc). The default is 'state'"),
				Required:    false,
				Value:       &params.Prefix,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(params)
		})

	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)

	return cmd
}

func newExportRuntimeCommand(prime *primer.Values) *captain.Command {
	runner := export.NewRuntime(prime)
	params := &export.RuntimeParams{}

	cmd := captain.NewCommand(
		"runtime",
		locale.Tl("export_runtime_title", "Exporting runtime"),
		locale.Tl("export_runtime_description", "Export the runtime associated with your runtime."),
		prime,
		[]*captain.Flag{},
		[]*captain.Argument{
			{
				Name:        "path",
				Description: locale.Tl("arg_export_runtime_path", "Optional path to your project's runtime if not inside your project"),
				Required:    false,
				Value:       &params.Path,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			return runner.Run(params)
		})

	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)

	return cmd
}

func newExportBuildPlanCommand(prime *primer.Values) *captain.Command {
	runner := export.NewBuildPlan(prime)
	params := &export.BuildPlanParams{Namespace: &project.Namespaced{}}

	cmd := captain.NewCommand(
		"buildplan",
		locale.Tl("export_buildplan_title", "Exporting Build Plan"),
		locale.Tl("export_buildplan_description", "Export the build plan for your project"),
		prime,
		[]*captain.Flag{
			{
				Name:        "namespace",
				Description: locale.Tl("export_buildplan_flags_namespace_description", "The namespace of the project to export the build plan for"),
				Value:       params.Namespace,
			},
			{
				Name:        "commit",
				Description: locale.Tl("export_buildplan_flags_commit_description", "The commit ID to export the build plan for"),
				Value:       &params.CommitID,
			},
			{
				Name:        "target",
				Description: locale.Tl("export_buildplan_flags_target_description", "The target to export the build plan for"),
				Value:       &params.Target,
			},
		},
		[]*captain.Argument{},
		func(_ *captain.Command, _ []string) error {
			return runner.Run(params)
		},
	)

	cmd.SetSupportsStructuredOutput()
	cmd.SetUnstable(true)

	return cmd
}
