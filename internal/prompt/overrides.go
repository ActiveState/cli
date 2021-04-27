package prompt

import (
	"gopkg.in/AlecAivazis/survey.v1"
)

type Select struct {
	*survey.Select
}

func (s *Select) Cleanup(interface{}) error {
	s.NewCursor().NextLine(1)
	return nil
}

type Input struct {
	*survey.Input
}

func (i *Input) Cleanup(val interface{}) error {
	i.NewCursor().NextLine(1)
	return nil
}

type Password struct {
	*survey.Password
}

func (i *Password) Cleanup(val interface{}) error {
	i.NewCursor().NextLine(1)
	return nil
}

type Confirm struct {
	*survey.Confirm
}

func (s *Confirm) Cleanup(interface{}) error {
	s.NewCursor().NextLine(1)
	// Keep the answer
	return nil
}
