package runtime

import (
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/go-openapi/strfmt"
)

func WithEventHandlers(handlers ...events.HandlerFunc) SetOpt {
	return func(opts *Opts) { opts.EventHandlers = handlers }
}

// WithAuthToken forwards the platform JWT to the build-log-streamer WebSocket
// so the server can authorize the stream. Empty token = anonymous.
func WithAuthToken(token string) SetOpt {
	return func(opts *Opts) { opts.AuthToken = token }
}

// WithDecryptionKey supplies a function that lazily fetches the organization
// AES-256 key used to decrypt private artifacts during install.
func WithDecryptionKey(fetch func() ([]byte, error)) SetOpt {
	return func(opts *Opts) {
		opts.OrgKey = fetch
	}
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

func WithPortable() SetOpt {
	return func(opts *Opts) { opts.Portable = true }
}

func WithCacheSize(mb int) SetOpt {
	return func(opts *Opts) { opts.CacheSize = mb }
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
