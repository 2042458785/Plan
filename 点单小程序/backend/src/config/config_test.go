package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYAMLConfigAndLocalOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	raw, e := os.ReadFile("config.example.yaml")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	local := `mysql:
  dbname: suxu_test_configuration
jwt:
  secret: test-secret-32-characters-long-configuration
server:
  port: "19090"
`
	if e = os.WriteFile(filepath.Join(dir, "config.local.yaml"), []byte(local), 0600); e != nil {
		t.Fatal(e)
	}
	cfg, e := LoadConfig(path)
	if e != nil {
		t.Fatal(e)
	}
	if cfg.Server.Port != "19090" || cfg.MySQL.DBName != "suxu_test_configuration" || cfg.MySQL.Host != "127.0.0.1" {
		t.Fatal("YAML 覆盖未合并主配置")
	}
	bad := local + "\n  whitelist: [\"/api/v1/admin\"]\n"
	bad = strings.Replace(bad, "jwt:\n", "jwt:\n  whitelist: [\"/api/v1/admin\"]\n", 1)
	bad = strings.TrimSuffix(bad, "\n  whitelist: [\"/api/v1/admin\"]\n")
	if e = os.WriteFile(filepath.Join(dir, "config.local.yaml"), []byte(bad), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = LoadConfig(path); e == nil {
		t.Fatal("管理组不应能加入认证白名单")
	}
}
func TestMissingConfigFails(t *testing.T) {
	if _, e := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml")); e == nil {
		t.Fatal("配置缺失未报错")
	}
}
