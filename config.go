package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var configDir = filepath.Join(os.Getenv("HOME"), ".config", "hop")
var configPath = filepath.Join(configDir, "config.yaml")
var oldConfigDir = filepath.Join(os.Getenv("HOME"), ".config", "devjump")
var oldConfigPath = filepath.Join(oldConfigDir, "config.yaml")

type PathAlias struct {
	Alias   string `yaml:"alias"`
	Path    string `yaml:"path"`
	Command string `yaml:"command,omitempty"`
}

type Host struct {
	Alias string      `yaml:"alias"`
	Host  string      `yaml:"host"`
	User  string      `yaml:"user"`
	Port  int         `yaml:"port"`
	Paths []PathAlias `yaml:"paths"`
}

type Config struct {
	Hosts []Host `yaml:"hosts"`
}

type legacyProject struct {
	Alias  string `yaml:"alias"`
	Host   string `yaml:"host"`
	User   string `yaml:"user"`
	Port   int    `yaml:"port"`
	Path   string `yaml:"path"`
	DevURL string `yaml:"dev_url,omitempty"`
}

type legacyConfig struct {
	Projects []legacyProject `yaml:"projects"`
}

func defaultConfig() *Config {
	return &Config{Hosts: []Host{}}
}

func configExists() bool {
	_, err := os.Stat(configPath)
	return err == nil
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

func InitConfig() error {
	cfg := defaultConfig()
	return SaveConfig(cfg)
}

func migrateOldConfig() error {
	if configExists() {
		return nil
	}
	if _, err := os.Stat(oldConfigPath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(oldConfigPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return err
	}
	fmt.Println("Config lama dari ~/.config/devjump/ berhasil dimigrasikan ke ~/.config/hop/")
	fmt.Println("Config lama TIDAK dihapus, silakan hapus manual jika sudah yakin tidak diperlukan.")
	return nil
}

func migrateSchemaV1ToV2() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err == nil && len(cfg.Hosts) > 0 {
		return nil
	}

	if len(cfg.Hosts) == 0 {
		var legacy legacyConfig
		if err := yaml.Unmarshal(data, &legacy); err != nil || len(legacy.Projects) == 0 {
			return nil
		}

		backupPath := configPath + ".v1.bak"
		if err := os.WriteFile(backupPath, data, 0600); err != nil {
			return err
		}

		for _, lp := range legacy.Projects {
			h := Host{
				Alias: lp.Alias,
				Host:  lp.Host,
				User:  lp.User,
				Port:  lp.Port,
				Paths: []PathAlias{
					{Alias: lp.Alias, Path: lp.Path},
				},
			}
			cfg.Hosts = append(cfg.Hosts, h)
		}

		if err := SaveConfig(&cfg); err != nil {
			return err
		}

		fmt.Println("Config lama (skema v1) berhasil dimigrasikan ke skema v2. Backup tersimpan di config.yaml.v1.bak.")
	}

	return nil
}
