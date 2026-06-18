package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AuthProfile stores reusable authentication settings scoped to a host pattern.
type AuthProfile struct {
	Name        string            `yaml:"name"`
	Scope       string            `yaml:"scope"`
	BearerToken string            `yaml:"bearer_token"`
	BasicAuth   string            `yaml:"basic_auth"`
	Cookies     []string          `yaml:"cookies"`
	Headers     map[string]string `yaml:"headers"`
}

// profileDir returns the directory where profiles are stored.
func profileDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "webhound", "profiles")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// SaveProfile persists an AuthProfile to disk.
func SaveProfile(p *AuthProfile) error {
	dir, err := profileDir()
	if err != nil {
		return fmt.Errorf("resolving profile dir: %w", err)
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshalling profile: %w", err)
	}
	path := filepath.Join(dir, p.Name+".yaml")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing profile: %w", err)
	}
	return nil
}

// LoadProfile loads a named AuthProfile from disk.
func LoadProfile(name string) (*AuthProfile, error) {
	dir, err := profileDir()
	if err != nil {
		return nil, fmt.Errorf("resolving profile dir: %w", err)
	}
	path := filepath.Join(dir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading profile %q: %w", name, err)
	}
	var p AuthProfile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing profile %q: %w", name, err)
	}
	return &p, nil
}

// ListProfiles returns names of all stored profiles.
func ListProfiles() ([]string, error) {
	dir, err := profileDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			names = append(names, e.Name()[:len(e.Name())-5])
		}
	}
	return names, nil
}

// DeleteProfile removes a stored profile.
func DeleteProfile(name string) error {
	dir, err := profileDir()
	if err != nil {
		return err
	}
	return os.Remove(filepath.Join(dir, name+".yaml"))
}

// ApplyProfile merges a profile into a ScanConfig.
func ApplyProfile(cfg *ScanConfig, p *AuthProfile) {
	if p.BearerToken != "" {
		cfg.BearerToken = p.BearerToken
	}
	if p.BasicAuth != "" {
		cfg.BasicAuth = p.BasicAuth
	}
	if len(p.Cookies) > 0 {
		cfg.Cookies = append(cfg.Cookies, p.Cookies...)
	}
	for k, v := range p.Headers {
		cfg.Headers = append(cfg.Headers, k+": "+v)
	}
}
