package tutor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalProviderConfigLoopbackAndAtomicPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "tutor-provider.json")
	c := LocalProviderConfig{Endpoint: "http://127.0.0.1:11434/v1/chat/completions", Model: "local-model", AdvancedModel: "local-reasoner"}
	if err := SaveLocalProvider(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLocalProvider(path)
	if err != nil || got != c {
		t.Fatalf("load=%+v err=%v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("file mode=%o", info.Mode().Perm())
	}
	info, err = os.Stat(filepath.Dir(path))
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("dir mode=%v err=%v", info, err)
	}
	for _, endpoint := range []string{"https://api.example/v1/chat/completions", "http://192.0.2.1/v1/chat/completions", "http://user:pass@127.0.0.1/v1/chat/completions", "http://127.0.0.1/v1/chat/completions?token=x"} {
		if ValidateLocalProvider(LocalProviderConfig{Endpoint: endpoint, Model: "m"}) == nil {
			t.Errorf("accepted unsafe endpoint %q", endpoint)
		}
	}
	if err = DeleteLocalProvider(path); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("config was not deleted: %v", err)
	}
	if ValidateLocalProvider(LocalProviderConfig{Model: "missing-endpoint"}) == nil {
		t.Fatal("empty endpoint accepted")
	}
	baseURL, err := NormalizeLocalProvider(LocalProviderConfig{Endpoint: "http://127.0.0.1:11434/v1", Model: "base-url-model"})
	if err != nil || baseURL.Endpoint != "http://127.0.0.1:11434/v1/chat/completions" {
		t.Fatalf("/v1 base URL not normalized: %+v %v", baseURL, err)
	}
	if err = os.WriteFile(path, []byte(`{"endpoint":"http://127.0.0.1:11434/v1","model":"m","api_key":"secret"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadLocalProvider(path); err == nil {
		t.Fatal("unknown key field in persisted config accepted")
	}
}

func TestCloudProviderConfigRequiresSafeHTTPSAndProtectsKeyFile(t *testing.T) {
	valid := CloudProviderConfig{Endpoint: "https://8.8.8.8/v1", APIKey: "secret-key", Model: "cloud-model"}
	c, err := NormalizeCloudProvider(valid)
	if err != nil || c.Endpoint != "https://8.8.8.8/v1/chat/completions" {
		t.Fatalf("normalize=%+v err=%v", c, err)
	}
	for _, bad := range []CloudProviderConfig{
		{Endpoint: "http://8.8.8.8/v1", APIKey: "k", Model: "m"},
		{Endpoint: "https://127.0.0.1/v1", APIKey: "k", Model: "m"},
		{Endpoint: "https://10.1.2.3/v1", APIKey: "k", Model: "m"},
		{Endpoint: "https://192.0.2.1/v1", APIKey: "k", Model: "m"},
		{Endpoint: "https://user:pass@8.8.8.8/v1", APIKey: "k", Model: "m"},
		{Endpoint: "https://8.8.8.8/v1?key=x", APIKey: "k", Model: "m"},
		{Endpoint: "https://8.8.8.8/v1", Model: "m"},
		{Endpoint: "https://8.8.8.8/v1", APIKey: "k"},
		{Endpoint: "https://8.8.8.8/v1", APIKey: strings.Repeat("k", 4097), Model: "m"},
	} {
		if _, err := NormalizeCloudProvider(bad); err == nil {
			t.Errorf("accepted unsafe cloud config %+v", bad)
		}
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "cloud.json")
	if err := SaveCloudProvider(path, valid); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v err=%v", info, err)
	}
	loaded, err := LoadCloudProvider(path)
	if err != nil || loaded != c {
		t.Fatalf("load=%+v err=%v", loaded, err)
	}
	if err = os.WriteFile(path, []byte(`{"endpoint":"https://8.8.8.8/v1","api_key":"k","model":"m","unexpected":"x"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadCloudProvider(path); err == nil {
		t.Fatal("unknown config field accepted")
	}
}

func TestDeleteProviderWithMissingParentIsNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-created", "cloud.json")
	if err := DeleteCloudProvider(path); err != nil {
		t.Fatalf("deleting a non-existent provider path should be a no-op: %v", err)
	}
}
