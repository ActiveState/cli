package model

import (
	"regexp"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
)

type Namespace string

const (
	// NamespacePlatform is the namespace used for platform requirements
	NamespacePlatform Namespace = `^platform$`

	// NamespaceLanguage is the namespace used for language requirements
	NamespaceLanguage = `^language$`

	// NamespacePackage is the namespace used for package requirements
	NamespacePackage = `/package$`
)

var (
	// FailNoCommit is a failure due to a non-existent commit
	FailNoCommit = failures.Type("model.fail.nocommit")
)

// NamespaceMatch Checks if the given namespace query matches the given namespace
func NamespaceMatch(query string, namespace Namespace) bool {
	match, err := regexp.Match(string(namespace), []byte(query))
	if err != nil {
		logging.Error("Could not match regex for %v, query: %s, error: %v", namespace, query, err)
	}
	return match
}
