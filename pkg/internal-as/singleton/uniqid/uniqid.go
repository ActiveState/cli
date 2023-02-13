// Package uniqid exposes some of the singleton/uniqid internal package.
package uniqid

import "github.com/ActiveState/cli/internal/singleton/uniqid"

func Text() string {
	return uniqid.Text()
}
