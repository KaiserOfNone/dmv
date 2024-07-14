package main

import (
	"io"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct{}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	doc, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = toml.Unmarshal(doc, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
