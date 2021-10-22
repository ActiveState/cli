package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
}

func (s *Suite) BeforeTest(suiteName, testName string) {
	s.T().Parallel()
}

func (s *Suite) TestA() {
	time.Sleep(time.Second)
}

func (s *Suite) TestB() {
	time.Sleep(time.Second)
}

func (s *Suite) TestC() {
	time.Sleep(time.Second)
}

func (s *Suite) TestD() {
	time.Sleep(time.Second)
}

func (s *Suite) TestE() {
	time.Sleep(time.Second)
}

func Test_Suite(t *testing.T) {
	t.Parallel()
suite.Run(t, new(Suite))
}
