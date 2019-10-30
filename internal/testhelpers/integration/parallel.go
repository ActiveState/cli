package integration

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"
)

// RunParallel is a drop-in replacement for testify/suite.Run that runs all tests in parallel.
// It uses reflection to create a new instance of the suite for each test
//
// Origin: https://gist.github.com/dansimau/128826e692d7834eb594bb7fd41d2926
func RunParallel(t *testing.T, s suite.TestingSuite) {
	if _, ok := s.(suite.SetupAllSuite); ok {
		t.Log("Warning: SetupSuite exists but not being run in parallel mode.")
	}
	if _, ok := s.(suite.TearDownAllSuite); ok {
		t.Log("Warning: TearDownSuite exists but not being run in parallel mode.")
	}

	methodFinder := reflect.TypeOf(s)
	tests := []testing.InternalTest{}
	for index := 0; index < methodFinder.NumMethod(); index++ {
		method := methodFinder.Method(index)
		ok, err := methodFilter(method.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "testify: invalid regexp for -m: %s\n", err)
			os.Exit(1)
		}

		if ok {
			newSuiteVal := reflect.New(reflect.TypeOf(reflect.ValueOf(s).Elem().Interface()))

			setupTest, _ := methodFinder.MethodByName("SetupTest")
			teardownTest, _ := methodFinder.MethodByName("TearDownTest")
			setT, _ := methodFinder.MethodByName("SetT")

			test := testing.InternalTest{
				Name: method.Name,
				F: func(t *testing.T) {
					t.Parallel()

					setT.Func.Call([]reflect.Value{newSuiteVal, reflect.ValueOf(t)})

					if setupTest.Func.IsValid() {
						setupTest.Func.Call([]reflect.Value{newSuiteVal})
					}

					defer func() {
						if teardownTest.Func.IsValid() {
							teardownTest.Func.Call([]reflect.Value{newSuiteVal})
						}
					}()

					method.Func.Call([]reflect.Value{newSuiteVal})
				},
			}
			tests = append(tests, test)
		}
	}

	if !testing.RunTests(func(_, _ string) (bool, error) { return true, nil },
		tests) {
		t.Fail()
	}
}

// Filtering method according to set regular expression
// specified command-line argument -m
func methodFilter(name string) (bool, error) {
	if ok, _ := regexp.MatchString("^Test", name); !ok {
		return false, nil
	}
	matchMethod := flag.Lookup("testify.m").Value.(flag.Getter).Get().(string)
	return regexp.MatchString(matchMethod, name)
}
