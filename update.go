package clicommon

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kardianos/osext"
)

// Inspired by https://github.com/yitsushi/totp-cli/blob/master/command/update.go

const (
	updateConfigName = "autoupdate"

	binaryFilePermissions = 0755

	githubLatestReleaseTemplate = "https://api.github.com/repos/%s/releases/latest"
	githubReleaseAssetTemplate  = "https://api.github.com/repos/%s/releases/assets/%d"
)

type AutoUpdater struct {
	config       autoUpdaterConfig
	configDir    *UserConfigDir
	buildVersion string
	githubRepo   string
}

type autoUpdaterConfig struct {
	AutoUpdate     *bool  `json:"autoUpdate"`
	LastUpdateTime *int64 `json:"lastUpdateTime"`
}

type updatedRelease struct {
	version     string
	downloadURL string
}

func NewAutoUpdater(
	configDir *UserConfigDir,
	buildVersion, githubRepo string,
) (error, *AutoUpdater) {
	shouldSave := false

	config := autoUpdaterConfig{}

	err := configDir.LoadConfig(updateConfigName, &config)
	if err != nil {
		return err, nil
	}

	if config.AutoUpdate == nil {
		fmt.Println("Automatic updating has not been configured, would you like to enable it? (only checks for updates every 24 hours)")
		shouldAutoUpdate := CliQuestionYesNoDefault("Auto update?", true)

		config.AutoUpdate = &shouldAutoUpdate

		var otherState string

		if shouldAutoUpdate {
			otherState = "disable"
		} else {
			otherState = "enable"
		}

		fmt.Println("You can " + otherState + " this later by running '" + os.Args[0] + " config autoupdate'")

		shouldSave = true
	}

	updater := &AutoUpdater{
		config:       config,
		configDir:    configDir,
		buildVersion: buildVersion,
		githubRepo:   githubRepo,
	}

	if shouldSave {
		err = updater.save()
		if err != nil {
			return err, nil
		}
	}

	return nil, updater
}

// TryAutoUpdateSelf checks for an update and replaces the existing executable
// with the new version if there is one. Update checks are debounced to every 24
// hours, and can be disabled with a config option.
func (updater *AutoUpdater) TryAutoUpdateSelf() error {
	err := updater.Init()
	if err != nil {
		return err
	}

	if *updater.config.AutoUpdate == false {
		return nil
	}

	update, err := updater.checkForUpdate(false)
	if err != nil {
		return err
	}

	if update != nil && updater.buildVersion != "dev" {
		err = update.apply(true)
		if err != nil {
			fmt.Println("Error performing self-update:", err)
		}
	}

	return nil
}

// TryManualUpdate checks for an update and replaces the existing executable
// with the new version if there is one. This will always run, without any
// debouncing or config options to disable it.
func (updater *AutoUpdater) TryManualUpdate() error {
	err := updater.Init()
	if err != nil {
		return err
	}

	update, err := updater.checkForUpdate(true)
	if err != nil {
		return err
	}

	if update != nil {
		err = update.apply(false)
		if err != nil {
			fmt.Println("Error performing self-update:", err)
		}
	} else {
		fmt.Println("No updates found")
	}

	return nil
}

// GetAutoUpdate gets if automatic updates are enabled
func GetAutoUpdate(configDir *UserConfigDir) (bool, error) {
	config := autoUpdaterConfig{}

	err := configDir.LoadConfig(updateConfigName, &config)
	if err != nil {
		return false, err
	}

	return *config.AutoUpdate, nil
}

// SetAutoUpdate configures if automatic updates are enabled
func SetAutoUpdate(configDir *UserConfigDir, shouldAutoUpdate bool) (bool, error) {
	config := autoUpdaterConfig{}

	err := configDir.LoadConfig(updateConfigName, &config)
	if err != nil {
		return false, err
	}

	changed := *config.AutoUpdate != shouldAutoUpdate

	if changed {
		config.AutoUpdate = &shouldAutoUpdate

		err = configDir.SaveConfig(updateConfigName, &config)
		if err != nil {
			return false, err
		}
	}

	return changed, nil
}

func (updater *AutoUpdater) Init() error {

	return nil
}

func (updater *AutoUpdater) save() error {
	return updater.configDir.SaveConfig(updateConfigName, &updater.config)
}

func (updater *AutoUpdater) checkForUpdate(force bool) (*updatedRelease, error) {
	type releaseAsset struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	type releaseResponse struct {
		TagName string         `json:"tag_name"`
		Assets  []releaseAsset `json:"assets"`
	}

	now := time.Now().Unix()

	if !force {
		if updater.config.LastUpdateTime != nil && *updater.config.LastUpdateTime > now-24*60*60 {
			return nil, nil
		}
	}

	fmt.Println("Checking for updates...")

	updater.config.LastUpdateTime = &now
	err := updater.save()
	if err != nil {
		return nil, err
	}

	var releaseData releaseResponse

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf(githubLatestReleaseTemplate, updater.githubRepo), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(body, &releaseData)
		if err != nil {
			return nil, err
		}

		remoteVersion := normalizeVersion(releaseData.TagName)
		localVersion := normalizeVersion(updater.buildVersion)

		if remoteVersion != localVersion {
			break
		}

		return nil, nil
	}

	update := updatedRelease{
		version: releaseData.TagName,
	}

	expectedSuffix := runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"

	for _, asset := range releaseData.Assets {
		if strings.HasSuffix(asset.Name, expectedSuffix) {
			update.downloadURL = fmt.Sprintf(githubReleaseAssetTemplate, updater.githubRepo, asset.ID)
			break
		}
	}

	return &update, nil
}

func (update *updatedRelease) apply(restart bool) error {
	fmt.Println("Updating to", update.version)

	thisExe, err := osext.Executable()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", update.downloadURL, nil)
	req.Header.Set("Accept", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Did not get 200 status code from update download")
	}

	gzipReader, _ := gzip.NewReader(resp.Body)
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err != nil {
			return err
		}

		if header.Name == "secrets" {
			break
		}
	}

	parentDir := filepath.Dir(thisExe)
	needsSudo := false
	tmpDir := parentDir

	file, err := ioutil.TempFile(tmpDir, filepath.Base(thisExe))
	if err != nil {
		needsSudo = true
		tmpDir = os.TempDir()

		file, err = ioutil.TempFile(tmpDir, filepath.Base(thisExe))
		if err != nil {
			return err
		}
	}

	defer file.Close()

	_, err = io.Copy(file, tarReader)
	if err != nil {
		return err
	}

	err = file.Chmod(binaryFilePermissions)
	if err != nil {
		return err
	}

	file.Close()

	if needsSudo {
		err = CallSudo("replaceExecutable", file.Name())
		if err != nil {
			return err
		}
	} else {
		err = os.Rename(file.Name(), thisExe)
		if err != nil {
			return err
		}
	}

	if restart {
		fmt.Println("Complete, restarting command...")

		env := os.Environ()
		args := os.Args
		err = syscall.Exec(thisExe, args, env)
		if err != nil {
			panic(err)
		}

		panic("Exec new updated process failed silently???")
	}

	return nil
}

func normalizeVersion(version string) string {
	if matches, _ := regexp.Match(`^v\d`, []byte(version)); matches {
		return version[1:]
	}

	return version
}
