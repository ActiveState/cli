package autostart

type App struct {
	Name string
	Exec string
	cfg  configable
}

type configable interface {
	Set(key string) error
}

func New(name, exec string, cfg configable) *App {
	return &App{
		Name: name,
		Exec: exec,
		cfg:  cfg,
	}
}
