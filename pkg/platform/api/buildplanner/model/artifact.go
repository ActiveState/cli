package model

import "github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"

func IsStateToolArtifact(mimeType string) bool {
	return mimeType == types.XArtifactMimeType ||
		mimeType == types.XActiveStateArtifactMimeType ||
		mimeType == types.XCamelInstallerMimeType
}

func IsSuccessArtifactStatus(status string) bool {
	return status == types.ArtifactSucceeded || status == types.ArtifactBlocked ||
		status == types.ArtifactStarted || status == types.ArtifactReady
}
