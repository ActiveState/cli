package autostart

type AppName string

func (a AppName) String() string {
	return string(a)
}

type app struct {
	Name    string
	Exec    string
	Args    []string
	cfg     Configurable
	options Options
}

type Options struct {
	LaunchFileName string
	IconFileName   string
	IconFileSource string
	GenericName    string
	Comment        string
	Keywords       string
}

type Configurable interface {
	Set(string, interface{}) error
	IsSet(string) bool
}

func New(name AppName, exec string, args []string, options Options, cfg Configurable) (*app, error) {
	return &app{
		Name:    name.String(),
		Exec:    exec,
		Args:    args,
		cfg:     cfg,
		options: options,
	}, nil
}

func (a *app) Enable() error {
	return a.enable()
}

func (a *app) Disable() error {
	return a.disable()
}
