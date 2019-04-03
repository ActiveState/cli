package auth

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

var prompter prompt.Prompter

func init() {
	prompter = prompt.New()
}

// Logout will clear any stored credentials
func Logout() {
	authentication.Logout()
	keypairs.DeleteWithDefaults()
}
