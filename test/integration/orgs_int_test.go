package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
	"github.com/stretchr/testify/suite"
)

type OrganizationsIntegrationTestSuite struct {
	testsuite.Suite
}

func TestOrganizationsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsIntegrationTestSuite))
}
