package alternative

type Setup struct {
}

func NewSetup() *Setup {
	return &Setup{}
}

func (s *Setup) PostInstall() error {
	panic("implement me")
}
