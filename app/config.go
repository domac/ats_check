package app

import (
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/domac/ats_check/util"
	"path/filepath"
)

type AppConfig struct {
	Parents                  []string
	Haproxys                 []string
	Parents_config_path      string
	Remap_config_path        string
	Records_config_path      string
	Health_check             string
	Retry                    int
	Retry_sleep_ms           int
	Check_duration_second    int
	filepath                 string
	Setup_records_config_cmd string
	Setup_parent_config_cmd  string
	Is_parent                int
}

//载入配置文件
func LoadConfig(fp string) (*AppConfig, error) {
	if fp == "" {
		return nil, errors.New("the config file dir is empty")
	}
	if err := util.CheckDataFileExist(fp); err != nil {
		return nil, err
	}
	var cfg *AppConfig
	if fp != "" {
		_, err := toml.DecodeFile(fp, &cfg)
		if err != nil {
			return nil, err
		}
	}
	cp, _ := filepath.Abs(fp)
	cfg.filepath = cp
	return cfg, nil
}
