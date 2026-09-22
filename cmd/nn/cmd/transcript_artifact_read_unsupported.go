//go:build !darwin && !linux

package cmd

func openTranscriptArtifact(string, string) (artifactReadFile, error) {
	return nil, errArtifactUnsupported
}
