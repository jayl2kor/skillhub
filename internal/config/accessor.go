package config

import (
	"fmt"
	"strconv"
	"strings"
)

// KeyValue represents a config key-value pair.
type KeyValue struct {
	Key   string
	Value string
}

var validRegistryFields = map[string]bool{
	"name":          true,
	"url":           true,
	"token":         true,
	"username":      true,
	"branch":        true,
	"skills_prefix": true,
}

// GetValue retrieves a config value by dot-notation key.
func (c *Config) GetValue(key string) (string, error) {
	parts := strings.SplitN(key, ".", 3)

	switch parts[0] {
	case "install_dir":
		if len(parts) != 1 {
			return "", fmt.Errorf("unknown config key %q", key)
		}
		return c.InstallDir, nil
	case "cache_dir":
		if len(parts) != 1 {
			return "", fmt.Errorf("unknown config key %q", key)
		}
		return c.CacheDir, nil
	case "log_dir":
		if len(parts) != 1 {
			return "", fmt.Errorf("unknown config key %q", key)
		}
		return c.LogDir, nil
	case "registries":
		if len(parts) < 3 {
			return "", fmt.Errorf("specify a full key: registries.<name>.<field>")
		}
		idx, err := c.resolveRegistryIndex(parts[1])
		if err != nil {
			return "", err
		}
		return c.getRegistryField(idx, parts[2])
	default:
		return "", fmt.Errorf("unknown config key %q", parts[0])
	}
}

// SetValue sets a config value by dot-notation key.
func (c *Config) SetValue(key, value string) error {
	parts := strings.SplitN(key, ".", 3)

	switch parts[0] {
	case "install_dir":
		if len(parts) != 1 {
			return fmt.Errorf("unknown config key %q", key)
		}
		c.InstallDir = value
		return nil
	case "cache_dir":
		if len(parts) != 1 {
			return fmt.Errorf("unknown config key %q", key)
		}
		c.CacheDir = value
		return nil
	case "log_dir":
		if len(parts) != 1 {
			return fmt.Errorf("unknown config key %q", key)
		}
		c.LogDir = value
		return nil
	case "registries":
		if len(parts) < 3 {
			return fmt.Errorf("specify a full key: registries.<name>.<field>")
		}
		idx, err := c.resolveRegistryIndex(parts[1])
		if err != nil {
			return err
		}
		return c.setRegistryField(idx, parts[2], value)
	default:
		return fmt.Errorf("unknown config key %q", parts[0])
	}
}

// ListValues returns all config key-value pairs in a deterministic order.
func (c *Config) ListValues() []KeyValue {
	kvs := []KeyValue{
		{Key: "install_dir", Value: c.InstallDir},
		{Key: "cache_dir", Value: c.CacheDir},
		{Key: "log_dir", Value: c.LogDir},
	}
	for _, r := range c.Registries {
		prefix := "registries." + r.Name
		kvs = append(kvs,
			KeyValue{Key: prefix + ".name", Value: r.Name},
			KeyValue{Key: prefix + ".url", Value: r.URL},
			KeyValue{Key: prefix + ".token", Value: r.Token},
			KeyValue{Key: prefix + ".username", Value: r.Username},
			KeyValue{Key: prefix + ".branch", Value: r.Branch},
			KeyValue{Key: prefix + ".skills_prefix", Value: r.SkillsPrefix},
		)
	}
	return kvs
}

func (c *Config) resolveRegistryIndex(segment string) (int, error) {
	// Try name-based lookup first
	for i, r := range c.Registries {
		if r.Name == segment {
			return i, nil
		}
	}
	// Fallback to numeric index
	idx, err := strconv.Atoi(segment)
	if err != nil {
		return -1, fmt.Errorf("registry %q not found", segment)
	}
	if idx < 0 || idx >= len(c.Registries) {
		return -1, fmt.Errorf("registry index %d out of range (have %d)", idx, len(c.Registries))
	}
	return idx, nil
}

func (c *Config) getRegistryField(idx int, field string) (string, error) {
	if !validRegistryFields[field] {
		return "", fmt.Errorf("unknown registry field %q", field)
	}
	r := c.Registries[idx]
	switch field {
	case "name":
		return r.Name, nil
	case "url":
		return r.URL, nil
	case "token":
		return r.Token, nil
	case "username":
		return r.Username, nil
	case "branch":
		return r.Branch, nil
	case "skills_prefix":
		return r.SkillsPrefix, nil
	}
	return "", fmt.Errorf("unknown registry field %q", field)
}

func (c *Config) setRegistryField(idx int, field, value string) error {
	if !validRegistryFields[field] {
		return fmt.Errorf("unknown registry field %q", field)
	}
	switch field {
	case "name":
		c.Registries[idx].Name = value
	case "url":
		c.Registries[idx].URL = value
	case "token":
		c.Registries[idx].Token = value
	case "username":
		c.Registries[idx].Username = value
	case "branch":
		c.Registries[idx].Branch = value
	case "skills_prefix":
		c.Registries[idx].SkillsPrefix = value
	}
	return nil
}
