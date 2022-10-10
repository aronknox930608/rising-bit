package tools

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bitrise-io/bitrise/configs"
	"github.com/bitrise-io/bitrise/tools/errorfinder"
	"github.com/bitrise-io/bitrise/tools/filterwriter"
	"github.com/bitrise-io/bitrise/tools/timeoutcmd"
	envman "github.com/bitrise-io/envman/cli"
	envmanEnv "github.com/bitrise-io/envman/env"
	envmanModels "github.com/bitrise-io/envman/models"
	"github.com/bitrise-io/go-utils/command"
	utilslog "github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	stepman "github.com/bitrise-io/stepman/cli"
	stepmanModels "github.com/bitrise-io/stepman/models"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// UnameGOOS ...
func UnameGOOS() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "Darwin", nil
	case "linux":
		return "Linux", nil
	}
	return "", fmt.Errorf("Unsupported platform (%s)", runtime.GOOS)
}

// UnameGOARCH ...
func UnameGOARCH() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "arm64", nil
	}
	return "", fmt.Errorf("Unsupported architecture (%s)", runtime.GOARCH)
}

// InstallToolFromGitHub ...
func InstallToolFromGitHub(toolname, githubUser, toolVersion string) error {
	unameGOOS, err := UnameGOOS()
	if err != nil {
		return fmt.Errorf("Failed to determine OS: %s", err)
	}
	unameGOARCH, err := UnameGOARCH()
	if err != nil {
		return fmt.Errorf("Failed to determine ARCH: %s", err)
	}
	downloadURL := "https://github.com/" + githubUser + "/" + toolname + "/releases/download/" + toolVersion + "/" + toolname + "-" + unameGOOS + "-" + unameGOARCH

	return InstallFromURL(toolname, downloadURL)
}

// DownloadFile ...
func DownloadFile(downloadURL, targetDirPath string) error {
	outFile, err := os.Create(targetDirPath)
	defer func() {
		if err := outFile.Close(); err != nil {
			log.Warnf("Failed to close (%s)", targetDirPath)
		}
	}()
	if err != nil {
		return fmt.Errorf("failed to create (%s), error: %s", targetDirPath, err)
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download from (%s), error: %s", downloadURL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("failed to close (%s) body", downloadURL)
		}
	}()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download from (%s), error: %s", downloadURL, err)
	}

	return nil
}

// InstallFromURL ...
func InstallFromURL(toolBinName, downloadURL string) error {
	if len(toolBinName) < 1 {
		return fmt.Errorf("no Tool (bin) Name provided! URL was: %s", downloadURL)
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("__tmp_download_dest__")
	if err != nil {
		return fmt.Errorf("failed to create tmp dir for download destination")
	}
	tmpDestinationPth := filepath.Join(tmpDir, toolBinName)

	if err := DownloadFile(downloadURL, tmpDestinationPth); err != nil {
		return fmt.Errorf("failed to download, error: %s", err)
	}

	bitriseToolsDirPath := configs.GetBitriseToolsDirPath()
	destinationPth := filepath.Join(bitriseToolsDirPath, toolBinName)

	if exist, err := pathutil.IsPathExists(destinationPth); err != nil {
		return fmt.Errorf("failed to check if file exist (%s), error: %s", destinationPth, err)
	} else if exist {
		if err := os.Remove(destinationPth); err != nil {
			return fmt.Errorf("failed to remove file (%s), error: %s", destinationPth, err)
		}
	}

	if err := MoveFile(tmpDestinationPth, destinationPth); err != nil {
		return fmt.Errorf("failed to copy (%s) to (%s), error: %s", tmpDestinationPth, destinationPth, err)
	}

	if err := os.Chmod(destinationPth, 0755); err != nil {
		return fmt.Errorf("failed to make file (%s) executable, error: %s", destinationPth, err)
	}

	return nil
}

// ------------------
// --- Stepman

// StepmanSetup ...
func StepmanSetup(collection string) error {
	return stepman.Setup(collection, "", utilslog.NewDefaultLogger(false))
}

// StepmanUpdate ...
func StepmanUpdate(collection string) error {
	return stepman.UpdateLibrary(collection, utilslog.NewDefaultLogger(false))
}

// StepmanActivate ...
func StepmanActivate(collection, stepID, stepVersion, dir, ymlPth string) error {
	return stepman.Activate(collection, stepID, stepVersion, dir, ymlPth, false, utilslog.NewDefaultLogger(false))
}

// StepmanStepInfo ...
func StepmanStepInfo(collection, stepID, stepVersion string) (stepmanModels.StepInfoModel, error) {
	return stepman.QueryStepInfo(collection, stepID, stepVersion, utilslog.NewDefaultLogger(false))
}

// StepmanRawStepList ...
func StepmanRawStepList(collection string) (string, error) {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "step-list", "--collection", collection, "--format", "raw"}
	return command.RunCommandAndReturnCombinedStdoutAndStderr("stepman", args...)
}

// StepmanJSONStepList ...
func StepmanJSONStepList(collection string) (string, error) {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "step-list", "--collection", collection, "--format", "json"}

	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer

	if err := command.RunCommandWithWriters(io.Writer(&outBuffer), io.Writer(&errBuffer), "stepman", args...); err != nil {
		return outBuffer.String(), fmt.Errorf("Error: %s, details: %s", err, errBuffer.String())
	}

	return outBuffer.String(), nil
}

//
// Share

// StepmanShare ...
func StepmanShare() error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "share", "--toolmode"}
	return command.RunCommand("stepman", args...)
}

// StepmanShareAudit ...
func StepmanShareAudit() error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "share", "audit", "--toolmode"}
	return command.RunCommand("stepman", args...)
}

// StepmanShareCreate ...
func StepmanShareCreate(tag, git, stepID string) error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "share", "create", "--tag", tag, "--git", git, "--stepid", stepID, "--toolmode"}
	return command.RunCommand("stepman", args...)
}

// StepmanShareFinish ...
func StepmanShareFinish() error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "share", "finish", "--toolmode"}
	return command.RunCommand("stepman", args...)
}

// StepmanShareStart ...
func StepmanShareStart(collection string) error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "share", "start", "--collection", collection, "--toolmode"}
	return command.RunCommand("stepman", args...)
}

// ------------------
// --- Envman

// EnvmanInit ...
func EnvmanInit(envStorePth string, clear bool) error {
	return envman.InitEnvStore(envStorePth, clear)
}

// EnvmanAdd ...
func EnvmanAdd(envStorePth, key, value string, expand, skipIfEmpty, sensitive bool) error {
	return envman.AddEnv(envStorePth, key, value, expand, false, skipIfEmpty, sensitive)
}

// EnvmanAddEnvs ...
func EnvmanAddEnvs(envstorePth string, envsList []envmanModels.EnvironmentItemModel) error {
	for _, env := range envsList {
		key, value, err := env.GetKeyValuePair()
		if err != nil {
			return err
		}

		opts, err := env.GetOptions()
		if err != nil {
			return err
		}

		isExpand := envmanModels.DefaultIsExpand
		if opts.IsExpand != nil {
			isExpand = *opts.IsExpand
		}

		skipIfEmpty := envmanModels.DefaultSkipIfEmpty
		if opts.SkipIfEmpty != nil {
			skipIfEmpty = *opts.SkipIfEmpty
		}

		sensitive := envmanModels.DefaultIsSensitive
		if opts.IsSensitive != nil {
			sensitive = *opts.IsSensitive
		}

		if err := EnvmanAdd(envstorePth, key, value, isExpand, skipIfEmpty, sensitive); err != nil {
			return err
		}
	}
	return nil
}

// EnvmanReadEnvList ...
func EnvmanReadEnvList(envStorePth string) (envmanModels.EnvsJSONListModel, error) {
	return envman.ReadEnvsJSONList(envStorePth, true, false, &envmanEnv.DefaultEnvironmentSource{})
}

// EnvmanClear ...
func EnvmanClear(envStorePth string) error {
	return envman.ClearEnvs(envStorePth)
}

// EnvmanRun runs a command through envman.
func EnvmanRun(envStorePth,
	workDirPth string,
	cmdArgs []string,
	timeout time.Duration,
	noOutputTimeout time.Duration,
	secrets []string,
	stdInPayload []byte,
) (int, error) {
	envs, err := envman.ReadAndEvaluateEnvs(envStorePth, &envmanEnv.DefaultEnvironmentSource{})
	if err != nil {
		return 1, err
	}

	var inReader io.Reader
	var outWriter io.Writer
	var errWriter io.Writer
	errorFinder := errorfinder.NewErrorFinder()
	var fw *filterwriter.Writer

	if !configs.IsSecretFiltering {
		outWriter = errorFinder.WrapWriter(os.Stdout)
		errWriter = errorFinder.WrapWriter(os.Stderr)
	} else {
		fw = filterwriter.New(secrets, os.Stdout)
		outWriter = errorFinder.WrapWriter(fw)
		errWriter = outWriter
	}

	inReader = os.Stdin
	if stdInPayload != nil {
		inReader = bytes.NewReader(stdInPayload)
	}

	name := cmdArgs[0]
	var args []string
	if len(cmdArgs) > 1 {
		args = cmdArgs[1:]
	}

	cmd := timeoutcmd.New(workDirPth, name, args...)
	cmd.SetTimeout(timeout)
	cmd.SetHangTimeout(noOutputTimeout)
	cmd.SetStandardIO(inReader, outWriter, errWriter)
	cmd.SetEnv(append(envs, "PWD="+workDirPth))

	err = cmd.Start()

	// flush the writer anyway if the process is finished
	if configs.IsSecretFiltering {
		_, ferr := fw.Flush()
		if ferr != nil {
			return 1, errorFinder.WrapError(ferr)
		}
	}

	return timeoutcmd.ExitStatus(err), errorFinder.WrapError(err)
}

// ------------------
// --- Utility

// GetSecretValues filters out built in configuration parameters from the secret envs
func GetSecretValues(secrets []envmanModels.EnvironmentItemModel) []string {
	var secretValues []string
	for _, secret := range secrets {
		key, value, err := secret.GetKeyValuePair()
		if err != nil || len(value) < 1 || IsBuiltInFlagTypeKey(key) {
			if err != nil {
				log.Warnf("Error getting key-value pair from secret (%v): %s", secret, err)
			}
			continue
		}
		secretValues = append(secretValues, value)
	}

	return secretValues
}

// MoveFile ...
func MoveFile(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err == nil {
		return nil
	}

	if linkErr, ok := err.(*os.LinkError); ok {
		if linkErr.Err == unix.EXDEV {
			info, err := os.Stat(oldpath)
			if err != nil {
				return err
			}

			data, err := ioutil.ReadFile(oldpath)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(newpath, data, info.Mode())
			if err != nil {
				return err
			}

			return os.Remove(oldpath)
		}
	}

	return err
}

// IsBuiltInFlagTypeKey returns true if the env key is a built-in flag type env key
func IsBuiltInFlagTypeKey(env string) bool {
	switch string(env) {
	case configs.IsSecretFilteringKey,
		configs.IsSecretEnvsFilteringKey,
		configs.CIModeEnvKey,
		configs.PRModeEnvKey,
		configs.DebugModeEnvKey,
		configs.LogLevelEnvKey,
		configs.PullRequestIDEnvKey:
		return true
	default:
		return false
	}
}
