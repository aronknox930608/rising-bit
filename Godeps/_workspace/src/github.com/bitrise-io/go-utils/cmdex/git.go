package cmdex

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"
)

// GitClone ...
func GitClone(uri, pth string) (err error) {
	if uri == "" {
		return errors.New("Git Clone 'uri' missing")
	}
	if pth == "" {
		return errors.New("Git Clone 'pth' missing")
	}
	if err = RunCommand("git", "clone", "--recursive", uri, pth); err != nil {
		log.Printf(" [!] Failed to git clone from (%s) to (%s)", uri, pth)
		return
	}
	return
}

// GitCloneTagOrBranch ...
func GitCloneTagOrBranch(uri, pth, tagOrBranch string) error {
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

// GitCloneTagOrBranchAndValidateCommitHash ...
func GitCloneTagOrBranchAndValidateCommitHash(uri, pth, version, commithash string) (err error) {
	if uri == "" {
		return errors.New("Git Clone 'uri' missing")
	}
	if pth == "" {
		return errors.New("Git Clone 'pth' missing")
	}
	if version == "" {
		return errors.New("Git Clone 'version' missing")
	}
	if commithash == "" {
		return errors.New("Git Clone 'commithash' missing")
	}
	if err = RunCommand("git", "clone", "--recursive", uri, pth, "--branch", version); err != nil {
		return
	}

	// cleanup
	defer func() {
		if err != nil {
			if err := RemoveDir(pth); err != nil {
				log.Printf(" [!] Failed to cleanup path (%s) error: (%v) ", pth, err)
			}
		}
	}()

	latestCommit, err := GitGetLatestCommitHashOnHead(pth)
	if err != nil {
		return
	}
	if commithash != latestCommit {
		return fmt.Errorf("Commit hash doesn't match the one specified for the version tag. (version tag: %s) (expected commit hash: %s) (got: %s)", version, latestCommit, commithash)
	}

	return
}

// GitPull ...
func GitPull(pth string) error {
	err := RunCommandInDir(pth, "git", []string{"pull"}...)
	if err != nil {
		log.Printf(" [!] Git pull failed, error (%v)", err)
		return err
	}
	return nil
}

// GitUpdate ...
func GitUpdate(git, pth string) error {
	if exists, err := pathutil.IsPathExists(pth); err != nil {
		return err
	} else if !exists {
		fmt.Println("[STEPMAN] - Git path does not exist, do clone")
		return GitClone(git, pth)
	}

	fmt.Println("[STEPMAN] - Git path exist, do pull")
	return GitPull(pth)
}

// GitCheckout ...
func GitCheckout(dir, commithash string) error {
	if commithash == "" {
		return errors.New("Git Clone 'hash' missing")
	}
	return RunCommandInDir(dir, "git", "checkout", commithash)
}

// GitCheckoutBranch ...
func GitCheckoutBranch(repoPath, branch string) error {
	if branch == "" {
		return errors.New("Git checkout 'branch' missing")
	}
	if err := GitCheckout(repoPath, branch); err != nil {
		return RunCommandInDir(repoPath, "git", "checkout", "-b", branch)
	}
	return nil
}

// GitAddFile ...
func GitAddFile(repoPath, filePath string) error {
	if filePath == "" {
		return errors.New("Git add 'file' missing")
	}
	return RunCommandInDir(repoPath, "git", "add", filePath)
}

// GitPushToOrigin ...
func GitPushToOrigin(repoPath, branch string) error {
	return RunCommandInDir(repoPath, "git", "push", "-u", "origin", branch)
}

// GitCheckIsNoChanges ...
func GitCheckIsNoChanges(repoPath string) error {
	return RunCommandInDir(repoPath, "git", "diff", "--cached", "--exit-code", "--quiet")
}

// GitCommit ...
func GitCommit(repoPath string, message string) error {
	if message == "" {
		return errors.New("Git commit 'message' missing")
	}
	if err := GitCheckIsNoChanges(repoPath); err != nil {
		return RunCommandInDir(repoPath, "git", "commit", "-m", message)
	}
	return nil
}

// GitGetLatestCommitHashOnHead ...
func GitGetLatestCommitHashOnHead(pth string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = pth
	bytes, err := cmd.CombinedOutput()
	cmdOutput := string(bytes)
	if err != nil {
		log.Printf(" [!] Output: %v", cmdOutput)
		return "", err
	}
	return strings.TrimSpace(cmdOutput), nil
}

// GitGetCommitHashOfHEAD ...
func GitGetCommitHashOfHEAD(pth string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = pth
	bytes, err := cmd.CombinedOutput()
	cmdOutput := string(bytes)
	if err != nil {
		log.Printf(" [!] Output: %s", cmdOutput)
		return "", err
	}
	return strings.TrimSpace(cmdOutput), nil
}
