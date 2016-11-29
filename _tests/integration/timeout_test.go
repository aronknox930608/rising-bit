package integration

import (
	"fmt"
	"testing"
	"time"

	"path/filepath"

	"os"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func Test_TimeoutTest(t *testing.T) {
	configPth := "timeout_test_bitrise.yml"

	t.Log("Timeout test")
	{
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__timeout_test__")
		require.NoError(t, err)
		defer func() {
			require.NoError(t, os.RemoveAll(tmpDir))
		}()

		testFilePth1 := filepath.Join(tmpDir, "file1")
		testFilePth2 := filepath.Join(tmpDir, "file2")

		envs := []string{
			fmt.Sprintf("TIMEOUT_TEST_FILE_PTH_1=%s", testFilePth1),
			fmt.Sprintf("TIMEOUT_TEST_FILE_PTH_2=%s", testFilePth2),
		}
		cmd := cmdex.NewCommand(binPath(), "run", "timeout", "--config", configPth)
		cmd.AppendEnvs(envs)

		start := time.Now()
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		elapsed := time.Since(start)

		require.EqualError(t, err, "exit status 1", out)
		require.Equal(t, true, elapsed < 12*time.Second)

		t.Log("Should exist")
		{
			exist, err := pathutil.IsPathExists(testFilePth1)
			require.NoError(t, err)
			require.Equal(t, true, exist)
		}

		t.Log("Should NOT exist")
		{
			exist, err := pathutil.IsPathExists(testFilePth2)
			require.NoError(t, err)
			require.Equal(t, false, exist)
		}
	}
}
