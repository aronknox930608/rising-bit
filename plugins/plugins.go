package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/pathutil"
)

func getOsAndArch() (string, string, error) {
	osOut, err := cmdex.RunCommandAndReturnCombinedStdoutAndStderr("uname", "-s")
	if err != nil {
		return "", "", err
	}

	archOut, err := cmdex.RunCommandAndReturnCombinedStdoutAndStderr("uname", "-m")
	if err != nil {
		return "", "", err
	}

	return osOut, archOut, nil
}

// DownloadPluginFromURL ....
func DownloadPluginFromURL(url, dst string) error {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]

	OS, arch, err := getOsAndArch()
	if err != nil {
		return err
	}

	urlWithSuffix := url
	urlSuffix := fmt.Sprintf("-%s-%s", OS, arch)
	if !strings.HasSuffix(url, urlSuffix) {
		urlWithSuffix = urlWithSuffix + urlSuffix
	}

	urls := []string{urlWithSuffix, url}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("plugin")
	if err != nil {
		return err
	}
	tmpDst := path.Join(tmpDir, fileName)
	output, err := os.Create(tmpDst)
	if err != nil {
		return err
	}
	defer func() {
		if err := output.Close(); err != nil {
			log.Errorf("Failed to close file, err: %s", err)
		}
	}()

	success := false
	var response *http.Response
	for _, aURL := range urls {

		response, err = http.Get(aURL)
		if response != nil {
			defer func() {
				if err := response.Body.Close(); err != nil {
					log.Errorf("Failed to close response body, err: %s", err)
				}
			}()
		}

		if err != nil {
			log.Errorf("%s", err)
		} else {
			success = true
			break
		}
	}
	if !success {
		return err
	}

	if _, err := io.Copy(output, response.Body); err != nil {
		return err
	}
	if err := cmdex.CopyFile(output.Name(), dst); err != nil {
		return err
	}

	return nil
}

// InstallPlugin ...
func InstallPlugin(pluginSource, pluginName, pluginType string) (string, error) {
	pluginPath, err := GetPluginPath(pluginName, pluginType)
	if err != nil {
		return "", err
	}

	if err := DownloadPluginFromURL(pluginSource, pluginPath); err != nil {
		return "", err
	}

	if err := os.Chmod(pluginPath, 0777); err != nil {
		return "", err
	}

	printableName := PrintableName(pluginName, pluginType)
	return printableName, nil
}

// DeletePlugin ...
func DeletePlugin(pluginName, pluginType string) error {
	pluginPath, err := GetPluginPath(pluginName, pluginType)
	if err != nil {
		return err
	}

	if exists, err := pathutil.IsPathExists(pluginPath); err != nil {
		return fmt.Errorf("Failed to check dir (%s), err: %s", pluginPath, err)
	} else if !exists {
		return fmt.Errorf("Plugin (%s) not installed", PrintableName(pluginName, pluginType))
	}
	return os.Remove(pluginPath)
}

// ListPlugins ...
func ListPlugins() (map[string][]Plugin, error) {
	collectPlugin := func(dir, pluginType string) ([]Plugin, error) {
		plugins := []Plugin{}

		pluginsPath, err := GetPluginPath("", pluginType)
		if err != nil {
			return []Plugin{}, err
		}

		files, err := ioutil.ReadDir(pluginsPath)
		if err != nil {
			return []Plugin{}, err
		}
		for _, file := range files {
			if !strings.HasPrefix(file.Name(), ".") {
				plugin, found, err := GetPlugin(file.Name(), pluginType)
				if err != nil {
					return []Plugin{}, err
				}
				if found {
					plugins = append(plugins, plugin)
				}
			}
		}
		return plugins, nil
	}

	pluginMap := map[string][]Plugin{}
	pluginsPath, err := GetPluginsDir()
	if err != nil {
		return map[string][]Plugin{}, err
	}

	pluginTypes := []string{TypeGeneric, TypeInit, TypeRun}
	for _, pType := range pluginTypes {
		ps, err := collectPlugin(pluginsPath, pType)
		if err != nil {
			return map[string][]Plugin{}, err
		}
		pluginMap[pType] = ps
	}

	return pluginMap, nil
}

// ParseArgs ...
func ParseArgs(args []string) (string, string, []string, bool) {
	const bitrisePluginPrefix = ":"

	log.Debugf("args: %v", args)

	if len(args) > 0 {
		plugin := ""
		pluginArgs := []string{}
		for idx, arg := range args {
			if strings.Contains(arg, bitrisePluginPrefix) {
				plugin = arg
				pluginArgs = args[idx:len(args)]
			}
		}

		// generic plugins
		if strings.HasPrefix(plugin, bitrisePluginPrefix) {
			pluginName := strings.TrimPrefix(plugin, bitrisePluginPrefix)
			return pluginName, TypeGeneric, pluginArgs, true
		}

		// typed plugins
		if strings.Contains(plugin, ":") {
			pluginSplits := strings.Split(plugin, ":")
			if len(pluginSplits) == 2 {
				pluginType := pluginSplits[0]
				pluginName := pluginSplits[1]
				return pluginName, pluginType, pluginArgs, true
			}
		}
	}

	return "", "", []string{}, false
}

// GetPlugin ...
func GetPlugin(name, pluginType string) (Plugin, bool, error) {
	pluginPath, err := GetPluginPath(name, pluginType)
	if err != nil {
		return Plugin{}, false, err
	}

	if exists, err := pathutil.IsPathExists(pluginPath); err != nil {
		return Plugin{}, false, fmt.Errorf("Failed to check dir (%s), err: %s", pluginPath, err)
	} else if !exists {
		return Plugin{}, false, nil
	}

	plugin := Plugin{
		Name: name,
		Path: pluginPath,
		Type: pluginType,
	}

	return plugin, true, nil
}

// RunPlugin ...
func RunPlugin(bitriseVersion string, plugin Plugin, args []string) (string, error) {
	var outBuffer bytes.Buffer

	bitriseInfos := map[string]string{
		"version": bitriseVersion,
	}
	bitriseInfosStr, err := json.Marshal(bitriseInfos)
	if err != nil {
		return "", err
	}

	pluginArgs := []string{string(bitriseInfosStr)}
	pluginArgs = append(pluginArgs, args...)

	err = cmdex.RunCommandWithWriters(io.Writer(&outBuffer), os.Stderr, plugin.Path, pluginArgs...)
	return outBuffer.String(), err
}
