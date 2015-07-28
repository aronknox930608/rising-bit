package bitrise

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// RunCopyFile ...
func RunCopyFile(src, dst string) error {
	args := []string{src, dst}
	return RunCommand("rsync", args...)
}

// RunCopyDir ...
func RunCopyDir(src, dst string, isOnlyContent bool) error {
	if isOnlyContent && !strings.HasSuffix(src, "/") {
		src = src + "/"
	}
	args := []string{"-r", src, dst}
	return RunCommand("rsync", args...)
}

// RunGitClone ...
func RunGitClone(uri, pth, tagOrBranch string) error {
	if uri == "" {
		return errors.New("Git Clone 'uri' missing")
	}
	if pth == "" {
		return errors.New("Git Clone 'path' missing")
	}
	if tagOrBranch == "" {
		return errors.New("Git Clone 'tag or branch' missing")
	}
	return RunCommand("git", []string{"clone", "--recursive", "--branch", tagOrBranch, uri, pth}...)
}

// ------------------
// --- Stepman

// RunStepmanSetup ...
func RunStepmanSetup(collection string) error {
	logLevel := log.GetLevel().String()
	args := []string{"--debug", "--loglevel", logLevel, "setup", "--collection", collection}
	return RunCommand("stepman", args...)
}

// RunStepmanActivate ...
func RunStepmanActivate(collection, stepID, stepVersion, dir, ymlPth string) error {
	logLevel := log.GetLevel().String()
	args := []string{"--debug", "--loglevel", logLevel, "activate", "--collection", collection,
		"--id", stepID, "--version", stepVersion, "--path", dir, "--copyyml", ymlPth}
	return RunCommand("stepman", args...)
}

// ------------------
// --- Envman

// RunEnvmanInit ...
func RunEnvmanInit() error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "init"}
	return RunCommand("envman", args...)
}

// RunEnvmanAdd ...
func RunEnvmanAdd(key, value string, expand bool) error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "add", "--key", key, "--append"}
	if !expand {
		args = []string{"--loglevel", logLevel, "add", "--key", key, "--no-expand", "--append"}
	}

	envman := exec.Command("envman", args...)
	envman.Stdin = strings.NewReader(value)
	envman.Stdout = os.Stdout
	envman.Stderr = os.Stderr
	return envman.Run()
}

// RunEnvmanRunInDir ...
func RunEnvmanRunInDir(dir string, cmd []string, logLevel string) error {
	if logLevel == "" {
		logLevel = log.GetLevel().String()
	}
	args := []string{"--loglevel", logLevel, "run"}
	args = append(args, cmd...)
	return RunCommandInDir(dir, "envman", args...)
}

// RunEnvmanRun ...
func RunEnvmanRun(cmd []string) error {
	return RunEnvmanRunInDir("", cmd, "")
}

// RunEnvmanEnvstoreTest ...
func RunEnvmanEnvstoreTest(pth string) error {
	logLevel := log.GetLevel().String()
	args := []string{"--loglevel", logLevel, "--path", pth, "print"}
	cmd := exec.Command("envman", args...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ------------------
// --- Common

// RunCommandAndReturnStdout ..
func RunCommandAndReturnStdout(cmdName string, cmdArgs ...string) (string, error) {
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stderr = os.Stderr
	outBytes, err := cmd.Output()
	outStr := string(outBytes)
	return strings.TrimSpace(outStr), err
}

// RunBashCommand ...
func RunBashCommand(cmdStr string) error {
	return RunCommand("bash", "-c", cmdStr)
}

// RunBashCommandLines ...
func RunBashCommandLines(cmdLines []string) error {
	for _, aLine := range cmdLines {
		if err := RunCommand("bash", "-c", aLine); err != nil {
			return err
		}
	}
	return nil
}

// RunCopy ...
func RunCopy(src, dst string) error {
	args := []string{src, dst}
	return RunCommand("rsync", args...)
}

// RunCommand ...
func RunCommand(name string, args ...string) error {
	return RunCommandInDir("", name, args...)
}

// RunCommandInDir ...
func RunCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if dir != "" {
		cmd.Dir = dir
	}
	log.Debugln("Run command: $", cmd)
	return cmd.Run()
}
