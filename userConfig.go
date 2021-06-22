package clicommon

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
)

type UserConfigDir struct {
	name string
}

func NewUserConfigDir(name string) *UserConfigDir {
	return &UserConfigDir{
		name: name,
	}
}

// GetConfigDir gets the os-dependent user configuration directory
func (userConfigDir *UserConfigDir) GetConfigDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "linux", "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		dir = fmt.Sprintf("%s/.config/%s/", home, userConfigDir.name)

	case "windows":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		dir = fmt.Sprintf("%s\\AppData\\Roaming\\%s\\", home, userConfigDir.name)
	}

	info, err := os.Stat(dir)
	if err != nil {
		err = os.MkdirAll(dir, 0770)
		if err != nil {
			return "", err
		}
	} else if !info.IsDir() {
		return "", errors.New("config path is not a directory")
	}

	return dir, nil
}

// LoadConfig loads a user config file of the given name
func (userConfigDir *UserConfigDir) LoadConfig(config string, data interface{}) error {
	dir, err := userConfigDir.GetConfigDir()
	if err != nil {
		return err
	}

	filename := dir + config + ".json"

	text, err := ioutil.ReadFile(filename)
	if err != nil {
		text = []byte("{}")
	}

	err = json.Unmarshal(text, data)
	if err != nil {
		return err
	}

	return nil
}

// SaveConfig saves a user config file of the given name
func (userConfigDir *UserConfigDir) SaveConfig(config string, data interface{}) error {
	dir, err := userConfigDir.GetConfigDir()
	if err != nil {
		return err
	}

	filename := dir + config + ".json"

	text, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, text, 0660)
	if err != nil {
		return err
	}

	return nil
}

// LoadExternalConfig reads an external config file, outside of the default user
// config directory
func LoadExternalConfig(filename string, data interface{}) error {
	text, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(text, data)
	if err != nil {
		return err
	}

	return nil
}

// SaveExternalConfig saves a file outside of the default user config directory
func SaveExternalConfig(filename string, data interface{}) error {
	text, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, text, 0666)
	if err != nil {
		return err
	}

	return nil
}
