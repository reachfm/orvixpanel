// Pure-Go PHP-FPM pool generator. No OS calls — fully testable.
package hosting

import (
	"fmt"
	"path"
	"strings"
	"time"
)

// FPMConfig is the input to GenerateFPMPool.
type FPMConfig struct {
	Username  string // account username
	Domain    string // domain name
	UnixUser  string // pool runs as this user; default = Username
	UnixGroup string // pool runs as this group; default = Username
	Listen    string // unix socket; default /run/php/orvix-<user>-<domain>.sock
	PHPVersion string // informational; not in pool config
	PM        string // "dynamic" | "static" | "ondemand"; default "ondemand"
	MaxChildren int // default 5
	StartServers int // default 2 (for dynamic only)
	MinSpareServers int // default 1
	MaxSpareServers int // default 3
	MaxRequests int // default 500
	MemoryLimit string // PHP memory_limit; default "256M"
	OpenBasedir string // open_basedir; default = document root
	Chdir string // chdir; default /
}

// GenerateFPMPool returns the rendered [orvix-<user>-<domain>] pool
// text. Drop into /etc/php/X.Y/fpm/pool.d/.
func GenerateFPMPool(c FPMConfig) (string, error) {
	if c.Username == "" || c.Domain == "" {
		return "", fmt.Errorf("FPMConfig: Username and Domain are required")
	}
	if c.UnixUser == "" {
		c.UnixUser = c.Username
	}
	if c.UnixGroup == "" {
		c.UnixGroup = c.Username
	}
	if c.Listen == "" {
		c.Listen = fmt.Sprintf("/run/php/orvix-%s-%s.sock", c.Username, c.Domain)
	}
	if c.PM == "" {
		c.PM = "ondemand"
	}
	if c.MaxChildren == 0 {
		c.MaxChildren = 5
	}
	if c.StartServers == 0 {
		c.StartServers = 2
	}
	if c.MinSpareServers == 0 {
		c.MinSpareServers = 1
	}
	if c.MaxSpareServers == 0 {
		c.MaxSpareServers = 3
	}
	if c.MaxRequests == 0 {
		c.MaxRequests = 500
	}
	if c.MemoryLimit == "" {
		c.MemoryLimit = "256M"
	}
	if c.OpenBasedir == "" {
		c.OpenBasedir = path.Join(DefaultPaths().AccountHome(c.Username), "public_html", c.Domain)
	}
	if c.Chdir == "" {
		c.Chdir = "/"
	}
	if c.PHPVersion == "" {
		c.PHPVersion = "8.5"
	}

	poolName := fmt.Sprintf("orvix-%s-%s", c.Username, c.Domain)
	var b strings.Builder
	fmt.Fprintf(&b, "; OrvixPanel generated php-fpm pool\n")
	fmt.Fprintf(&b, "; account=%s domain=%s php=%s generated=%s\n",
		c.Username, c.Domain, c.PHPVersion, time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "[%s]\n", poolName)
	fmt.Fprintf(&b, "user = %s\n", c.UnixUser)
	fmt.Fprintf(&b, "group = %s\n", c.UnixGroup)
	fmt.Fprintf(&b, "listen = %s\n", c.Listen)
	fmt.Fprintf(&b, "listen.owner = www-data\n")
	fmt.Fprintf(&b, "listen.group = www-data\n")
	fmt.Fprintf(&b, "pm = %s\n", c.PM)
	fmt.Fprintf(&b, "pm.max_children = %d\n", c.MaxChildren)
	if c.PM == "dynamic" {
		fmt.Fprintf(&b, "pm.start_servers = %d\n", c.StartServers)
		fmt.Fprintf(&b, "pm.min_spare_servers = %d\n", c.MinSpareServers)
		fmt.Fprintf(&b, "pm.max_spare_servers = %d\n", c.MaxSpareServers)
	}
	fmt.Fprintf(&b, "pm.max_requests = %d\n", c.MaxRequests)
	fmt.Fprintf(&b, "chdir = %s\n", c.Chdir)
	fmt.Fprintf(&b, "php_admin_value[memory_limit] = %s\n", c.MemoryLimit)
	fmt.Fprintf(&b, "php_admin_value[open_basedir] = %s\n", c.OpenBasedir)
	fmt.Fprintf(&b, "php_admin_value[upload_max_filesize] = 64M\n")
	fmt.Fprintf(&b, "php_admin_value[post_max_size] = 64M\n")
	fmt.Fprintf(&b, "php_admin_flag[expose_php] = off\n")
	// Catchable error logging per-pool — operators love this.
	fmt.Fprintf(&b, "catch_workers_output = yes\n")
	fmt.Fprintf(&b, "decorate_workers_output = no\n")
	return b.String(), nil
}
