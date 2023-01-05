package app

type App struct {
	Name string
	Exec string
	Args []string
}

func New(name string, exec string, args []string) (*App, error) {
	return &App{
		Name: name,
		Exec: exec,
		Args: args,
	}, nil
}
