package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type InterfaceConfig struct {
	LAN    string `yaml:"lan"`
	IFB    string `yaml:"ifb"`
	Subnet string `yaml:"subnet"`
}

type Profile struct {
	Delay       string `yaml:"delay"`
	Jitter      string `yaml:"jitter,omitempty"`
	Correlation string `yaml:"correlation,omitempty"`
	Loss        string `yaml:"loss,omitempty"`
	Duplicate   string `yaml:"duplicate,omitempty"`
	Reorder     string `yaml:"reorder,omitempty"`
	Corrupt     string `yaml:"corrupt,omitempty"`
	Rate        string `yaml:"rate,omitempty"`
}

type Config struct {
	Interfaces  InterfaceConfig    `yaml:"interfaces"`
	RootRate    string             `yaml:"root_rate"`
	Profiles    map[string]Profile `yaml:"profiles"`
	Assignments map[string]string  `yaml:"assignments"`
	Names       map[string]string  `yaml:"names"` // MAC -> custom name
}

func DefaultConfig() *Config {
	return &Config{
		Interfaces: InterfaceConfig{
			LAN:    "",
			IFB:    "ifb0",
			Subnet: "",
		},
		RootRate: "1gbit",
		Profiles: map[string]Profile{
			"3G": {
				Delay:       "200ms",
				Jitter:      "50ms",
				Correlation: "25%",
				Loss:        "1.5%",
				Rate:        "2mbit",
			},
			"LTE": {
				Delay:       "50ms",
				Jitter:      "10ms",
				Correlation: "25%",
				Loss:        "0.5%",
				Rate:        "50mbit",
			},
			"Lossy-WiFi": {
				Delay:   "15ms",
				Jitter:  "5ms",
				Loss:    "3%",
				Reorder: "1%",
				Rate:    "20mbit",
			},
			"Starlink": {
				Delay:       "40ms",
				Jitter:      "7ms",
				Correlation: "25%",
				Loss:        "1%",
				Rate:        "100mbit",
			},
			"Satellite": {
				Delay:       "600ms",
				Jitter:      "50ms",
				Correlation: "25%",
				Loss:        "1.5%",
				Rate:        "5mbit",
			},
			"DSL": {
				Delay:  "25ms",
				Jitter: "5ms",
				Loss:   "0.2%",
				Rate:   "25mbit",
			},
			"Cable": {
				Delay:  "10ms",
				Jitter: "2ms",
				Loss:   "0.05%",
				Rate:   "200mbit",
			},
		},
		Assignments: make(map[string]string),
		Names:       make(map[string]string),
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Assignments == nil {
		cfg.Assignments = make(map[string]string)
	}
	if cfg.Names == nil {
		cfg.Names = make(map[string]string)
	}
	return cfg, nil
}

func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *Config) ValidateIP(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	parsed = parsed.To4()
	if parsed == nil {
		return fmt.Errorf("not an IPv4 address: %s", ip)
	}
	_, subnet, err := net.ParseCIDR(c.Interfaces.Subnet)
	if err != nil {
		return fmt.Errorf("invalid subnet in config: %s", c.Interfaces.Subnet)
	}
	if !subnet.Contains(parsed) {
		return fmt.Errorf("IP %s is not in subnet %s", ip, c.Interfaces.Subnet)
	}
	return nil
}

func (c *Config) ValidateProfile(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		names := c.ProfileNames()
		return fmt.Errorf("unknown profile %q (available: %v)", name, names)
	}
	return nil
}

func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
