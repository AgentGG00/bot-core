package sourceloader

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Scripts struct {
	Backup       string `yaml:"backup"`
	Restore      string `yaml:"restore"`
	Update       string `yaml:"update"`
	CheckVersion string `yaml:"check_version"`
}

type Target struct {
	Name           string  `yaml:"name"`
	GithubRepo     string  `yaml:"github_repo"`
	ComposeService string  `yaml:"compose_service"`
	Scripts        Scripts `yaml:"scripts"`
}

type LockWindow struct {
	Name                string `yaml:"name"`
	Day                 string `yaml:"day"`
	Start               string `yaml:"start"`
	End                 string `yaml:"end"`
	ConditionScript     string `yaml:"condition_script"`
	BufferBeforeMinutes int    `yaml:"buffer_before_minutes"`
	BufferAfterMinutes  int    `yaml:"buffer_after_minutes"`
}

type Config struct {
	Telegram struct {
		BotTokenEnv string `yaml:"bot_token_env"`
		ChatIDEnv   string `yaml:"chat_id_env"`
	} `yaml:"telegram"`
	Totp struct {
		SecretEnv string `yaml:"secret_env"`
	} `yaml:"totp"`
	LLM struct {
		OllamaHostEnv string `yaml:"ollama_host_env"`
		Model         string `yaml:"model"`
	} `yaml:"llm"`
	Targets           []Target     `yaml:"targets"`
	LockWindows       []LockWindow `yaml:"lock_windows"`
	PollIntervalHours int          `yaml:"poll_interval_hours"`
}

type Source struct {
	Dir     string
	Config  Config
	Env     map[string]string
	Prompt  string
	Context string
}

func Load(dir string) (*Source, error) {
	cfgBytes, err := os.ReadFile(filepath.Join(dir, "config", "source.yaml"))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
		return nil, err
	}

	env, err := loadEnv(filepath.Join(dir, ".env"))
	if err != nil {
		return nil, err
	}

	promptBytes, err := os.ReadFile(filepath.Join(dir, "config", "prompt.md"))
	if err != nil {
		return nil, err
	}

	contextBytes, err := os.ReadFile(filepath.Join(dir, "config", "context.md"))
	if err != nil {
		return nil, err
	}

	return &Source{
		Dir:     dir,
		Config:  cfg,
		Env:     env,
		Prompt:  string(promptBytes),
		Context: string(contextBytes),
	}, nil
}

func loadEnv(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		env[key] = value
	}
	return env, scanner.Err()
}
