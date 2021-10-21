package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type OrganizationsIntegrationTestSuite struct {
	tagsuite.Suite
}

func TestOrganizationsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsIntegrationTestSuite))
}
