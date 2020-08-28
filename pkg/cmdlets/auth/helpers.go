package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// Logout will clear any stored credentials
func Logout() {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
}
