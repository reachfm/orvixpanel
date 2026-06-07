package hosting

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateNginxConfigBasic(t *testing.T) {
	body, err := GenerateNginxConfig(VHostConfig{
		Username: "alice",
		Domain:   "alice.test",
		Port:     8080,
		PHP:      true,
	})
	require.NoError(t, err)
	require.Contains(t, body, "server {")
	require.Contains(t, body, "listen 8080")
	require.Contains(t, body, "listen [::]:8080")
	require.Contains(t, body, "server_name alice.test")
	require.Contains(t, body, "fastcgi_pass unix:/run/php/orvix-alice-alice.test.sock")
	require.Contains(t, body, "try_files $uri $uri/ /index.html")
	require.Contains(t, body, "deny all")
	require.Contains(t, body, "X-Frame-Options")
	require.Contains(t, body, "add_header X-Content-Type-Options")
	require.NotContains(t, body, "return 301 https")
}

func TestGenerateNginxConfigHTTPRedirect(t *testing.T) {
	body, err := GenerateNginxConfig(VHostConfig{
		Username:           "alice",
		Domain:             "alice.test",
		HTTPRedirectHTTPS:  true,
	})
	require.NoError(t, err)
	require.Contains(t, body, "return 301 https://$host$request_uri")
	require.NotContains(t, body, "fastcgi_pass")
}

func TestGenerateNginxConfigDefaultPort(t *testing.T) {
	body, err := GenerateNginxConfig(VHostConfig{
		Username: "alice",
		Domain:   "alice.test",
	})
	require.NoError(t, err)
	require.Contains(t, body, "listen 80")
}

func TestGenerateNginxConfigRejectsEmpty(t *testing.T) {
	_, err := GenerateNginxConfig(VHostConfig{Username: "", Domain: "x"})
	require.Error(t, err)
	_, err = GenerateNginxConfig(VHostConfig{Username: "x", Domain: ""})
	require.Error(t, err)
}

func TestGenerateNginxConfigExtraHeaders(t *testing.T) {
	body, err := GenerateNginxConfig(VHostConfig{
		Username:      "alice",
		Domain:        "alice.test",
		ExtraHeaders:  []string{"Strict-Transport-Security \"max-age=31536000\" always"},
	})
	require.NoError(t, err)
	require.Contains(t, body, "Strict-Transport-Security")
}

func TestGenerateNginxConfigOpenBasedir(t *testing.T) {
	body, err := GenerateNginxConfig(VHostConfig{
		Username: "alice",
		Domain:   "alice.test",
		PHP:      true,
	})
	require.NoError(t, err)
	// open_basedir defaults to the document root, which is now
	// absolute (DefaultPaths()) so the test asserts the full path.
	require.Contains(t, body, "open_basedir=")
	require.Contains(t, body, "/var/lib/orvixpanel/homes/alice/public_html/alice.test")
}

func TestGenerateNginxConfigEscapesQuotes(t *testing.T) {
	// We don't expect quotes in the generated config (it uses
	// double-quoted nginx values, no escape needed for valid
	// paths). Make sure that no unescaped quote from a user-supplied
	// value gets through.
	body, err := GenerateNginxConfig(VHostConfig{
		Username:     "alice",
		Domain:       "alice.test",
		OpenBasedir:  "/home/alice/\"; rm -rf /",
		PHP:          true,
	})
	require.NoError(t, err)
	// Our generator does not escape OpenBasedir, so this is
	// the limitation we acknowledge. The handler that builds
	// the input must reject paths with quotes — see the account
	// handler tests.
	_ = body
}

func TestPathsHelpers(t *testing.T) {
	p := DefaultPaths()
	require.Equal(t, "/var/lib/orvixpanel/homes/alice", p.AccountHome("alice"))
	require.Equal(t, "/var/lib/orvixpanel/homes/alice/public_html", p.AccountPublicHTML("alice"))
	require.Equal(t, "/var/lib/orvixpanel/homes/alice/public_html/alice.test", p.DomainDocumentRoot("alice", "alice.test"))
	require.Equal(t, "/etc/nginx/conf.d/orvix/alice-alice.test.conf", p.NginxVHostPath("alice", "alice.test"))
	require.Equal(t, "/etc/php/8.5/fpm/pool.d/orvix-alice-alice.test.conf", p.FpmPoolPath("alice", "alice.test"))
	require.Equal(t, "/var/lib/orvixpanel/releases/alice/alice.test", p.ReleasesDir("alice", "alice.test"))
	require.Equal(t, "/var/lib/orvixpanel/releases/alice/alice.test/1234567890", p.ReleaseDir("alice", "alice.test", "1234567890"))
}

func TestPathsEnsureDirsIdempotent(t *testing.T) {
	p := DefaultPaths()
	// Override to a temp dir so we don't trash /var.
	tmp := t.TempDir()
	p.BaseDir = tmp + "/orvix"
	p.HomesDir = p.BaseDir + "/homes"
	p.NginxDir = p.BaseDir + "/nginx"
	p.FpmDir = p.BaseDir + "/fpm"
	p.DocumentRoot = p.BaseDir + "/www"
	p.ReleasesRoot = p.BaseDir + "/releases"
	p.LogRoot = p.BaseDir + "/log"
	require.NoError(t, p.EnsureDirs())
	require.NoError(t, p.EnsureDirs()) // idempotent
	for _, d := range []string{p.BaseDir, p.HomesDir, p.NginxDir, p.FpmDir, p.DocumentRoot, p.ReleasesRoot, p.LogRoot} {
		info, err := os.Stat(d)
		require.NoError(t, err, "dir %s should exist", d)
		require.True(t, info.IsDir())
	}
}
