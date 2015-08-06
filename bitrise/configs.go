package bitrise

import (
	"os"
	"path"

	"github.com/bitrise-io/go-pathutil/pathutil"
)

const (
	bitriseVersionSetupStateFileName = "version.setup"
)

func getBitriseConfigsDirPath() string {
	return path.Join(pathutil.UserHomeDir(), ".bitrise")
}

func getBitriseConfigVersionSetupFilePath() string {
	return path.Join(getBitriseConfigsDirPath(), bitriseVersionSetupStateFileName)
}

func ensureBitriseConfigDirExists() error {
	confDirPth := getBitriseConfigsDirPath()
	isExists, err := pathutil.IsDirExists(confDirPth)
	if !isExists || err != nil {
		if err := os.MkdirAll(confDirPth, 0777); err != nil {
			return err
		}
	}
	return nil
}

// CheckSetupForVersion ...
func CheckSetupForVersion(ver string) bool {
	configPth := getBitriseConfigVersionSetupFilePath()
	cont, err := ReadStringFromFile(configPth)
	if err != nil {
		return false
	}
	return (cont == ver)
}

// SaveSetupSuccessForVersion ...
func SaveSetupSuccessForVersion(ver string) error {
	if err := ensureBitriseConfigDirExists(); err != nil {
		return err
	}
	configPth := getBitriseConfigVersionSetupFilePath()
	return WriteStringToFile(configPth, ver)
}
