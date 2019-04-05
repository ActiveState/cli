package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var Prompter prompt.Prompter

func init() {
	Prompter = prompt.New()
}

// Logout will clear any stored credentials
func Logout() {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
}
