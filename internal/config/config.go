package config

import (
	"encoding/json"
	"os"
	"fmt"
)

type Config struct {
	DbUrl string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (*Config, error) {
	filePath, err := getConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("error while reading config")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error while reading config")
	}
	defer file.Close()

	var payload Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("error while reading config")
	}

	return &payload, nil
}

func (c Config) SetUser(userName string) error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("error while setting user")
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error while setting user")
	}
	defer file.Close()

	c.CurrentUserName = userName

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("error while setting user")
	}

	return nil
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()

	if err != nil {
		return "", fmt.Errorf("error while reading config path")
	}

	return home + "/.gatorconfig.json", nil
}
