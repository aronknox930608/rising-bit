package integration

import (
	"testing"

	"github.com/bitrise-io/go-utils/command"
	"github.com/stretchr/testify/require"
)

func Test_TriggerCheck(t *testing.T) {
	configPth := "trigger_check_test_bitrise.yml"
	secretsPth := "trigger_check_test_secrets.yml"

	t.Log("PR mode : from secrets - is_pull_request_allowed : true")
	{
		cmd := command.New(binPath(), "trigger-check", "pr_allowed", "--config", configPth, "--inventory", secretsPth, "--format", "json")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)
		require.Equal(t, `{"pattern":"pr_allowed","workflows":["pr_allowed"]}`, out, out)
	}

	t.Log("Not PR mode - is_pull_request_allowed : true")
	{
		cmd := command.New(binPath(), "trigger-check", "pr_allowed", "--config", configPth, "--format", "json")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)
		require.Equal(t, `{"pattern":"pr_allowed","workflows":["pr_allowed"]}`, out)
	}

	t.Log("Not PR mode - is_pull_request_allowed : false")
	{
		cmd := command.New(binPath(), "trigger-check", "only_code_push", "--config", configPth, "--format", "json")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)
		require.Equal(t, `{"pattern":"only_code_push","workflows":["only_code_push"]}`, out, out)
	}

	t.Log("PR mode : from secrets - is_pull_request_allowed : false")
	{
		cmd := command.New(binPath(), "trigger-check", "only_code_push", "--config", configPth, "--inventory", secretsPth, "--format", "json")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)
		require.Equal(t, `{"pattern":"only_code_push","workflows":["fall_back"]}`, out)
	}

	t.Log("Not PR mode - is_pull_request_allowed : false")
	{
		cmd := command.New(binPath(), "trigger-check", "fall_back", "--config", configPth, "--format", "json")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.NoError(t, err, out)
		require.Equal(t, `{"pattern":"fall_back","workflows":["fall_back"]}`, out)
	}
}

func Test_InvalidTriggerCheck(t *testing.T) {
	configPth := "trigger_check_test_empty_bitrise.yml"
	secretsPth := "trigger_check_test_secrets.yml"

	t.Log("Empty trigger pattern - PR mode : from secrets - is_pull_request_allowed : true")
	{
		cmd := command.New(binPath(), "trigger-check", "", "--config", configPth, "--inventory", secretsPth, "--format", "json")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.Error(t, err, out)
	}

	t.Log("Empty triggered workflow id - PR mode : from secrets - is_pull_request_allowed : true")
	{
		cmd := command.New(binPath(), "trigger-check", "trigger_empty_workflow_id", "--config", configPth, "--inventory", secretsPth, "--format", "json")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		require.Error(t, err, out)
	}
}
