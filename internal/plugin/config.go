package plugin

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPath          = "quota-ledger/samples.jsonl"
	defaultRetentionDays = 35
)

type config struct {
	path      string
	retention time.Duration
}

type configField struct {
	Name        string
	Type        string
	Default     any
	Description string
}

var configFields = []configField{
	{
		Name:        "path",
		Type:        "string",
		Default:     defaultPath,
		Description: "JSONL file that stores quota samples. A relative path resolves against the CPA working directory.",
	},
	{
		Name:        "retention-days",
		Type:        "integer",
		Default:     defaultRetentionDays,
		Description: "Days of samples to keep. Older samples are dropped on load and at most hourly on write.",
	},
}

func parseConfig(raw []byte) (config, error) {
	var values struct {
		Path          *string `yaml:"path"`
		RetentionDays *int    `yaml:"retention-days"`
	}
	if err := yaml.Unmarshal(raw, &values); err != nil {
		return config{}, err
	}

	path := defaultPath
	if values.Path != nil {
		path = strings.TrimSpace(*values.Path)
	}
	if path == "" {
		return config{}, errors.New("path must not be empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return config{}, err
	}

	retentionDays := defaultRetentionDays
	if values.RetentionDays != nil {
		retentionDays = *values.RetentionDays
	}
	if retentionDays < 1 {
		return config{}, errors.New("retention-days must be at least 1")
	}
	return config{path: absolute, retention: time.Duration(retentionDays) * 24 * time.Hour}, nil
}
