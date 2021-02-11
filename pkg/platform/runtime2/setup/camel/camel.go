package camel

import (
	runtime "github.com/ActiveState/cli/pkg/platform/runtime2"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/common"
)

var _ common.Setuper = &Setup{}
var _ common.ArtifactSetuper = &ArtifactSetup{}

type Setup struct {
}

type ArtifactSetup struct {
}

func NewSetup() *Setup {
	return &Setup{}
}

func (s *Setup) ArtifactSetup(artifactID runtime.ArtifactID) common.ArtifactSetuper {
	return &ArtifactSetup{}
}

func (s *Setup) PostInstall() error {
	panic("implement me")
}

func (as *ArtifactSetup) NeedsSetup() bool {
	panic("implement me")
}

func (as *ArtifactSetup) Move(from string) error {
	panic("implement me")
}

func (as *ArtifactSetup) MetaDataCollection() error {
	panic("implement me")
}

func (as *ArtifactSetup) Relocate() error {
	panic("implement me")
}
