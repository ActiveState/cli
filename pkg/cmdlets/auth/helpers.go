package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// Prompter is accessible so tests can overwrite it with Mock.  Do not use if you're not writing code for this package
var Prompter prompt.Prompter

func init() {
	Prompter = prompt.New()
}

// Logout will clear any stored credentials
func Logout() {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
}
