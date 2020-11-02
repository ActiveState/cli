package prompt

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type Select struct {
	*survey.Select
}

func (s *Select) Cleanup(interface{}) error {
	terminal.CursorNextLine(1)
	return nil
}

type Input struct {
	*survey.Input
}

func (i *Input) Cleanup(val interface{}) error {
	terminal.CursorNextLine(1)
	return nil
}

type Password struct {
	*survey.Password
}

func (i *Password) Cleanup(val interface{}) error {
	terminal.CursorNextLine(1)
	return nil
}

type Confirm struct {
	*survey.Confirm
}

func (s *Confirm) Cleanup(interface{}) error {
	terminal.CursorNextLine(1)
	// Keep the answer
	return nil
}
