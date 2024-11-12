package runtime

import (
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/go-openapi/strfmt"
)

func WithEventHandlers(handlers ...events.HandlerFunc) SetOpt {
	return func(opts *Opts) { opts.EventHandlers = handlers }
}

func WithBuildlogFilePath(path string) SetOpt {
	return func(opts *Opts) { opts.BuildlogFilePath = path }
}

func WithBuildProgressUrl(url string) SetOpt {
	return func(opts *Opts) { opts.BuildProgressUrl = url }
}

func WithPreferredLibcVersion(version string) SetOpt {
	return func(opts *Opts) { opts.PreferredLibcVersion = version }
}

func WithArchive(dir string, platformID strfmt.UUID, ext string) SetOpt {
	return func(opts *Opts) {
		opts.FromArchive = &fromArchive{dir, platformID, ext}
	}
}

func WithAnnotations(owner, project string, commitUUID strfmt.UUID) SetOpt {
	return func(opts *Opts) {
		opts.Annotations.Owner = owner
		opts.Annotations.Project = project
		opts.Annotations.CommitUUID = commitUUID
	}
}
