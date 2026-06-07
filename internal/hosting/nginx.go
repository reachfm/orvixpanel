// Pure-Go nginx vhost generator. No OS calls — fully testable
// on any platform. The actual file write + `nginx -t` validation
// happens in WriteVHostConfig (provision_linux.go).
package hosting

import (
	"fmt"
	"path"
	"strings"
	"time"
)

// VHostConfig is the input to GenerateNginxConfig. Fields map
// directly to the nginx template variables.
type VHostConfig struct {
	Username    string // account username
	Domain      string // domain name (e.g. example.com)
	DocumentRoot string // absolute path; defaults to DomainDocumentRoot
	Port        int    // listen port; 0 → 80
	HTTPRedirectHTTPS bool // issue 301 to https:// when set
	OpenBasedir string // optional open_basedir; defaults to DocumentRoot
	PHP         bool   // include the PHP-FPM upstream block
	FpmSocket   string // unix socket; default "/run/php/orvix-<user>-<domain>.sock"
	ExtraHeaders []string // raw headers to add verbatim
}

// GenerateNginxConfig returns the rendered server-block text. The
// caller is responsible for writing it to a path and running
// `nginx -t` to validate.
func GenerateNginxConfig(v VHostConfig) (string, error) {
	if v.Username == "" || v.Domain == "" {
		return "", fmt.Errorf("VHostConfig: Username and Domain are required")
	}
	if v.Port == 0 {
		v.Port = 80
	}
	if v.DocumentRoot == "" {
		// Use DefaultPaths so the DocumentRoot is absolute.
		// The handler passes an absolute path explicitly when it
		// has the d.Hosting.Paths (e.g. dev/test override).
		v.DocumentRoot = path.Join(DefaultPaths().AccountHome(v.Username), "public_html", v.Domain)
	}
	if v.OpenBasedir == "" {
		v.OpenBasedir = v.DocumentRoot
	}
	if v.FpmSocket == "" {
		v.FpmSocket = fmt.Sprintf("/run/php/orvix-%s-%s.sock", v.Username, v.Domain)
	}

	var b strings.Builder
	// Server block.
	fmt.Fprintf(&b, "# OrvixPanel generated vhost\n")
	fmt.Fprintf(&b, "# account=%s domain=%s generated=%s\n", v.Username, v.Domain, time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "server {\n")
	fmt.Fprintf(&b, "    listen %d;\n", v.Port)
	fmt.Fprintf(&b, "    listen [::]:%d;\n", v.Port)
	fmt.Fprintf(&b, "    server_name %s;\n", v.Domain)

	// Document root.
	fmt.Fprintf(&b, "    root %s;\n", v.DocumentRoot)
	fmt.Fprintf(&b, "    index index.html index.php;\n")

	// Security headers.
	fmt.Fprintf(&b, "    add_header X-Content-Type-Options \"nosniff\" always;\n")
	fmt.Fprintf(&b, "    add_header X-Frame-Options \"SAMEORIGIN\" always;\n")
	fmt.Fprintf(&b, "    add_header Referrer-Policy \"strict-origin-when-cross-origin\" always;\n")

	// Extra headers from operator config.
	for _, h := range v.ExtraHeaders {
		fmt.Fprintf(&b, "    add_header %s;\n", h)
	}

	// HTTP→HTTPS redirect.
	if v.HTTPRedirectHTTPS {
		fmt.Fprintf(&b, "    return 301 https://$host$request_uri;\n")
		fmt.Fprintf(&b, "}\n")
		return b.String(), nil
	}

	// PHP-FPM upstream.
	if v.PHP {
		fmt.Fprintf(&b, "    location ~ \\.php$ {\n")
		fmt.Fprintf(&b, "        include snippets/fastcgi-php.conf;\n")
		fmt.Fprintf(&b, "        fastcgi_pass unix:%s;\n", v.FpmSocket)
		fmt.Fprintf(&b, "        fastcgi_param PHP_VALUE \"open_basedir=%s\";\n", v.OpenBasedir)
		fmt.Fprintf(&b, "    }\n")
	}

	// Static files.
	fmt.Fprintf(&b, "    location / {\n")
	fmt.Fprintf(&b, "        try_files $uri $uri/ /index.html;\n")
	fmt.Fprintf(&b, "    }\n")

	// Deny dotfiles.
	fmt.Fprintf(&b, "    location ~ /\\. {\n")
	fmt.Fprintf(&b, "        deny all;\n")
	fmt.Fprintf(&b, "    }\n")

	// Log files.
	fmt.Fprintf(&b, "    access_log /var/log/nginx/%s-%s.access.log;\n", v.Username, v.Domain)
	fmt.Fprintf(&b, "    error_log  /var/log/nginx/%s-%s.error.log;\n", v.Username, v.Domain)

	fmt.Fprintf(&b, "}\n")
	return b.String(), nil
}
