package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ExportIntegrationTestSuite struct {
	suite.Suite
}

func TestExportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExportIntegrationTestSuite))
}
