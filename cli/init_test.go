package cli

import (
	"testing"

	"github.com/bitrise-io/bitrise/bitrise"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	// should be valid
	bitriseConfContent, err := generateBitriseYMLContent("App Title", "master")
	require.NoError(t, err)
	require.NotEqual(t, "", bitriseConfContent)
	require.Contains(t, bitriseConfContent, `- BITRISE_PROJECT_TITLE: "App Title"`)
	require.Contains(t, bitriseConfContent, `- BITRISE_DEV_BRANCH: "master"`)

	bitriseConfModel, err := bitrise.ConfigModelFromYAMLBytes([]byte(bitriseConfContent))
	require.NoError(t, err)

	err = bitriseConfModel.Validate()
	require.NoError(t, err)
}
