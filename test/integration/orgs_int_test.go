package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type OrganizationsIntegrationTestSuite struct {
	suite.Suite
}

func TestOrganizationsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsIntegrationTestSuite))
}
