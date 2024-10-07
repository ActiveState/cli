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
	raw *rawBuildScript
}

func init() {
	// Guard against emptyBuildExpression having parsing issues
	if !condition.BuiltViaCI() || condition.InActiveStateCI() {
		err := New().UnmarshalBuildExpression([]byte(emptyBuildExpression))
		if err != nil {
			panic(err)
		}
	}
}

func Create() *BuildScript {
	bs := New()
	// We don't handle unmarshalling errors here, see the init function for that.
	// Since the empty build expression is a constant there's really no need to error check this each time.
	_ = bs.UnmarshalBuildExpression([]byte(emptyBuildExpression))
	return bs
}

func New() *BuildScript {
	bs := Create()
	return bs
}

func (b *BuildScript) AtTime() *time.Time {
	return b.raw.AtTime
}

func (b *BuildScript) SetAtTime(t time.Time) {
	b.raw.AtTime = &t
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
	bb, err := deep.Copy(b)
	if err != nil {
		return nil, errs.Wrap(err, "unable to clone buildscript")
	}
	return bb, nil
}
