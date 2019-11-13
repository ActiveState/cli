package cmdtree

import (
	"strings"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/initialize"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/spf13/viper"
)

type InitArgs struct {
	Namespace string
}

func (args *InitArgs) Prepare() error {
	if args.Namespace == "" {
		return failures.FailUserInput.New("err_init_must_provide_namespace")
	}

	return nil
}

type InitOpts struct {
	Language string
	Skeleton string
	Path     string
}

func newInitCommand() *captain.Command {
	initRunner := initialize.NewInit(viper.GetViper())

	var (
		args = InitArgs{}
		opts = InitOpts{}
	)
	return captain.NewCommand(
		"init",
		locale.T("init_description"),
		[]*captain.Flag{
			{
				Name:        "path",
				Shorthand:   "",
				Description: locale.T("arg_state_init_path_description"),
				Type:        captain.TypeString,
				StringVar:   &opts.Path,
			},
			{
				Name:        "language",
				Shorthand:   "",
				Description: locale.T("flag_state_init_language_description"),
				Type:        captain.TypeString,
				StringVar:   &opts.Language,
			},
			{
				Name:        "skeleton",
				Shorthand:   "",
				Description: locale.T("flag_state_init_skeleton_description"),
				Type:        captain.TypeString,
				StringVar:   &opts.Skeleton,
			},
		},
		[]*captain.Argument{
			&captain.Argument{
				Name:        locale.T("arg_state_init_namespace"),
				Description: locale.T("arg_state_init_namespace_description"),
				Variable:    &args.Namespace,
			},
		},
		func(ccmd *captain.Command, _ []string) error {
			err := args.Prepare()
			if err != nil {
				return err
			}

			params, err := newInitRunParams(args, opts)
			if err != nil {
				return err
			}

			return initRunner.Run(params)
		},
	)
}

func newInitRunParams(args InitArgs, opts InitOpts) (*initialize.RunParams, error) {
	ns, fail := project.ParseNamespace(args.Namespace)
	if fail != nil {
		return nil, fail
	}

	var runLang *language.Language
	if opts.Language != "" {
		lang := language.MakeByName(opts.Language)
		if lang == language.Unknown {
			return nil, failures.FailUserInput.New(
				"err_init_invalid_language",
				opts.Language, strings.Join(language.AvailableNames(), ", "),
			)
		}
		runLang = &lang
	}

	return &initialize.RunParams{
		Owner:    ns.Owner,
		Project:  ns.Project,
		Path:     opts.Path,
		Language: runLang,
		Skeleton: initialize.SkeletonStyle(opts.Skeleton),
	}, nil
}
