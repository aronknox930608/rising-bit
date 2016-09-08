package toolkits

import (
	"fmt"
	"path/filepath"

	"github.com/bitrise-io/bitrise/utils"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/stringutil"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

// BashToolkit ...
type BashToolkit struct {
}

// Check ...
func (toolkit BashToolkit) Check() (bool, ToolkitCheckResult, error) {
	binPath, err := utils.CheckProgramInstalledPath("bash")
	if err != nil {
		return false, ToolkitCheckResult{}, fmt.Errorf("Failed to get bash binary path, error: %s", err)
	}

	verOut, err := cmdex.RunCommandAndReturnStdout("bash", "--version")
	if err != nil {
		return false, ToolkitCheckResult{}, fmt.Errorf("Failed to check bash version, error: %s", err)
	}

	verStr := stringutil.ReadFirstLine(verOut, true)

	return false, ToolkitCheckResult{
		Path:    binPath,
		Version: verStr,
	}, nil
}

// Bootstrap ...
func (toolkit BashToolkit) Bootstrap() error {
	return nil
}

// Install ...
func (toolkit BashToolkit) Install() error {
	return nil
}

// ToolkitName ...
func (toolkit BashToolkit) ToolkitName() string {
	return "bash"
}

// PrepareForStepRun ...
func (toolkit BashToolkit) PrepareForStepRun(step stepmanModels.StepModel, stepIDorURI, stepVersion, stepAbsDirPath string) error {
	return nil
}

// StepRunCommandArguments ...
func (toolkit BashToolkit) StepRunCommandArguments(stepDirPath, stepIDorURI, stepVersion string) ([]string, error) {
	stepFilePath := filepath.Join(stepDirPath, "step.sh")
	cmd := []string{"bash", stepFilePath}
	return cmd, nil
}
