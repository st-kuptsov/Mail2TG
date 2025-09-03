package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
)

// Config хранит основную конфигурацию приложения
type Config struct {
	IMAP          IMAPConfig     `yaml:"imap"`
	Telegram      TelegramConfig `yaml:"telegram"`
	Route         []RouteConfig  `yaml:"route"`
	Logging       LogConfig      `yaml:"log_settings"`
	CheckInterval int            `yaml:"check_interval"`
	SecretsPath   string         `yaml:"secrets"`
	ServicePort   int            `yaml:"service_port" env-default:"9090"`
}

type IMAPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string
}

type TelegramConfig struct {
	Token          string `yaml:"token"`
	DefaultChannel string `yaml:"default_channel"`
	ErrorsChannel  string `yaml:"errors_channel"`
}

type RouteConfig struct {
	Folders []Folder `yaml:"folders"`
}

type Folder struct {
	Name  string `yaml:"name"`
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	Pattern string `yaml:"pattern"`
	Channel string `yaml:"channel"`
}

// LogConfig для логирования
type LogConfig struct {
	Directory  string `yaml:"directory" env-default:"logs"`
	Filename   string `yaml:"filename" env-default:"app.log"`
	MaxSize    int    `yaml:"max_size" env-default:"10"`
	MaxBackups int    `yaml:"max_backups" env-default:"1"`
	MaxAge     int    `yaml:"max_age" env-default:"1"`
	Compress   bool   `yaml:"compress" env-default:"true"`
	Level      string `yaml:"level" env-default:"info"`
	Console    bool   `yaml:"console_enabled" env-default:"false"`
}

// GetConfig загружает конфигурацию из файла, возвращает указатель и ошибку
func GetConfig(configPath string) (*Config, error) {
	log.Println("Чтение конфигурации...")

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		// Передаем указатель в GetDescription
		if help, err2 := cleanenv.GetDescription(&cfg, nil); err2 == nil {
			log.Println(help)
		}
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Загружаем секреты, если указан путь
	if err := cfg.LoadSecrets(); err != nil {
		return nil, err
	}

	// Валидация (если есть метод Validate)
	if validator, ok := interface{}(&cfg).(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}

// LoadSecrets загружает пароли и токены из отдельного файла
func (c *Config) LoadSecrets() error {
	if c.SecretsPath == "" {
		return nil
	}

	type Secrets struct {
		IMAP struct {
			Password string `yaml:"password"`
		} `yaml:"imap"`
		Telegram struct {
			Token string `yaml:"token"`
		} `yaml:"telegram"`
	}

	var sec Secrets
	if err := cleanenv.ReadConfig(c.SecretsPath, &sec); err != nil {
		return err
	}

	c.IMAP.Password = sec.IMAP.Password
	c.Telegram.Token = sec.Telegram.Token

	return nil
}
