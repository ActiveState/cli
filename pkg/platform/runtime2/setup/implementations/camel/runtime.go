package camel

type Setup struct {
}

func NewSetup() *Setup {
	return &Setup{}
}

func (s *Setup) PostInstall() error {
	return nil
}
