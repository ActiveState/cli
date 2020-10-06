package outputhelper

import "github.com/ActiveState/cli/internal/output"

type TestOutputer struct{}

func (o *TestOutputer) Type() output.Format      { return "" }
func (o *TestOutputer) Print(value interface{})  {}
func (o *TestOutputer) Error(value interface{})  {}
func (o *TestOutputer) Notice(value interface{}) {}
func (o *TestOutputer) Config() *output.Config   { return nil }
