package config

import (
	"errors"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Cron   string `yaml:"cron"`
	Watchs struct {
		Docker struct {
			Images []string `yaml:"images"`
		} `yaml:"docker"`
		Git struct {
			Remote   string   `yaml:"remote"`
			Username string   `yaml:"username"`
			Password string   `yaml:"password"`
			Branches []string `yaml:"branches"`
			Tags     bool     `yaml:"tags"`
		} `yaml:"git"`
	} `yaml:"watchs"`
	Scripts []string `yaml:"scripts"`
	Logs    struct {
		RetainDays int `yaml:"retain_days"`
	} `yaml:"logs"`
}

func Load(cfgAbs string) (*Config, string, error) {
	b, err := ioutil.ReadFile(cfgAbs)
	if err != nil {
		return nil, "", err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, "", err
	}
	root := filepath.Dir(cfgAbs)
	if c.Logs.RetainDays <= 0 {
		c.Logs.RetainDays = 7
	}
	for i, s := range c.Scripts {
		if !filepath.IsAbs(s) {
			c.Scripts[i] = filepath.Clean(filepath.Join(root, s))
		}
	}
	if len(c.Watchs.Docker.Images) == 0 && !c.Watchs.Git.Tags && len(c.Watchs.Git.Branches) == 0 {
		return nil, "", errors.New("no watchs configured")
	}
	return &c, root, nil
}
