package packages

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"

	"github.com/ActiveState/cli/pkg/projectfile"
)

var (
	yesErr = true
	noErr  = false
)

type dependencies struct {
	apiMock   *apiMock.Mock
	authMock  *authMock.Mock
	invMock   *invMock.Mock
	graphMock *graphMock.Mock
}

func (ds *dependencies) setUp() {
	ds.apiMock = apiMock.Init()
	ds.invMock = invMock.Init()
	ds.authMock = authMock.Init()
	ds.graphMock = graphMock.Init()

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	httpmock.Register("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001")
	ds.authMock.MockLoggedin()
	ds.invMock.MockIngredientsByName()
	ds.apiMock.MockCommit()
	ds.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
	ds.graphMock.Checkpoint(graphMock.NoOptions)
}

func (ds *dependencies) cleanUp() {
	ds.invMock.Close()
	ds.apiMock.Close()
	ds.authMock.Close()
	ds.graphMock.Close()
}

func handleTest(t *testing.T, out *outputhelper.Catcher, run func() error, wantContains string, wantErr bool) {
	deps := &dependencies{}
	deps.setUp()
	defer deps.cleanUp()

	err := run()
	if !wantErr && err != nil {
		t.Errorf("got %v, want nil", err)
		return
	}
	outTxt := out.Output()

	if wantErr {
		if err == nil {
			t.Errorf("got nil, want err")
			return
		}
		outTxt = err.Error()
	}

	if !strings.Contains(outTxt, wantContains) {
		t.Errorf("got %s, want (contains) %s", outTxt, wantContains)
	}
}
