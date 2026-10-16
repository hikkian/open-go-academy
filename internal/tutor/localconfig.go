package tutor

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// LocalProviderConfig deliberately has no API-key field: configurable providers
// are restricted to unauthenticated loopback HTTP endpoints.
type LocalProviderConfig struct {
	Endpoint      string `json:"endpoint"`
	Model         string `json:"model"`
	AdvancedModel string `json:"advanced_model,omitempty"`
}

// CloudProviderConfig contains the server-only credential for an explicitly
// selected HTTPS OpenAI-compatible provider. Never serialize it into API output.
type CloudProviderConfig struct {
	Endpoint      string `json:"endpoint"`
	APIKey        string `json:"api_key"`
	Model         string `json:"model"`
	AdvancedModel string `json:"advanced_model,omitempty"`
}

func NormalizeCloudProvider(c CloudProviderConfig) (CloudProviderConfig, error) {
	if strings.TrimSpace(c.Endpoint) == "" || len(c.Endpoint) > 2048 || len(c.APIKey) == 0 || len(c.APIKey) > 4096 || strings.TrimSpace(c.APIKey) != c.APIKey || len(c.Model) == 0 || len(c.Model) > 128 || len(c.AdvancedModel) > 128 || strings.TrimSpace(c.Model) != c.Model || strings.TrimSpace(c.AdvancedModel) != c.AdvancedModel || strings.ContainsAny(c.APIKey+c.Model+c.AdvancedModel, "\r\n\x00") {
		return CloudProviderConfig{}, errors.New("invalid cloud tutor provider fields")
	}
	u, err := url.Parse(strings.TrimSpace(c.Endpoint))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return CloudProviderConfig{}, errors.New("cloud tutor requires an HTTPS endpoint without credentials, query, or fragment")
	}
	if u.Path == "" || u.Path == "/" || u.Path == "/v1" || u.Path == "/v1/" {
		u.Path = "/v1/chat/completions"
		c.Endpoint = u.String()
	}
	if _, err = New(Config{Endpoint: c.Endpoint, APIKey: c.APIKey, EconomyModel: c.Model, AdvancedModel: c.AdvancedModel, AllowCloud: true, MaxOutputTokens: 512}); err != nil {
		return CloudProviderConfig{}, err
	}
	return c, nil
}

func LoadCloudProvider(path string) (CloudProviderConfig, error) {
	var c CloudProviderConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if len(b) > 8192 {
		return c, errors.New("cloud tutor config exceeds size limit")
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return CloudProviderConfig{}, errors.New("invalid trailing cloud tutor config data")
	}
	return NormalizeCloudProvider(c)
}

func SaveCloudProvider(path string, c CloudProviderConfig) error {
	var err error
	if c, err = NormalizeCloudProvider(c); err != nil {
		return err
	}
	return saveProviderJSON(path, c)
}

// RestoreValidatedCloudProvider atomically restores a value loaded and validated
// by LoadCloudProvider. It deliberately avoids DNS/network revalidation while
// rolling back a failed local provider transition.
func RestoreValidatedCloudProvider(path string, c CloudProviderConfig) error {
	return saveProviderJSON(path, c)
}

func DeleteCloudProvider(path string) error { return deleteProvider(path) }

func ValidateLocalProvider(c LocalProviderConfig) error {
	if strings.TrimSpace(c.Endpoint) == "" || len(c.Endpoint) > 2048 || len(c.Model) == 0 || len(c.Model) > 128 || len(c.AdvancedModel) > 128 || strings.TrimSpace(c.Model) != c.Model || strings.TrimSpace(c.AdvancedModel) != c.AdvancedModel {
		return errors.New("invalid local tutor provider fields")
	}
	if strings.ContainsAny(c.Model+c.AdvancedModel, "\r\n\x00") {
		return errors.New("invalid model name")
	}
	g, err := New(Config{Endpoint: c.Endpoint, EconomyModel: c.Model, AdvancedModel: c.AdvancedModel, MaxOutputTokens: 512})
	if err != nil {
		return err
	}
	if !g.Enabled() {
		return errors.New("local tutor endpoint is required")
	}
	return nil
}

// NormalizeLocalProvider completes the common OpenAI-compatible `/v1` base URL.
func NormalizeLocalProvider(c LocalProviderConfig) (LocalProviderConfig, error) {
	u, err := url.Parse(strings.TrimSpace(c.Endpoint))
	if err != nil {
		return LocalProviderConfig{}, err
	}
	if u.Path == "/v1" || u.Path == "/v1/" {
		u.Path = "/v1/chat/completions"
		c.Endpoint = u.String()
	}
	if err = ValidateLocalProvider(c); err != nil {
		return LocalProviderConfig{}, err
	}
	return c, nil
}

func LoadLocalProvider(path string) (LocalProviderConfig, error) {
	var c LocalProviderConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if len(b) > 4096 {
		return c, errors.New("local tutor config exceeds size limit")
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return LocalProviderConfig{}, errors.New("invalid trailing local tutor config data")
	}
	if c, err = NormalizeLocalProvider(c); err != nil {
		return LocalProviderConfig{}, err
	}
	return c, nil
}

func SaveLocalProvider(path string, c LocalProviderConfig) error {
	var err error
	if c, err = NormalizeLocalProvider(c); err != nil {
		return err
	}
	return saveProviderJSON(path, c)
}

// RestoreValidatedLocalProvider atomically restores a value loaded and validated
// by LoadLocalProvider during a failed provider switch.
func RestoreValidatedLocalProvider(path string, c LocalProviderConfig) error {
	return saveProviderJSON(path, c)
}

func saveProviderJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tutor-provider-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(b)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpName, path); err != nil {
		return err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func DeleteLocalProvider(path string) error {
	return deleteProvider(path)
}

func deleteProvider(path string) error {
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if _, statErr := os.Stat(filepath.Dir(path)); os.IsNotExist(statErr) {
			return nil
		} else if statErr != nil {
			return statErr
		}
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
