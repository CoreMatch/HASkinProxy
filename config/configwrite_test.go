package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const rewriteFixture = `# HASkinProxy deployment config
server:
    listen_addr: ":2702"
    public_url: http://localhost:2702

upstream:
    base_url: http://hrpauth:2778
    timeout: 10
    client_id: hrpauth-internal-super
    client_secret: super-secret

presence:
    enabled: true
    name: HASkinProxy
`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestRewriteClientCredentialsReplacesUpstreamPair(t *testing.T) {
	path := writeFixture(t, rewriteFixture)

	n, err := RewriteClientCredentials(path, "hrpauth-internal-super", "super-secret", "HASkinProxy", "low-secret", []string{"upstream"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("sections replaced = %d, want 1 (upstream)", n)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten: %v", err)
	}
	text := string(out)
	if strings.Contains(text, "super-secret") {
		t.Error("rewritten config still contains the super secret")
	}
	if !strings.Contains(text, "client_id: HASkinProxy") || !strings.Contains(text, "client_secret: low-secret") {
		t.Errorf("upstream section must carry the new pair, got:\n%s", text)
	}
	if !strings.Contains(text, "# HASkinProxy deployment config") {
		t.Error("comments must survive the rewrite")
	}
	if !strings.Contains(text, "base_url: http://hrpauth:2778") || !strings.Contains(text, "name: HASkinProxy") {
		t.Error("unrelated fields must survive the rewrite")
	}

	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(bak) != rewriteFixture {
		t.Error(".bak must contain the pre-change content")
	}

	for _, p := range []string{path, path + ".bak"} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("%s mode = %v, want 0600", p, info.Mode().Perm())
		}
	}

	// The rewritten file must load through the regular LoadConfig path.
	if err := LoadConfig(path); err != nil {
		t.Fatalf("LoadConfig on rewritten file: %v", err)
	}
	if AppConfig.Upstream.ClientID != "HASkinProxy" || AppConfig.Upstream.ClientSecret != "low-secret" {
		t.Errorf("upstream = %q/%q, want the new pair", AppConfig.Upstream.ClientID, AppConfig.Upstream.ClientSecret)
	}
}

func TestRewriteClientCredentialsNoMatchIsNoop(t *testing.T) {
	path := writeFixture(t, rewriteFixture)
	before, _ := os.ReadFile(path)

	n, err := RewriteClientCredentials(path, "nobody", "nothing", "x", "y", []string{"upstream"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("sections replaced = %d, want 0", n)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("no-match rewrite must not touch the file")
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Error("no-match rewrite must not create a backup")
	}
}

func TestRewriteClientCredentialsLeavesDifferentPair(t *testing.T) {
	path := writeFixture(t, strings.Replace(rewriteFixture,
		"client_id: hrpauth-internal-super\n    client_secret: super-secret",
		"client_id: other-client\n    client_secret: other-secret", 1))

	n, err := RewriteClientCredentials(path, "hrpauth-internal-super", "super-secret", "x", "y", []string{"upstream"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("sections replaced = %d, want 0 (pair differs)", n)
	}
	out, _ := os.ReadFile(path)
	if !strings.Contains(string(out), "other-secret") {
		t.Error("a section with different credentials must stay untouched")
	}
}

func TestRewriteClientCredentialsRejectsEmptyValues(t *testing.T) {
	path := writeFixture(t, rewriteFixture)
	if _, err := RewriteClientCredentials(path, "", "", "a", "b", []string{"upstream"}); err == nil {
		t.Fatal("expected an error for empty old credentials")
	}
	if _, err := RewriteClientCredentials(path, "x", "y", "", "", []string{"upstream"}); err == nil {
		t.Fatal("expected an error for empty new credentials")
	}
}

func TestRewriteClientCredentialsInvalidSource(t *testing.T) {
	path := writeFixture(t, "not: [valid yaml")
	if _, err := RewriteClientCredentials(path, "a", "b", "c", "d", []string{"upstream"}); err == nil {
		t.Fatal("expected an error for an unparseable config file")
	}
}

func TestRewriteClientCredentialsWriteFailureKeepsFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits only")
	}
	if os.Getuid() == 0 {
		t.Skip("root bypasses directory permission checks")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(rewriteFixture), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	if _, err := RewriteClientCredentials(path, "hrpauth-internal-super", "super-secret", "x", "y", []string{"upstream"}); err == nil {
		t.Fatal("expected an error when the directory is read-only")
	}
	out, _ := os.ReadFile(path)
	if string(out) != rewriteFixture {
		t.Error("a failed write-back must leave the original file intact")
	}
}

func TestIsRenderedDeployment(t *testing.T) {
	if IsRenderedDeployment() {
		t.Fatal("no render envs set, must report false")
	}
	t.Setenv(RenderedConfigTemplateEnv, "/etc/ha/config.yaml.tmpl")
	t.Setenv(RenderedConfigPathEnv, "/app/config.yaml")
	if !IsRenderedDeployment() {
		t.Fatal("both render envs set, must report true")
	}
}
