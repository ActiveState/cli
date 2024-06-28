package buildscript

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/ascript"
)

// BuildScript is what we want consuming code to work with. This specifically makes the raw
// presentation private as no consuming code should ever be looking at the raw representation.
// Instead this package should facilitate the use-case of the consuming code through convenience
// methods that are easy to understand and work with.
type BuildScript struct {
	as *ascript.AScript
}

func New() (*BuildScript, error) {
	return UnmarshalBuildExpression([]byte(emptyBuildExpression), nil)
}

func Unmarshal(data []byte) (*BuildScript, error) {
	as, err := ascript.Unmarshal(data)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to marshal AScript")
	}
	return &BuildScript{as}, nil
}

func (b *BuildScript) Marshal() ([]byte, error) {
	return b.as.Marshal()
}

func (b *BuildScript) AtTime() *time.Time {
	return b.as.AtTime
}

func (b *BuildScript) SetAtTime(t time.Time) {
	b.as.AtTime = &t
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
