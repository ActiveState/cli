package autostart

type App struct {
	Name string
	Exec string
}

func New(name, exec string) *App {
	return &App{
		Name: name,
		Exec: exec,
	}
}
