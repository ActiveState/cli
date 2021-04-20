package autostart

type App struct {
	Name string
	Exec string
}

type configable interface {
	Set(key string, value interface{}) error
}

func New(name, exec string) *App {
	return &App{
		Name: name,
		Exec: exec,
	}
}
