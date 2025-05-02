package config

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

// LLMConfig 存储大语言模型相关配置
type LLMConfig struct {
	Model  string `toml:"model"`
	APIKey string `toml:"api_key"`
	APIURL string `toml:"api_url"`
}

// ClientConfig 存储应用程序客户端配置
type ClientConfig struct {
	MaxConcurrentRequests int    `toml:"max_concurrent_requests"`
	Prompt                string `toml:"prompt"`
	AutoDetectCJK         bool   `toml:"auto_detect_cjk"`
}

// Config 存储应用配置
type Config struct {
	LLM    LLMConfig    `toml:"llm"`
	Client ClientConfig `toml:"client"`
}

// 默认配置值
const (
	DefaultAPIURL                = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	DefaultAPIKEY                = "sk-xxxx"
	DefaultOpenaiModel           = "qwen-turbo-latest"
	DefaultMaxConcurrentRequests = 5
	DefaultPrompt                = "You are a professional translator. Translate Japanese to Simplified Chinese directly. Keep all alphanumeric characters unchanged. Ensure accuracy of technical terms. No explanations needed."
	DefaultAutoDetectCJK         = true
)

// 配置文件名
const (
	configFileName = "config.toml"
	appDirName     = "Exceltranslator"
)

// GetConfigDir 返回配置目录路径
func GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("无法获取用户配置目录: %w", err)
	}

	appConfigDir := filepath.Join(configDir, appDirName)
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("无法创建应用配置目录: %w", err)
	}

	return appConfigDir, nil
}

// LoadConfig 从配置文件加载配置
func LoadConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, configFileName)

	// 如果配置文件不存在，创建默认配置并保存
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &Config{
			Client: ClientConfig{
				MaxConcurrentRequests: DefaultMaxConcurrentRequests,
				Prompt:                DefaultPrompt,
				AutoDetectCJK:         DefaultAutoDetectCJK,
			},
			LLM: LLMConfig{
				Model:  DefaultOpenaiModel,
				APIKey: DefaultAPIKEY,
				APIURL: DefaultAPIURL,
			},
		}

		// 保存默认配置
		if err := SaveConfig(defaultConfig); err != nil {
			return nil, fmt.Errorf("保存默认配置失败: %w", err)
		}

		return defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, configFileName)

	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("编码配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}
