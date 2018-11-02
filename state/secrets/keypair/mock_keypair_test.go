package keypair_test

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/stretchr/testify/mock"
)

type MockKeypair struct {
	mock.Mock
}

func (m *MockKeypair) EncodePrivateKey() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockKeypair) EncodePublicKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func NewMockGeneratorFunc(mockKeypair *MockKeypair) keypairs.GeneratorFunc {
	return func() (keypairs.Keypair, error) {
		return mockKeypair, nil
	}
}
