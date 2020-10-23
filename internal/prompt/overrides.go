package prompt

import (
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type Select struct {
	*survey.Select
}

func (s *Select) Cleanup(interface{}) error {
	// Keep the answer
	return nil
}

type Input struct {
	*survey.Input
}

func (i *Input) Cleanup(val interface{}) error {
	// render the template summarizing the current state
	out, err := core.RunTemplate(survey.InputQuestionTemplate, survey.InputTemplateData{Input: *i.Input, Answer: val.(string), ShowAnswer: true})
	if err != nil {
		return err
	}

	// print the summary
	terminal.Print(out)

	// nothing went wrong
	return nil
}

type Confirm struct {
	*survey.Confirm
}

func (s *Confirm) Cleanup(interface{}) error {
	// Keep the answer
	return nil
}
