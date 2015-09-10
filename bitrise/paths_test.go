package bitrise

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitPaths(t *testing.T) {
	//
	// BITRISE_SOURCE_DIR

	// Unset BITRISE_SOURCE_DIR -> after InitPaths BITRISE_SOURCE_DIR should be CurrentDir
	if os.Getenv(BitriseSourceDirEnvKey) != "" {
		require.Equal(t, nil, os.Unsetenv(BitriseSourceDirEnvKey))
	}
	require.Equal(t, nil, InitPaths())
	require.Equal(t, CurrentDir, os.Getenv(BitriseSourceDirEnvKey))

	// Set BITRISE_SOURCE_DIR -> after InitPaths BITRISE_SOURCE_DIR should keep content
	require.Equal(t, nil, os.Setenv(BitriseSourceDirEnvKey, "$HOME/test"))
	require.Equal(t, nil, InitPaths())
	require.Equal(t, "$HOME/test", os.Getenv(BitriseSourceDirEnvKey))

	//
	// BITRISE_DEPLOY_DIR

	// Unset BITRISE_DEPLOY_DIR -> after InitPaths BITRISE_DEPLOY_DIR should be ./build
	deployDir, err := filepath.Abs(path.Join(CurrentDir, "build"))
	require.Equal(t, nil, err)

	if os.Getenv(BitriseDeployDirEnvKey) != "" {
		require.Equal(t, nil, os.Unsetenv(BitriseDeployDirEnvKey))
	}
	require.Equal(t, nil, InitPaths())
	require.Equal(t, deployDir, os.Getenv(BitriseDeployDirEnvKey))

	// Set BITRISE_DEPLOY_DIR -> after InitPaths BITRISE_DEPLOY_DIR should keep content
	require.Equal(t, nil, os.Setenv(BitriseDeployDirEnvKey, "$HOME/test"))
	require.Equal(t, nil, InitPaths())
	require.Equal(t, "$HOME/test", os.Getenv(BitriseDeployDirEnvKey))
}
