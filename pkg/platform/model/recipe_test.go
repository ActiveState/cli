package model_test

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/pkg/platform/model/projects"
)

func TestFetchRecipeForProject(t *testing.T) {
	fail := authentication.Get().AuthenticateWithToken("Y2UyYTU3ZDktNmJkYS00NTIwLTlkNDEtZmEwMjllMzM4NzZlJlp1Wnl1VnVMa2ViYTk4OTlb")
	assert.NoError(t, fail.ToError())

	pj, fail := projects.FetchByName("ActiveState", "ActivePython-3.5")
	assert.NoError(t, fail.ToError())

	recipe, fail := model.FetchRecipeForProject(pj)
	assert.NoError(t, fail.ToError())

	fmt.Printf("%v", recipe)
	assert.False(t, true)
}
