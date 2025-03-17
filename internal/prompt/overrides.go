package prompt

import (
	"gopkg.in/AlecAivazis/survey.v1"
)

type Select struct {
	*survey.Select
	nonInteractiveChoice *string
}

func (s *Select) Prompt() (interface{}, error) {
	if s.nonInteractiveChoice == nil {
		return s.Select.Prompt()
	}

	idx := 0
	for i, choice := range s.Select.Options {
		if choice == *s.nonInteractiveChoice {
			idx = i
			break
		}
	}

	err := s.Select.Render(
		survey.SelectQuestionTemplate,
		survey.SelectTemplateData{
			Select:        *s.Select,
			PageEntries:   s.Select.Options,
			SelectedIndex: idx,
		})
	if err != nil {
		return nil, err
	}

	return *s.nonInteractiveChoice, nil
}

func (s *Select) Cleanup(interface{}) error {
	s.NewCursor().NextLine(1)
	return nil
}

type Input struct {
	*survey.Input
	nonInteractiveResponse *string
}

func (i *Input) Prompt() (interface{}, error) {
	if i.nonInteractiveResponse == nil {
		return i.Input.Prompt()
	}

	err := i.Input.Render(
		survey.InputQuestionTemplate,
		survey.InputTemplateData{Input: *i.Input})
	if err != nil {
		return nil, err
	}

	return *i.nonInteractiveResponse, nil
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
	nonInteractiveChoice *bool
}

func (s *Confirm) Prompt() (interface{}, error) {
	if s.nonInteractiveChoice == nil {
		return s.Confirm.Prompt()
	}

	err := s.Confirm.Render(
		survey.ConfirmQuestionTemplate,
		survey.ConfirmTemplateData{Confirm: *s.Confirm})
	if err != nil {
		return nil, err
	}

	return *s.nonInteractiveChoice, nil
}

func (s *Confirm) Cleanup(interface{}) error {
	s.NewCursor().NextLine(1)
	// Keep the answer
	return nil
}
