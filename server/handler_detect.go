package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var scanPorts = []struct {
	Port int
	Type string
}{
	{3306, "mysql"}, {3307, "mysql"}, {3308, "mysql"},
	{3309, "mysql"}, {3310, "mysql"}, {3316, "mysql"},
	{6379, "redis"}, {6380, "redis"}, {6381, "redis"},
}

var scanHosts = []string{
	"127.0.0.1",
	"172.17.0.1", "172.18.0.1", "172.19.0.1",
	"172.17.0.2", "172.18.0.2", "172.19.0.2",
	"172.17.0.3", "172.18.0.3",
}

var (
	detectCache   []DetectedInstance
	detectMu      sync.RWMutex
	detectRunning bool
)

func init() {
	go autoDetect()
}

func autoDetect() {
	time.Sleep(3 * time.Second)
	runDetect()
}

func runDetect() {
	detectMu.Lock()
	if detectRunning {
		detectMu.Unlock()
		return
	}
	detectRunning = true
	detectMu.Unlock()

	defer func() {
		detectMu.Lock()
		detectRunning = false
		detectMu.Unlock()
	}()

	results := make(chan DetectedInstance, 200)
	var detected []DetectedInstance
	var wg sync.WaitGroup

	// 1. Docker containers with credential extraction
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanDockerWithCredentials(results)
	}()

	// 2. Port scanning
	for _, host := range scanHosts {
		for _, sp := range scanPorts {
			wg.Add(1)
			go func(h string, port int, typ string) {
				defer wg.Done()
				if scanPort(h, port, 1*time.Second) {
					inst := DetectedInstance{
						Name:   fmt.Sprintf("%s:%d", strings.ToUpper(typ), port),
						Type:   typ,
						Host:   h,
						Port:   port,
						Source: "宿主机",
						Status: "未认证",
					}

					// Try auto-authentication with common credentials
					if typ == "mysql" {
						if tryAuthMySQL(h, port, &inst) {
							inst.Status = "已认证"
						}
					} else if typ == "redis" {
						if tryAuthRedis(h, port, &inst) {
							inst.Status = "已认证"
						}
					}

					results <- inst
				}
			}(host, sp.Port, sp.Type)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Deduplicate
	seen := make(map[string]bool)
	for inst := range results {
		key := fmt.Sprintf("%s:%d", inst.Host, inst.Port)
		if !seen[key] {
			seen[key] = true
			detected = append(detected, inst)
		} else {
			// If already seen, update if this one has better auth status
			for i, existing := range detected {
				if existing.Host == inst.Host && existing.Port == inst.Port {
					if inst.Status == "已认证" && existing.Status != "已认证" {
						detected[i] = inst
					}
					break
				}
			}
		}
	}

	detectMu.Lock()
	detectCache = detected
	detectMu.Unlock()
}

func detectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method == "GET" {
		detectMu.RLock()
		data := detectCache
		detectMu.RUnlock()
		writeJSON(w, map[string]interface{}{"code": 0, "data": data})
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runDetect()
	detectMu.RLock()
	data := detectCache
	detectMu.RUnlock()
	writeJSON(w, map[string]interface{}{"code": 0, "data": data})
}

func scanPort(host string, port int, timeout time.Duration) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

type dockerContainer struct {
	ID       string
	Name     string
	Image    string
	HostPort int
	DbType   string
	EnvVars  map[string]string
}

func scanDockerWithCredentials(results chan<- DetectedInstance) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return
	}

	// Get container list
	cmd := exec.Command(dockerPath, "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Ports}}")
	var out strings.Builder
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var containers []dockerContainer

	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}

		image := strings.ToLower(parts[2])
		var dbType string
		if strings.Contains(image, "mysql") || strings.Contains(image, "mariadb") {
			dbType = "mysql"
		} else if strings.Contains(image, "redis") {
			dbType = "redis"
		} else {
			continue
		}

		hostPort := extractHostPort(parts[3], dbType)
		if hostPort == 0 {
			continue
		}

		// Get environment variables
		envVars := getDockerEnvVars(dockerPath, parts[0])

		containers = append(containers, dockerContainer{
			ID:       parts[0],
			Name:     parts[1],
			Image:    parts[2],
			HostPort: hostPort,
			DbType:   dbType,
			EnvVars:  envVars,
		})
	}

	// Authenticate each container
	for _, c := range containers {
		inst := DetectedInstance{
			Name:      fmt.Sprintf("%s [%s]", strings.ToUpper(c.DbType), c.Name),
			Type:      c.DbType,
			Host:      "127.0.0.1",
			Port:      c.HostPort,
			Source:    "Docker",
			Status:    "未认证",
			Container: c.Name,
		}

		if c.DbType == "mysql" {
			inst.Username = "root"
			// Try extracted password first
			if pwd, ok := c.EnvVars["MYSQL_ROOT_PASSWORD"]; ok && pwd != "" {
				inst.Password = pwd
				if tryMySQLAuth("127.0.0.1", c.HostPort, "root", pwd) {
					inst.Status = "已认证"
					inst.Version = getVersionMySQL("127.0.0.1", c.HostPort, "root", pwd)
					results <- inst
					continue
				}
			}

			// Try empty password
			if _, ok := c.EnvVars["MYSQL_ALLOW_EMPTY_PASSWORD"]; ok {
				if tryMySQLAuth("127.0.0.1", c.HostPort, "root", "") {
					inst.Password = ""
					inst.Status = "已认证"
					inst.Version = getVersionMySQL("127.0.0.1", c.HostPort, "root", "")
					results <- inst
					continue
				}
			}

			// Try common passwords
			commonPasswords := []string{"", "root", "123456", "12345678", "password"}
			for _, pwd := range commonPasswords {
				if tryMySQLAuth("127.0.0.1", c.HostPort, "root", pwd) {
					inst.Password = pwd
					inst.Status = "已认证"
					inst.WeakPassword = true
					inst.Version = getVersionMySQL("127.0.0.1", c.HostPort, "root", pwd)
					break
				}
			}

		} else if c.DbType == "redis" {
			// Try extracted password
			if pwd, ok := c.EnvVars["REDIS_PASSWORD"]; ok && pwd != "" {
				inst.Password = pwd
				if tryRedisAuth("127.0.0.1", c.HostPort, pwd) {
					inst.Status = "已认证"
					results <- inst
					continue
				}
			}

			// Try no password
			if tryRedisAuth("127.0.0.1", c.HostPort, "") {
				inst.Password = ""
				inst.Status = "已认证"
				results <- inst
				continue
			}

			// Try common passwords
			commonRedisPasswords := []string{"", "redis", "123456", "password"}
			for _, pwd := range commonRedisPasswords {
				if tryRedisAuth("127.0.0.1", c.HostPort, pwd) {
					inst.Password = pwd
					inst.Status = "已认证"
					inst.WeakPassword = true
					break
				}
			}
		}

		results <- inst
	}
}

func getDockerEnvVars(dockerPath, containerID string) map[string]string {
	envVars := make(map[string]string)

	cmd := exec.Command(dockerPath, "inspect", "--format", "{{json .Config.Env}}", containerID)
	out, err := cmd.Output()
	if err != nil {
		return envVars
	}

	var envList []string
	if err := json.Unmarshal(out, &envList); err != nil {
		return envVars
	}

	for _, env := range envList {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	return envVars
}

func tryAuthMySQL(host string, port int, inst *DetectedInstance) bool {
	credentials := []struct {
		user string
		pass string
		weak bool
	}{
		{"root", "", true},
		{"root", "root", true},
		{"root", "123456", true},
		{"root", "12345678", true},
		{"root", "password", true},
	}

	for _, cred := range credentials {
		if tryMySQLAuth(host, port, cred.user, cred.pass) {
			inst.Username = cred.user
			inst.Password = cred.pass
			inst.Version = getVersionMySQL(host, port, cred.user, cred.pass)
			inst.WeakPassword = cred.weak
			return true
		}
	}
	return false
}

func tryAuthRedis(host string, port int, inst *DetectedInstance) bool {
	passwords := []struct {
		pass string
		weak bool
	}{
		{"", true},
		{"redis", true},
		{"123456", true},
		{"password", true},
	}

	for _, pwd := range passwords {
		if tryRedisAuth(host, port, pwd.pass) {
			inst.Password = pwd.pass
			inst.WeakPassword = pwd.weak
			return true
		}
	}
	return false
}

func tryMySQLAuth(host string, port int, username, password string) bool {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=2s&readTimeout=2s",
		username, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false
	}
	defer db.Close()

	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	return err == nil
}

func getVersionMySQL(host string, port int, username, password string) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=2s",
		username, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return ""
	}
	defer db.Close()

	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		return ""
	}
	return version
}

func tryRedisAuth(host string, port int, password string) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	if password != "" {
		resp, err := redisDo(conn, "AUTH", password)
		if err != nil {
			return false
		}
		if s, ok := resp.(string); ok && s != "OK" {
			return false
		}
	}

	resp, err := redisDo(conn, "PING")
	if err != nil {
		return false
	}
	if pong, ok := resp.(string); ok && pong == "PONG" {
		return true
	}
	return false
}

func extractHostPort(ports string, dbType string) int {
	defaultPort := 3306
	if dbType == "redis" {
		defaultPort = 6379
	}

	parts := strings.Split(ports, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, fmt.Sprintf("->%d/tcp", defaultPort)) ||
			strings.Contains(p, fmt.Sprintf("->%d", defaultPort)) {
			idx := strings.Index(p, ":")
			if idx >= 0 {
				rest := p[idx+1:]
				arrowIdx := strings.Index(rest, "->")
				if arrowIdx >= 0 {
					portStr := rest[:arrowIdx]
					var port int
					fmt.Sscanf(portStr, "%d", &port)
					if port > 0 {
						return port
					}
				}
			}
		}
	}
	return 0
}
