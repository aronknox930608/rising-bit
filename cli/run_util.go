package cli

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/bitrise/bitrise"
	"github.com/bitrise-io/bitrise/configs"
	"github.com/bitrise-io/bitrise/models"
	"github.com/bitrise-io/bitrise/plugins"
	"github.com/bitrise-io/bitrise/tools"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/versions"
	stepmanModels "github.com/bitrise-io/stepman/models"
)

func isPRMode(prGlobalFlag bool, inventoryEnvironments []envmanModels.EnvironmentItemModel) (bool, error) {
	prIDEnv := os.Getenv(configs.PullRequestIDEnvKey)
	prModeEnv := os.Getenv(configs.PRModeEnvKey)

	if prGlobalFlag || prIDEnv != "" || prModeEnv == "true" {
		return true, nil
	}

	for _, env := range inventoryEnvironments {
		key, value, err := env.GetKeyValuePair()
		if err != nil {
			return false, err
		}

		if key == configs.PullRequestIDEnvKey && value != "" {
			return true, nil
		}
		if key == configs.PRModeEnvKey && value == "true" {
			return true, nil
		}
	}

	return false, nil
}

func registerPrMode(isPRMode bool) error {
	configs.IsPullRequestMode = isPRMode

	if isPRMode {
		log.Info(colorstring.Yellow("bitrise runs in PR mode"))
		return os.Setenv(configs.PRModeEnvKey, "true")
	}
	return os.Setenv(configs.PRModeEnvKey, "false")
}

func isCIMode(ciGlobalFlag bool, inventoryEnvironments []envmanModels.EnvironmentItemModel) (bool, error) {
	ciModeEnv := os.Getenv(configs.CIModeEnvKey)

	if ciGlobalFlag || ciModeEnv == "true" {
		return true, nil
	}

	for _, env := range inventoryEnvironments {
		key, value, err := env.GetKeyValuePair()
		if err != nil {
			return false, err
		}

		if key == configs.CIModeEnvKey && value == "true" {
			return true, nil
		}
	}

	return false, nil
}

func registerCIMode(isCIMode bool) error {
	configs.IsCIMode = isCIMode

	if isCIMode {
		log.Info(colorstring.Yellow("bitrise runs in CI mode"))
		return os.Setenv(configs.CIModeEnvKey, "true")
	}
	return os.Setenv(configs.CIModeEnvKey, "false")
}

// GetBitriseConfigFromBase64Data ...
func GetBitriseConfigFromBase64Data(configBase64Str string) (models.BitriseDataModel, []string, error) {
	configBase64Bytes, err := base64.StdEncoding.DecodeString(configBase64Str)
	if err != nil {
		return models.BitriseDataModel{}, []string{}, fmt.Errorf("Failed to decode base 64 string, error: %s", err)
	}

	config, warnings, err := bitrise.ConfigModelFromYAMLBytes(configBase64Bytes)
	if err != nil {
		return models.BitriseDataModel{}, warnings, fmt.Errorf("Failed to parse bitrise config, error: %s", err)
	}

	return config, warnings, nil
}

// GetBitriseConfigFilePath ...
func GetBitriseConfigFilePath(bitriseConfigPath string) (string, error) {
	if bitriseConfigPath == "" {
		log.Debugln("[BITRISE_CLI] - Workflow path not defined, searching for " + DefaultBitriseConfigFileName + " in current folder...")

		bitriseConfigPath = path.Join(configs.CurrentDir, DefaultBitriseConfigFileName)

		if exist, err := pathutil.IsPathExists(bitriseConfigPath); err != nil {
			return "", err
		} else if !exist {
			return "", errors.New("No workflow yml found")
		}
	}

	return bitriseConfigPath, nil
}

// CreateBitriseConfigFromCLIParams ...
func CreateBitriseConfigFromCLIParams(bitriseConfigBase64Data, bitriseConfigPath string) (models.BitriseDataModel, []string, error) {
	bitriseConfig := models.BitriseDataModel{}
	warnings := []string{}

	if bitriseConfigBase64Data != "" {
		config, warns, err := GetBitriseConfigFromBase64Data(bitriseConfigBase64Data)
		warnings = warns
		if err != nil {
			return models.BitriseDataModel{}, warnings, fmt.Errorf("Failed to get config (bitrise.yml) from base 64 data, err: %s", err)
		}
		bitriseConfig = config
	} else {
		bitriseConfigPath, err := GetBitriseConfigFilePath(bitriseConfigPath)
		if err != nil {
			return models.BitriseDataModel{}, []string{}, fmt.Errorf("Failed to get config (bitrise.yml) path: %s", err)
		}
		if bitriseConfigPath == "" {
			return models.BitriseDataModel{}, []string{}, errors.New("Failed to get config (bitrise.yml) path: empty bitriseConfigPath")
		}

		config, warns, err := bitrise.ReadBitriseConfig(bitriseConfigPath)
		warnings = warns
		if err != nil {
			return models.BitriseDataModel{}, warnings, fmt.Errorf("Config (path:%s) in not valid: %s", bitriseConfigPath, err)
		}
		bitriseConfig = config
	}

	isConfigVersionOK, err := versions.IsVersionGreaterOrEqual(models.Version, bitriseConfig.FormatVersion)
	if err != nil {
		log.Warn("bitrise CLI model version: ", models.Version)
		log.Warn("bitrise.yml Format Version: ", bitriseConfig.FormatVersion)
		return models.BitriseDataModel{}, warnings, fmt.Errorf("Failed to compare bitrise CLI models's version with the bitrise.yml FormatVersion: %s", err)
	}
	if !isConfigVersionOK {
		log.Warnf("The bitrise.yml has a higher Format Version (%s) than the bitrise CLI model's version (%s).", bitriseConfig.FormatVersion, models.Version)
		return models.BitriseDataModel{}, warnings, errors.New("This bitrise.yml was created with and for a newer version of bitrise CLI, please upgrade your bitrise CLI to use this bitrise.yml!")
	}

	return bitriseConfig, warnings, nil
}

// GetInventoryFromBase64Data ...
func GetInventoryFromBase64Data(inventoryBase64Str string) ([]envmanModels.EnvironmentItemModel, error) {
	inventoryBase64Bytes, err := base64.StdEncoding.DecodeString(inventoryBase64Str)
	if err != nil {
		return []envmanModels.EnvironmentItemModel{}, fmt.Errorf("Failed to decode base 64 string, error: %s", err)
	}

	inventory, err := bitrise.InventoryModelFromYAMLBytes(inventoryBase64Bytes)
	if err != nil {
		return []envmanModels.EnvironmentItemModel{}, err
	}

	return inventory.Envs, nil
}

// GetInventoryFilePath ...
func GetInventoryFilePath(inventoryPath string) (string, error) {
	if inventoryPath == "" {
		log.Debugln("[BITRISE_CLI] - Inventory path not defined, searching for " + DefaultSecretsFileName + " in current folder...")
		inventoryPath = path.Join(configs.CurrentDir, DefaultSecretsFileName)

		if exist, err := pathutil.IsPathExists(inventoryPath); err != nil {
			return "", err
		} else if !exist {
			inventoryPath = ""
		}
	}

	return inventoryPath, nil
}

// CreateInventoryFromCLIParams ...
func CreateInventoryFromCLIParams(inventoryBase64Data, inventoryPath string) ([]envmanModels.EnvironmentItemModel, error) {
	inventoryEnvironments := []envmanModels.EnvironmentItemModel{}

	if inventoryBase64Data != "" {
		inventory, err := GetInventoryFromBase64Data(inventoryBase64Data)
		if err != nil {
			return []envmanModels.EnvironmentItemModel{}, fmt.Errorf("Failed to get inventory from base 64 data, err: %s", err)
		}
		inventoryEnvironments = inventory
	} else {
		inventoryPath, err := GetInventoryFilePath(inventoryPath)
		if err != nil {
			return []envmanModels.EnvironmentItemModel{}, fmt.Errorf("Failed to get inventory path: %s", err)
		}

		if inventoryPath != "" {
			var err error
			inventory, err := bitrise.CollectEnvironmentsFromFile(inventoryPath)
			if err != nil {
				return []envmanModels.EnvironmentItemModel{}, fmt.Errorf("Invalid invetory format: %s", err)
			}
			inventoryEnvironments = inventory
		}
	}

	return inventoryEnvironments, nil
}

func getCurrentBitriseSourceDir(envlist []envmanModels.EnvironmentItemModel) (string, error) {
	bitriseSourceDir := os.Getenv(configs.BitriseSourceDirEnvKey)
	for i := len(envlist) - 1; i >= 0; i-- {
		env := envlist[i]

		key, value, err := env.GetKeyValuePair()
		if err != nil {
			return bitriseSourceDir, err
		}

		if key == configs.BitriseSourceDirEnvKey && value != "" {
			return value, nil
		}
	}
	return bitriseSourceDir, nil
}

func runStep(step stepmanModels.StepModel, stepIDData models.StepIDData, stepDir string, environments []envmanModels.EnvironmentItemModel, buildRunResults models.BuildRunResultsModel) (int, []envmanModels.EnvironmentItemModel, error) {
	log.Debugf("[BITRISE_CLI] - Try running step: %s (%s)", stepIDData.IDorURI, stepIDData.Version)

	// Check & Install Step Dependencies
	if len(step.Dependencies) > 0 {
		log.Warnf("step.dependencies is deprecated... Use step.deps instead.")
	}

	if len(step.Deps.Brew) > 0 || len(step.Deps.AptGet) > 0 || len(step.Deps.CheckOnly) > 0 {
		//
		// New dependency handling
		for _, checkOnlyDep := range step.Deps.CheckOnly {
			if err := bitrise.DependencyTryCheckTool(checkOnlyDep.Name); err != nil {
				return 1, []envmanModels.EnvironmentItemModel{}, err
			}
			log.Infof(" * "+colorstring.Green("[OK]")+" Step dependency (%s) installed, available.", checkOnlyDep.Name)
		}

		switch runtime.GOOS {
		case "darwin":
			for _, brewDep := range step.Deps.Brew {
				if err := bitrise.InstallWithBrewIfNeeded(brewDep.Name, configs.IsCIMode); err != nil {
					log.Infof("Failed to install (%s) with brew", brewDep.Name)
					return 1, []envmanModels.EnvironmentItemModel{}, err
				}
				log.Infof(" * "+colorstring.Green("[OK]")+" Step dependency (%s) installed, available.", brewDep.Name)
			}
		case "linux":
			for _, aptGetDep := range step.Deps.AptGet {
				log.Infof("Start installing (%s) with apt-get", aptGetDep.Name)
				if err := bitrise.InstallWithAptGetIfNeeded(aptGetDep.Name, configs.IsCIMode); err != nil {
					log.Infof("Failed to install (%s) with apt-get", aptGetDep.Name)
					return 1, []envmanModels.EnvironmentItemModel{}, err
				}
				log.Infof(" * "+colorstring.Green("[OK]")+" Step dependency (%s) installed, available.", aptGetDep.Name)
			}
		default:
			return 1, []envmanModels.EnvironmentItemModel{}, errors.New("Unsupported os")
		}
	} else if len(step.Dependencies) > 0 {
		log.Info("Deprecated dependencies found")
		//
		// Deprecated dependency handling
		for _, dep := range step.Dependencies {
			isSkippedBecauseOfPlatform := false
			switch dep.Manager {
			case depManagerBrew:
				if runtime.GOOS == "darwin" {
					err := bitrise.InstallWithBrewIfNeeded(dep.Name, configs.IsCIMode)
					if err != nil {
						return 1, []envmanModels.EnvironmentItemModel{}, err
					}
				} else {
					isSkippedBecauseOfPlatform = true
				}
				break
			case depManagerTryCheck:
				err := bitrise.DependencyTryCheckTool(dep.Name)
				if err != nil {
					return 1, []envmanModels.EnvironmentItemModel{}, err
				}
				break
			default:
				return 1, []envmanModels.EnvironmentItemModel{}, errors.New("Not supported dependency (" + dep.Manager + ") (" + dep.Name + ")")
			}

			if isSkippedBecauseOfPlatform {
				log.Debugf(" * Dependency (%s) skipped, manager (%s) not supported on this platform (%s)", dep.Name, dep.Manager, runtime.GOOS)
			} else {
				log.Infof(" * "+colorstring.Green("[OK]")+" Step dependency (%s) installed, available.", dep.Name)
			}
		}
	}

	// Collect step inputs
	if err := tools.EnvmanInitAtPath(configs.InputEnvstorePath); err != nil {
		return 1, []envmanModels.EnvironmentItemModel{}, err
	}

	if err := bitrise.ExportEnvironmentsList(environments); err != nil {
		return 1, []envmanModels.EnvironmentItemModel{}, err
	}

	evaluatedInputs := []envmanModels.EnvironmentItemModel{}
	for _, input := range step.Inputs {
		key, value, err := input.GetKeyValuePair()
		if err != nil {
			return 1, []envmanModels.EnvironmentItemModel{}, err
		}

		options, err := input.GetOptions()
		if err != nil {
			return 1, []envmanModels.EnvironmentItemModel{}, err
		}

		if options.IsTemplate != nil && *options.IsTemplate {
			outStr, err := tools.EnvmanJSONPrint(configs.InputEnvstorePath)
			if err != nil {
				return 1, []envmanModels.EnvironmentItemModel{}, fmt.Errorf("EnvmanJSONPrint failed, err: %s", err)
			}

			envList, err := envmanModels.NewEnvJSONList(outStr)
			if err != nil {
				return 1, []envmanModels.EnvironmentItemModel{}, fmt.Errorf("CreateFromJSON failed, err: %s", err)
			}

			evaluatedValue, err := bitrise.EvaluateTemplateToString(value, configs.IsCIMode, configs.IsPullRequestMode, buildRunResults, envList)
			if err != nil {
				return 1, []envmanModels.EnvironmentItemModel{}, err
			}

			input[key] = evaluatedValue
		}

		evaluatedInputs = append(evaluatedInputs, input)
	}
	environments = append(environments, evaluatedInputs...)

	if err := tools.EnvmanInitAtPath(configs.InputEnvstorePath); err != nil {
		return 1, []envmanModels.EnvironmentItemModel{}, err
	}

	if err := bitrise.ExportEnvironmentsList(environments); err != nil {
		return 1, []envmanModels.EnvironmentItemModel{}, err
	}

	// Run step
	stepCmd := path.Join(stepDir, "step.sh")
	cmd := []string{"bash", stepCmd}
	bitriseSourceDir, err := getCurrentBitriseSourceDir(environments)
	if err != nil {
		return 1, []envmanModels.EnvironmentItemModel{}, err
	}
	if bitriseSourceDir == "" {
		bitriseSourceDir = configs.CurrentDir
	}

	if exit, err := tools.EnvmanRun(configs.InputEnvstorePath, bitriseSourceDir, cmd); err != nil {
		stepOutputs, envErr := bitrise.CollectEnvironmentsFromFile(configs.OutputEnvstorePath)
		if envErr != nil {
			return 1, []envmanModels.EnvironmentItemModel{}, envErr
		}

		return exit, stepOutputs, err
	}

	stepOutputs, err := bitrise.CollectEnvironmentsFromFile(configs.OutputEnvstorePath)
	if err != nil {
		return 1, []envmanModels.EnvironmentItemModel{}, err
	}
	log.Debugf("[BITRISE_CLI] - Step executed: %s (%s)", stepIDData.IDorURI, stepIDData.Version)

	return 0, stepOutputs, nil
}

func activateAndRunSteps(workflow models.WorkflowModel, defaultStepLibSource string, buildRunResults models.BuildRunResultsModel, environments *[]envmanModels.EnvironmentItemModel, isLastWorkflow bool) models.BuildRunResultsModel {
	log.Debugln("[BITRISE_CLI] - Activating and running steps")

	// ------------------------------------------
	// In function global variables - These are global for easy use in local register step run result methods.
	var stepStartTime time.Time

	// Holds pointer to current step info, for easy usage in local register step run result methods.
	// The value is filled with the current running step info.
	var stepInfoPtr stepmanModels.StepInfoModel
	var stepIdxPtr int

	// ------------------------------------------
	// In function method - Registration methods, for register step run results.
	registerStepRunResults := func(runIf string, resultCode, exitCode int, err error, isLastStep, printStepHeader bool) {
		if printStepHeader {
			bitrise.PrintRunningStepHeader(stepInfoPtr, stepIdxPtr)
		}

		stepInfoCopy := stepmanModels.StepInfoModel{
			ID:            stepInfoPtr.ID,
			Title:         stepInfoPtr.Title,
			Version:       stepInfoPtr.Version,
			Latest:        stepInfoPtr.Latest,
			SupportURL:    stepInfoPtr.SupportURL,
			SourceCodeURL: stepInfoPtr.SourceCodeURL,
			GlobalInfo:    stepInfoPtr.GlobalInfo,
		}

		stepResults := models.StepRunResultsModel{
			StepInfo: stepInfoCopy,
			Status:   resultCode,
			Idx:      buildRunResults.ResultsCount(),
			RunTime:  time.Now().Sub(stepStartTime),
			Error:    err,
			ExitCode: exitCode,
		}

		isExitStatusError := true
		if err != nil {
			isExitStatusError = errorutil.IsExitStatusError(err)
		}

		switch resultCode {
		case models.StepRunStatusCodeSuccess:
			buildRunResults.SuccessSteps = append(buildRunResults.SuccessSteps, stepResults)
			break
		case models.StepRunStatusCodeFailed:
			if !isExitStatusError {
				log.Errorf("Step (%s) failed, error: %s", stepInfoCopy.Title, err)
			}

			buildRunResults.FailedSteps = append(buildRunResults.FailedSteps, stepResults)
			break
		case models.StepRunStatusCodeFailedSkippable:
			if !isExitStatusError {
				log.Warnf("Step (%s) failed, but was marked as skippable, error: %s", stepInfoCopy.Title, err)
			} else {
				log.Warnf("Step (%s) failed, but was marked as skippable", stepInfoCopy.Title)
			}

			buildRunResults.FailedSkippableSteps = append(buildRunResults.FailedSkippableSteps, stepResults)
			break
		case models.StepRunStatusCodeSkipped:
			log.Warnf("A previous step failed, and this step (%s) was not marked as IsAlwaysRun, skipped", stepInfoCopy.Title)

			buildRunResults.SkippedSteps = append(buildRunResults.SkippedSteps, stepResults)
			break
		case models.StepRunStatusCodeSkippedWithRunIf:
			log.Warn("The step's (" + stepInfoCopy.Title + ") Run-If expression evaluated to false - skipping")
			if runIf != "" {
				log.Info("The Run-If expression was: ", colorstring.Blue(runIf))
			}

			buildRunResults.SkippedSteps = append(buildRunResults.SkippedSteps, stepResults)
			break
		default:
			log.Error("Unkown result code")
			return
		}

		bitrise.PrintRunningStepFooter(stepResults, isLastStep)
	}

	// ------------------------------------------
	// Main - Preparing & running the steps
	for idx, stepListItm := range workflow.Steps {
		// Per step variables
		stepStartTime = time.Now()
		isLastStep := isLastWorkflow && (idx == len(workflow.Steps)-1)
		stepInfoPtr = stepmanModels.StepInfoModel{}
		stepIdxPtr = idx

		// Per step cleanup
		if err := bitrise.SetBuildFailedEnv(buildRunResults.IsBuildFailed()); err != nil {
			log.Error("Failed to set Build Status envs")
		}

		if err := bitrise.CleanupStepWorkDir(); err != nil {
			registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
			continue
		}

		//
		// Preparing the step
		if err := tools.EnvmanInitAtPath(configs.InputEnvstorePath); err != nil {
			registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
			continue
		}

		if err := bitrise.ExportEnvironmentsList(*environments); err != nil {
			registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
			continue
		}

		// Get step id & version data
		compositeStepIDStr, workflowStep, err := models.GetStepIDStepDataPair(stepListItm)
		if err != nil {
			registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
			continue
		}
		stepInfoPtr.ID = compositeStepIDStr
		if workflowStep.Title != nil && *workflowStep.Title != "" {
			stepInfoPtr.Title = *workflowStep.Title
		} else {
			stepInfoPtr.Title = compositeStepIDStr
		}

		stepIDData, err := models.CreateStepIDDataFromString(compositeStepIDStr, defaultStepLibSource)
		if err != nil {
			registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
			continue
		}
		stepInfoPtr.ID = stepIDData.IDorURI
		if stepInfoPtr.Title == "" {
			stepInfoPtr.Title = stepIDData.IDorURI
		}
		stepInfoPtr.Version = stepIDData.Version
		stepInfoPtr.StepLib = stepIDData.SteplibSource

		//
		// Activating the step
		stepDir := configs.BitriseWorkStepsDirPath
		stepYMLPth := path.Join(configs.BitriseWorkDirPath, "current_step.yml")

		if stepIDData.SteplibSource == "path" {
			log.Debugf("[BITRISE_CLI] - Local step found: (path:%s)", stepIDData.IDorURI)
			stepAbsLocalPth, err := pathutil.AbsPath(stepIDData.IDorURI)
			if err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}

			log.Debugln("stepAbsLocalPth:", stepAbsLocalPth, "|stepDir:", stepDir)

			if err := cmdex.CopyDir(stepAbsLocalPth, stepDir, true); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}

			if err := cmdex.CopyFile(path.Join(stepAbsLocalPth, "step.yml"), stepYMLPth); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}
		} else if stepIDData.SteplibSource == "git" {
			log.Debugf("[BITRISE_CLI] - Remote step, with direct git uri: (uri:%s) (tag-or-branch:%s)", stepIDData.IDorURI, stepIDData.Version)
			if err := cmdex.GitCloneTagOrBranch(stepIDData.IDorURI, stepDir, stepIDData.Version); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}

			if err := cmdex.CopyFile(path.Join(stepDir, "step.yml"), stepYMLPth); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}
		} else if stepIDData.SteplibSource == "_" {
			log.Debugf("[BITRISE_CLI] - Steplib independent step, with direct git uri: (uri:%s) (tag-or-branch:%s)", stepIDData.IDorURI, stepIDData.Version)

			// Steplib independent steps are completly defined in workflow
			stepYMLPth = ""
			if err := workflowStep.FillMissingDefaults(); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}

			if err := cmdex.GitCloneTagOrBranch(stepIDData.IDorURI, stepDir, stepIDData.Version); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}
		} else if stepIDData.SteplibSource != "" {
			log.Debugf("[BITRISE_CLI] - Steplib (%s) step (id:%s) (version:%s) found, activating step", stepIDData.SteplibSource, stepIDData.IDorURI, stepIDData.Version)
			if err := tools.StepmanSetup(stepIDData.SteplibSource); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}

			isLatestVersionOfStep := (stepIDData.Version == "")
			if isLatestVersionOfStep && !buildRunResults.IsStepLibUpdated(stepIDData.SteplibSource) {
				log.Infof("Step uses latest version -- Updating StepLib ...")
				if err := tools.StepmanUpdate(stepIDData.SteplibSource); err != nil {
					log.Warnf("Step uses latest version, but failed to update StepLib, err: %s", err)
				} else {
					buildRunResults.StepmanUpdates[stepIDData.SteplibSource]++
				}
			}

			outStr, err := tools.StepmanJSONStepLibStepInfo(stepIDData.SteplibSource, stepIDData.IDorURI, stepIDData.Version)
			if err != nil {
				if buildRunResults.IsStepLibUpdated(stepIDData.SteplibSource) {
					registerStepRunResults("", models.StepRunStatusCodeFailed, 1, fmt.Errorf("StepmanJSONStepLibStepInfo failed, err: %s", err), isLastStep, true)
					continue
				}
				// May StepLib should be updated
				log.Infof("Step info not found in StepLib (%s) -- Updating ...", stepIDData.SteplibSource)
				if err := tools.StepmanUpdate(stepIDData.SteplibSource); err != nil {
					registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
					continue
				}
				buildRunResults.StepmanUpdates[stepIDData.SteplibSource]++

				outStr, err = tools.StepmanJSONStepLibStepInfo(stepIDData.SteplibSource, stepIDData.IDorURI, stepIDData.Version)
				if err != nil {
					registerStepRunResults("", models.StepRunStatusCodeFailed, 1, fmt.Errorf("StepmanJSONStepLibStepInfo failed, err: %s", err), isLastStep, true)
					continue
				}
			}

			stepInfo, err := stepmanModels.StepInfoModel{}.CreateFromJSON(outStr)
			if err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, fmt.Errorf("CreateFromJSON failed, err: %s", err), isLastStep, true)
				continue
			}

			stepInfoPtr.ID = stepInfo.ID
			if stepInfoPtr.Title == "" {
				stepInfoPtr.Title = stepInfo.ID
			}
			stepInfoPtr.Version = stepInfo.Version
			stepInfoPtr.Latest = stepInfo.Latest
			stepInfoPtr.GlobalInfo = stepInfo.GlobalInfo

			if err := tools.StepmanActivate(stepIDData.SteplibSource, stepIDData.IDorURI, stepIDData.Version, stepDir, stepYMLPth); err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			} else {
				log.Debugf("[BITRISE_CLI] - Step activated: (ID:%s) (version:%s)", stepIDData.IDorURI, stepIDData.Version)
			}
		} else {
			registerStepRunResults("", models.StepRunStatusCodeFailed, 1, fmt.Errorf("Invalid stepIDData: No SteplibSource or LocalPath defined (%v)", stepIDData), isLastStep, true)
			continue
		}

		// Fill step info with default step info, if exist
		mergedStep := workflowStep
		if stepYMLPth != "" {
			specStep, err := bitrise.ReadSpecStep(stepYMLPth)
			log.Debugf("Spec read from YML: %#v\n", specStep)
			if err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}

			mergedStep, err = models.MergeStepWith(specStep, workflowStep)
			if err != nil {
				registerStepRunResults("", models.StepRunStatusCodeFailed, 1, err, isLastStep, true)
				continue
			}
		}

		if mergedStep.SupportURL != nil {
			stepInfoPtr.SupportURL = *mergedStep.SupportURL
		}
		if mergedStep.SourceCodeURL != nil {
			stepInfoPtr.SourceCodeURL = *mergedStep.SourceCodeURL
		}

		//
		// Run step
		bitrise.PrintRunningStepHeader(stepInfoPtr, idx)
		if mergedStep.RunIf != nil && *mergedStep.RunIf != "" {
			outStr, err := tools.EnvmanJSONPrint(configs.InputEnvstorePath)
			if err != nil {
				registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeFailed, 1, fmt.Errorf("EnvmanJSONPrint failed, err: %s", err), isLastStep, false)
				continue
			}

			envList, err := envmanModels.NewEnvJSONList(outStr)
			if err != nil {
				registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeFailed, 1, fmt.Errorf("CreateFromJSON failed, err: %s", err), isLastStep, false)
				continue
			}

			isRun, err := bitrise.EvaluateTemplateToBool(*mergedStep.RunIf, configs.IsCIMode, configs.IsPullRequestMode, buildRunResults, envList)
			if err != nil {
				registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeFailed, 1, err, isLastStep, false)
				continue
			}
			if !isRun {
				registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeSkippedWithRunIf, 0, err, isLastStep, false)
				continue
			}
		}

		isAlwaysRun := stepmanModels.DefaultIsAlwaysRun
		if mergedStep.IsAlwaysRun != nil {
			isAlwaysRun = *mergedStep.IsAlwaysRun
		} else {
			log.Warn("Step (%s) mergedStep.IsAlwaysRun is nil, should not!", stepIDData.IDorURI)
		}

		if buildRunResults.IsBuildFailed() && !isAlwaysRun {
			registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeSkipped, 0, err, isLastStep, false)
		} else {
			exit, outEnvironments, err := runStep(mergedStep, stepIDData, stepDir, *environments, buildRunResults)
			*environments = append(*environments, outEnvironments...)
			if err != nil {
				if *mergedStep.IsSkippable {
					registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeFailedSkippable, exit, err, isLastStep, false)
				} else {
					registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeFailed, exit, err, isLastStep, false)
				}
			} else {
				registerStepRunResults(*mergedStep.RunIf, models.StepRunStatusCodeSuccess, 0, nil, isLastStep, false)
			}
		}
	}

	return buildRunResults
}

func runWorkflow(workflow models.WorkflowModel, steplibSource string, buildRunResults models.BuildRunResultsModel, environments *[]envmanModels.EnvironmentItemModel, isLastWorkflow bool) models.BuildRunResultsModel {
	bitrise.PrintRunningWorkflow(workflow.Title)

	*environments = append(*environments, workflow.Environments...)
	return activateAndRunSteps(workflow, steplibSource, buildRunResults, environments, isLastWorkflow)
}

func activateAndRunWorkflow(workflowID string, workflow models.WorkflowModel, bitriseConfig models.BitriseDataModel, buildRunResults models.BuildRunResultsModel, environments *[]envmanModels.EnvironmentItemModel, lastWorkflowID string) (models.BuildRunResultsModel, error) {
	var err error
	// Run these workflows before running the target workflow
	for _, beforeWorkflowID := range workflow.BeforeRun {
		beforeWorkflow, exist := bitriseConfig.Workflows[beforeWorkflowID]
		if !exist {
			return buildRunResults, fmt.Errorf("Specified Workflow (%s) does not exist!", beforeWorkflowID)
		}
		if beforeWorkflow.Title == "" {
			beforeWorkflow.Title = beforeWorkflowID
		}
		buildRunResults, err = activateAndRunWorkflow(beforeWorkflowID, beforeWorkflow, bitriseConfig, buildRunResults, environments, lastWorkflowID)
		if err != nil {
			return buildRunResults, err
		}
	}

	// Run the target workflow
	isLastWorkflow := (workflowID == lastWorkflowID)
	buildRunResults = runWorkflow(workflow, bitriseConfig.DefaultStepLibSource, buildRunResults, environments, isLastWorkflow)

	// Run these workflows after running the target workflow
	for _, afterWorkflowID := range workflow.AfterRun {
		afterWorkflow, exist := bitriseConfig.Workflows[afterWorkflowID]
		if !exist {
			return buildRunResults, fmt.Errorf("Specified Workflow (%s) does not exist!", afterWorkflowID)
		}
		if afterWorkflow.Title == "" {
			afterWorkflow.Title = afterWorkflowID
		}
		buildRunResults, err = activateAndRunWorkflow(afterWorkflowID, afterWorkflow, bitriseConfig, buildRunResults, environments, lastWorkflowID)
		if err != nil {
			return buildRunResults, err
		}
	}

	return buildRunResults, nil
}

func lastWorkflowIDInConfig(workflowToRunID string, bitriseConfig models.BitriseDataModel) (string, error) {
	workflowToRun, exist := bitriseConfig.Workflows[workflowToRunID]
	if !exist {
		return "", errors.New("No worfklow exist with ID: " + workflowToRunID)
	}

	if len(workflowToRun.AfterRun) > 0 {
		lastAfterID := workflowToRun.AfterRun[len(workflowToRun.AfterRun)-1]
		wfID, err := lastWorkflowIDInConfig(lastAfterID, bitriseConfig)
		if err != nil {
			return "", err
		}
		workflowToRunID = wfID
	}
	return workflowToRunID, nil
}

// RunWorkflowWithConfiguration ...
func runWorkflowWithConfiguration(
	startTime time.Time,
	workflowToRunID string,
	bitriseConfig models.BitriseDataModel,
	secretEnvironments []envmanModels.EnvironmentItemModel) (models.BuildRunResultsModel, error) {

	if err := configs.InitPaths(); err != nil {
		return models.BuildRunResultsModel{}, fmt.Errorf("Failed to initialize required paths: %s", err)
	}

	workflowToRun, exist := bitriseConfig.Workflows[workflowToRunID]
	if !exist {
		return models.BuildRunResultsModel{}, fmt.Errorf("Specified Workflow (%s) does not exist!", workflowToRunID)
	}

	if workflowToRun.Title == "" {
		workflowToRun.Title = workflowToRunID
	}

	// Envman setup
	if err := os.Setenv(configs.EnvstorePathEnvKey, configs.OutputEnvstorePath); err != nil {
		return models.BuildRunResultsModel{}, fmt.Errorf("Failed to add env, err: %s", err)
	}

	if err := os.Setenv(configs.FormattedOutputPathEnvKey, configs.FormattedOutputPath); err != nil {
		return models.BuildRunResultsModel{}, fmt.Errorf("Failed to add env, err: %s", err)
	}

	if err := tools.EnvmanInit(); err != nil {
		return models.BuildRunResultsModel{}, errors.New("Failed to run envman init")
	}

	// App level environment
	environments := append(secretEnvironments, bitriseConfig.App.Environments...)

	if err := os.Setenv("BITRISE_TRIGGERED_WORKFLOW_ID", workflowToRunID); err != nil {
		return models.BuildRunResultsModel{}, fmt.Errorf("Failed to set BITRISE_TRIGGERED_WORKFLOW_ID env: %s", err)
	}
	if err := os.Setenv("BITRISE_TRIGGERED_WORKFLOW_TITLE", workflowToRun.Title); err != nil {
		return models.BuildRunResultsModel{}, fmt.Errorf("Failed to set BITRISE_TRIGGERED_WORKFLOW_TITLE env: %s", err)
	}

	environments = append(environments, workflowToRun.Environments...)

	lastWorkflowID, err := lastWorkflowIDInConfig(workflowToRunID, bitriseConfig)
	if err != nil {
		return models.BuildRunResultsModel{}, fmt.Errorf("Failed to get last workflow id: %s", err)
	}

	buildRunResults := models.BuildRunResultsModel{
		StartTime:      startTime,
		StepmanUpdates: map[string]int{},
	}

	buildRunResults, err = activateAndRunWorkflow(workflowToRunID, workflowToRun, bitriseConfig, buildRunResults, &environments, lastWorkflowID)
	if err != nil {
		return buildRunResults, errors.New("[BITRISE_CLI] - Failed to activate and run workflow " + workflowToRunID)
	}

	// Build finished
	bitrise.PrintSummary(buildRunResults)

	// Trigger WorkflowRunDidFinish
	if err := plugins.TriggerEvent(plugins.DidFinishRun, buildRunResults); err != nil {
		log.Warnf("Failed to trigger WorkflowRunDidFinish, error: %s", err)
	}

	return buildRunResults, nil
}
