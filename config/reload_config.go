package config

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// CachedConfig хранит конфигурацию и её хеши
type CachedConfig struct {
	Config      *Config
	ConfigHash  string
	SecretsHash string
}

// LoadConfigWithHash загружает конфиг и считает хеши
func LoadConfigWithHash(path string) (*CachedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	cfg, err := GetConfig(path)
	if err != nil {
		return nil, err
	}

	secretsHash := ""
	if cfg.SecretsPath != "" {
		if sData, err := os.ReadFile(cfg.SecretsPath); err == nil {
			secretsHash = fmt.Sprintf("%x", sha256.Sum256(sData))
		}
	}

	return &CachedConfig{
		Config:      cfg,
		ConfigHash:  hash,
		SecretsHash: secretsHash,
	}, nil
}

// ReloadIfChanged перечитывает конфиг и секреты, если хеш изменился
func (c *CachedConfig) ReloadIfChanged(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot read config file: %w", err)
	}

	newHash := fmt.Sprintf("%x", sha256.Sum256(data))
	cfg, err := GetConfig(path)
	if err != nil {
		return false, err
	}

	changed := false

	if newHash != c.ConfigHash {
		c.Config = cfg
		c.ConfigHash = newHash
		changed = true
	}

	// Проверяем secrets
	if cfg.SecretsPath != "" {
		sData, err := os.ReadFile(cfg.SecretsPath)
		if err != nil {
			return changed, fmt.Errorf("cannot read secrets file: %w", err)
		}
		newSecretsHash := fmt.Sprintf("%x", sha256.Sum256(sData))
		if newSecretsHash != c.SecretsHash {
			// Перечитываем секреты
			if err := c.Config.LoadSecrets(); err != nil {
				return changed, err
			}
			c.SecretsHash = newSecretsHash
			changed = true
		}
	}

	return changed, nil
}
