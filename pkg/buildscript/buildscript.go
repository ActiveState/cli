package buildscript

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
)

// BuildScript is what we want consuming code to work with. This specifically makes the raw
// presentation private as no consuming code should ever be looking at the raw representation.
// Instead this package should facilitate the use-case of the consuming code through convenience
// methods that are easy to understand and work with.
type BuildScript struct {
	raw     *rawBuildScript
	project string
	atTime  *time.Time
}

func New() (*BuildScript, error) {
	return UnmarshalBuildExpression([]byte(emptyBuildExpression), "", nil)
}

func (b *BuildScript) Project() string {
	return b.project
}

func (b *BuildScript) SetProject(url string) {
	b.project = url
}

func (b *BuildScript) AtTime() *time.Time {
	return b.atTime
}

func (b *BuildScript) SetAtTime(t time.Time) {
	b.atTime = &t
}

func (b *BuildScript) Equals(other *BuildScript) (bool, error) {
	myBytes, err := b.Marshal()
	if err != nil {
		return false, errs.New("Unable to marshal this buildscript: %s", errs.JoinMessage(err))
	}
	otherBytes, err := other.Marshal()
	if err != nil {
		return false, errs.New("Unable to marshal other buildscript: %s", errs.JoinMessage(err))
	}
	return string(myBytes) == string(otherBytes), nil
}

func (b *BuildScript) Clone() (*BuildScript, error) {
	m, err := b.Marshal()
	if err != nil {
		return nil, errs.Wrap(err, "unable to marshal this buildscript")
	}

	u, err := Unmarshal(m)
	if err != nil {
		return nil, errs.Wrap(err, "unable to unmarshal buildscript")
	}
	return u, nil
}
