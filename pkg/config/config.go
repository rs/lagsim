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

// DirectionalProfile holds netem parameters for a single direction.
type DirectionalProfile struct {
	Delay        string `yaml:"delay,omitempty"`
	Jitter       string `yaml:"jitter,omitempty"`
	Correlation  string `yaml:"correlation,omitempty"`
	Distribution string `yaml:"distribution,omitempty"` // normal, pareto, paretonormal
	Loss         string `yaml:"loss,omitempty"`
	ECN          bool   `yaml:"ecn,omitempty"` // mark with ECN CE bit instead of dropping
	Duplicate    string `yaml:"duplicate,omitempty"`
	Reorder      string `yaml:"reorder,omitempty"` // e.g. "25% gap 5" or just "1%"
	Corrupt      string `yaml:"corrupt,omitempty"`
	Rate         string `yaml:"rate,omitempty"`
	Slot         string `yaml:"slot,omitempty"` // e.g. "20ms 5ms" (min delay, optional jitter)
}

// Profile defines network conditions. Base fields apply to both directions.
// Optional Download/Upload overrides replace specific params per direction.
type Profile struct {
	DirectionalProfile `yaml:",inline"`
	Download           *DirectionalProfile `yaml:"download,omitempty"`
	Upload             *DirectionalProfile `yaml:"upload,omitempty"`
}

// Resolved returns the effective parameters for a direction ("download" or "upload")
// by overlaying directional overrides on top of the base profile.
func (p Profile) Resolved(direction string) DirectionalProfile {
	var override *DirectionalProfile
	if direction == "download" {
		override = p.Download
	} else {
		override = p.Upload
	}
	if override == nil {
		return p.DirectionalProfile
	}
	base := p.DirectionalProfile
	if override.Delay != "" {
		base.Delay = override.Delay
	}
	if override.Jitter != "" {
		base.Jitter = override.Jitter
	}
	if override.Correlation != "" {
		base.Correlation = override.Correlation
	}
	if override.Distribution != "" {
		base.Distribution = override.Distribution
	}
	if override.Loss != "" {
		base.Loss = override.Loss
	}
	if override.ECN {
		base.ECN = true
	}
	if override.Duplicate != "" {
		base.Duplicate = override.Duplicate
	}
	if override.Reorder != "" {
		base.Reorder = override.Reorder
	}
	if override.Corrupt != "" {
		base.Corrupt = override.Corrupt
	}
	if override.Rate != "" {
		base.Rate = override.Rate
	}
	if override.Slot != "" {
		base.Slot = override.Slot
	}
	return base
}

// IsAsymmetric returns true if the profile has directional overrides.
func (p Profile) IsAsymmetric() bool {
	return p.Download != nil || p.Upload != nil
}

type Config struct {
	Interfaces  InterfaceConfig    `yaml:"interfaces"`
	RootRate    string             `yaml:"root_rate"`
	Profiles    map[string]*Profile `yaml:"profiles"`
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
		Profiles: map[string]*Profile{
			"3G": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "100ms",
					Jitter:       "30ms",
					Correlation:  "25%",
					Distribution: "paretonormal",
					Loss:         "1.5%",
					Rate:         "2mbit",
					Slot:         "40ms 10ms",
				},
				Upload: &DirectionalProfile{Rate: "0.5mbit", Jitter: "50ms", Loss: "2.5%"},
			},
			"LTE": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "20ms",
					Jitter:       "5ms",
					Correlation:  "25%",
					Distribution: "paretonormal",
					Loss:         "0.5%",
					Rate:         "50mbit",
					Slot:         "10ms 3ms",
				},
				Upload: &DirectionalProfile{Rate: "15mbit", Jitter: "8ms", Loss: "1%"},
			},
			"5G": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "5ms",
					Jitter:       "1ms",
					Correlation:  "25%",
					Distribution: "paretonormal",
					Loss:         "0.05%",
					Rate:         "300mbit",
				},
				Upload: &DirectionalProfile{Rate: "100mbit", Loss: "0.1%"},
			},
			"Edge-2G": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "150ms",
					Jitter:       "60ms",
					Correlation:  "25%",
					Distribution: "paretonormal",
					Loss:         "5%",
					Rate:         "0.1mbit",
					Slot:         "80ms 20ms",
				},
				Upload: &DirectionalProfile{Rate: "0.05mbit", Jitter: "100ms", Loss: "8%"},
			},
			"Lossy-WiFi": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "5ms",
					Jitter:       "3ms",
					Correlation:  "25%",
					Distribution: "pareto",
					Loss:         "3%",
					Reorder:      "1% gap 5",
					Rate:         "20mbit",
					Slot:         "5ms 2ms",
				},
			},
			"Starlink": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "20ms",
					Jitter:       "5ms",
					Correlation:  "25%",
					Distribution: "normal",
					Loss:         "0.5%",
					Reorder:      "0.5%",
					Rate:         "100mbit",
				},
				Upload: &DirectionalProfile{Rate: "20mbit", Jitter: "10ms", Loss: "1%"},
			},
			"Satellite": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "300ms",
					Jitter:       "30ms",
					Correlation:  "25%",
					Distribution: "normal",
					Loss:         "1.5%",
					Rate:         "5mbit",
				},
				Upload: &DirectionalProfile{Rate: "1mbit", Jitter: "50ms", Loss: "2.5%"},
			},
			"DSL": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "15ms",
					Jitter:       "3ms",
					Distribution: "normal",
					Loss:         "0.2%",
					Rate:         "25mbit",
				},
				Upload: &DirectionalProfile{Rate: "3mbit"},
			},
			"Cable": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "5ms",
					Jitter:       "1ms",
					Distribution: "normal",
					Loss:         "0.05%",
					Rate:         "200mbit",
				},
				Upload: &DirectionalProfile{Rate: "20mbit"},
			},
			"Airplane-WiFi": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "150ms",
					Jitter:       "30ms",
					Correlation:  "25%",
					Distribution: "pareto",
					Loss:         "3%",
					Reorder:      "1% gap 5",
					Rate:         "2mbit",
					Slot:         "30ms 10ms",
				},
				Upload: &DirectionalProfile{Rate: "1mbit", Loss: "5%", Jitter: "50ms"},
			},
			"Congested": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "50ms",
					Jitter:       "40ms",
					Correlation:  "50%",
					Distribution: "paretonormal",
					Loss:         "5%",
					Reorder:      "2% gap 3",
					Rate:         "1mbit",
				},
				Upload: &DirectionalProfile{Rate: "0.5mbit"},
			},
			"Bursty": {
				DirectionalProfile: DirectionalProfile{
					Delay:  "10ms",
					Jitter: "2ms",
					Loss:   "gemodel 0.5% 15% 100% 0%",
					Rate:   "50mbit",
				},
			},
			"ECN-Datacenter": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "1ms",
					Jitter:       "0.5ms",
					Distribution: "normal",
					Loss:         "2%",
					ECN:          true,
					Rate:         "1gbit",
				},
			},
			"ECN-WAN": {
				DirectionalProfile: DirectionalProfile{
					Delay:        "25ms",
					Jitter:       "5ms",
					Correlation:  "25%",
					Distribution: "normal",
					Loss:         "0.5%",
					ECN:          true,
					Rate:         "100mbit",
				},
				Upload: &DirectionalProfile{Rate: "50mbit"},
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
	// Remove profiles set to null in YAML (allows disabling built-ins).
	for name, p := range cfg.Profiles {
		if p == nil {
			delete(cfg.Profiles, name)
		}
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
	// Don't persist built-in profiles — only save user overrides/additions.
	defaults := DefaultConfig()
	saved := cfg.Profiles
	filtered := make(map[string]*Profile, len(saved))
	for name, p := range saved {
		if dp, ok := defaults.Profiles[name]; ok && profileEqual(p, dp) {
			continue // skip unchanged built-in
		}
		filtered[name] = p
	}
	cfg.Profiles = filtered
	data, err := yaml.Marshal(cfg)
	cfg.Profiles = saved // restore full set in memory
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

func profileEqual(a, b *Profile) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.DirectionalProfile == b.DirectionalProfile &&
		dirPtrEqual(a.Download, b.Download) &&
		dirPtrEqual(a.Upload, b.Upload)
}

func dirPtrEqual(a, b *DirectionalProfile) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
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
