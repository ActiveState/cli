package app

type App struct {
	Name    string
	Exec    string
	Args    []string
	options Options
}

type Options struct {
	IconFileName    string
	IconFileSource  string
	IsGUIApp        bool
	MacHideDockIcon bool // macOS plist HideDockIcon
}

func New(name string, exec string, args []string, opts Options) (*App, error) {
	return &App{
		Name:    name,
		Exec:    exec,
		Args:    args,
		options: opts,
	}, nil
}

func (a *App) Install() error {
	return a.install()
}

func (a *App) Uninstall() error {
	return a.uninstall()
}
