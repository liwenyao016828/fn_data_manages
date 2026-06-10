package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ============================================================
// 配置
// ============================================================

var scanPorts = []int{
	3306, 3307, 3308, 3309, 3310, 3316, // MySQL / MariaDB
	5432, 5433, 5434, // PostgreSQL
	6379, 6380, 6381, // Redis
}

// 只扫描 loopback。Docker bridge IP (172.x.x.x) 与 127.0.0.1 指向同一物理服务，
// 由 Docker 容器的 ReachableFrom 字段补充可达地址，避免重复。
var scanHosts = []string{"127.0.0.1"}

// 通用弱密码表（Docker 环境变量优先，这些仅作 fallback）
var weakPasswords = map[string][]struct {
	user string
	pass string
}{
	"mysql":      {{"root", ""}, {"root", "root"}, {"root", "123456"}, {"root", "password"}},
	"mariadb":    {{"root", ""}, {"root", "root"}, {"root", "123456"}, {"root", "password"}},
	"postgresql": {{"postgres", ""}, {"postgres", "postgres"}, {"postgres", "123456"}, {"postgres", "password"}},
	"redis":      {{"", ""}, {"", "redis"}, {"", "123456"}, {"", "password"}},
}

// dockerSocketPath 是飞牛宿主 docker daemon 的 Unix Socket
// 通过 fnpack 挂载到容器内（见 fpk/cmd/main 中的 setupDockerAccess）
var dockerSocketPath = "/var/run/docker.sock"

// dockerAvailable 在启动时探一次，全局可读
var (
	dockerAvailableMu sync.RWMutex
	dockerAvailable   bool
)

func init() {
	dockerAvailableMu.Lock()
	dockerAvailable = pingDockerDaemon()
	dockerAvailableMu.Unlock()
	loadIgnoreFromDisk()
	go autoDetect()
	go dockerWatcher()
}

// dockerWatcher 周期性重探 socket（fnpack 可能在容器启动后才挂入）
func dockerWatcher() {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for range t.C {
		newVal := pingDockerDaemon()
		dockerAvailableMu.Lock()
		old := dockerAvailable
		dockerAvailable = newVal
		dockerAvailableMu.Unlock()
		if newVal != old {
			// 状态变化时跑一次扫描
			runDetect()
		}
	}
}

func pingDockerDaemon() bool {
	// 先看 socket 文件是否存在
	if _, err := os.Stat(dockerSocketPath); err != nil {
		return false
	}
	// 调一下 /_ping，能响应就说明 daemon 可达
	_, status, err := dockerUnixSocketGet("/_ping", "text/plain")
	if err != nil {
		return false
	}
	return status == http.StatusOK
}

// dockerUnixSocketGet 走 Unix Socket 调 Docker Engine API
// accept 决定响应内容：JSON 解析 或 原始文本
func dockerUnixSocketGet(path, accept string) ([]byte, int, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", dockerSocketPath, 3*time.Second)
			},
		},
	}
	req, err := http.NewRequest("GET", "http://unix"+path, nil)
	if err != nil {
		return nil, 0, err
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// dockerUnixSocketGetJSON 是上面函数的 JSON 便捷封装
func dockerUnixSocketGetJSON(path string, out interface{}) (int, error) {
	body, status, err := dockerUnixSocketGet(path, "application/json")
	if err != nil {
		return status, err
	}
	if status != http.StatusOK {
		return status, fmt.Errorf("status %d: %s", status, string(body))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return status, err
		}
	}
	return status, nil
}

// ============================================================
// 缓存 & 锁
// ============================================================

var (
	detectCache   []DetectedInstance
	detectMu      sync.RWMutex
	detectRunning bool
)

func init() {
	loadIgnoreFromDisk()
	go autoDetect()
}

func autoDetect() {
	time.Sleep(3 * time.Second)
	runDetect()
}

// ============================================================
// 核心扫描逻辑
// ============================================================

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
	var wg sync.WaitGroup

	// ---- Phase 1: Docker 容器 ----
	dockerContainers := getDockerContainers()
	dockerClaimedPorts := make(map[int]string) // hostPort -> containerName
	for _, c := range dockerContainers {
		dockerClaimedPorts[c.HostPort] = c.Name
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, c := range dockerContainers {
			results <- buildDockerInstance(c)
		}
	}()

	// ---- Phase 2: 端口扫描（仅 127.0.0.1，跳过 Docker 已认领端口）----
	for _, host := range scanHosts {
		for _, port := range scanPorts {
			if _, claimed := dockerClaimedPorts[port]; claimed {
				continue
			}
			wg.Add(1)
			go func(h string, p int) {
				defer wg.Done()
				inst := probeHostPort(h, p)
				if inst != nil {
					results <- *inst
				}
			}(host, port)
		}
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// ---- 收集 & 去重 ----
	var detected []DetectedInstance
	seen := make(map[string]int) // fingerprint -> index
	for inst := range results {
		key := inst.Fingerprint
		if key == "" {
			key = fmt.Sprintf("%s:%d:%s", inst.Host, inst.Port, inst.Type)
		}
		if idx, ok := seen[key]; ok {
			// 升级：如果新结果已认证而旧的不是，替换
			if inst.Status == "已认证" && detected[idx].Status != "已认证" {
				detected[idx] = inst
			}
			continue
		}
		seen[key] = len(detected)
		detected = append(detected, inst)
	}

	// ---- 应用忽略列表 ----
	ignored := loadIgnoreSet()
	for i := range detected {
		detected[i].Ignored = ignored[detected[i].Fingerprint]
	}

	detectMu.Lock()
	detectCache = detected
	detectMu.Unlock()
}

// buildDockerInstance 从 Docker 容器信息构建 DetectedInstance，并尝试自动认证
func buildDockerInstance(c dockerContainer) DetectedInstance {
	// 决策：主地址用哪个？
	// - 如果容器有宿主机端口映射，127.0.0.1:hostPort 永远可达（前提是 app 能访问宿主机网络）
	// - 如果没有端口映射，必须用容器 IP（需要 app 与 db 在同一 docker 网络）
	// 两者都有时，主地址显示容器 IP（更真实），但 127.0.0.1:hostPort 作为可达地址
	var host string
	var port int
	if c.HostPort > 0 {
		host = "127.0.0.1"
		port = c.HostPort
	} else {
		host = c.ContainerIP
		port = c.ContainerPort
	}

	inst := DetectedInstance{
		Name:        fmt.Sprintf("%s [%s]", strings.ToUpper(c.DbType), c.Name),
		Type:        c.DbType,
		Host:        host,
		Port:        port,
		Source:      "Docker",
		Image:       c.Image,
		Status:      "未认证",
		Container:   c.Name,
		ContainerID: c.ID,
		Fingerprint: fmt.Sprintf("docker:%s:%s", c.Name, c.DbType),
	}

	// 可达地址列表：把所有可能的访问方式都列出来，方便切换
	if c.HostPort > 0 {
		inst.ReachableFrom = []string{fmt.Sprintf("127.0.0.1:%d (宿主机映射)", c.HostPort)}
	} else {
		inst.ReachableFrom = []string{}
	}
	if c.ContainerIP != "" {
		// 把主 IP 之外的其他 bridge IP 也补上
		allIPs := getContainerNetworkIPs(c.ID)
		for _, ip := range allIPs {
			ipBase := strings.SplitN(ip, " ", 2)[0]
			if ipBase == c.ContainerIP {
				continue
			}
			inst.ReachableFrom = append(inst.ReachableFrom, ip)
		}
		// 把主容器 IP 也加进去（如果不是主地址的话）
		if c.HostPort > 0 {
			inst.ReachableFrom = append(inst.ReachableFrom,
				fmt.Sprintf("%s:%d (容器内网)", c.ContainerIP, c.ContainerPort))
		}
	}

	// 1) 优先用 Docker 环境变量中的密码，尝试主地址
	if envPwd := extractEnvPassword(c); envPwd != "" {
		if tryAuth(host, port, c.DbType, envPwd, &inst) {
			return inst
		}
		// 失败时尝试容器 IP（只有无端口映射的场景才需要 fallback）
		if c.HostPort == 0 && c.ContainerIP != "" {
			if tryAuth(c.ContainerIP, c.ContainerPort, c.DbType, envPwd, &inst) {
				inst.Host = c.ContainerIP
				inst.Port = c.ContainerPort
				return inst
			}
		}
	}

	// 2) Fallback: 弱密码表
	if !tryWeakPasswordsAndGet(host, port, c.DbType, &inst) {
		if c.HostPort == 0 && c.ContainerIP != "" {
			if tryWeakPasswordsAndGet(c.ContainerIP, c.ContainerPort, c.DbType, &inst) {
				inst.Host = c.ContainerIP
				inst.Port = c.ContainerPort
			}
		}
	}
	return inst
}

// tryWeakPasswordsAndGet 尝试弱密码表，返回是否成功
func tryWeakPasswordsAndGet(host string, port int, dbType string, inst *DetectedInstance) bool {
	creds, ok := weakPasswords[dbType]
	if !ok {
		return false
	}
	for _, c := range creds {
		if tryAuth(host, port, dbType, c.pass, inst) {
			inst.Username = c.user
			inst.WeakPassword = true
			return true
		}
	}
	return false
}

// probeHostPort 探测 127.0.0.1 上的单个端口
func probeHostPort(host string, port int) *DetectedInstance {
	if !scanPort(host, port, 1*time.Second) {
		return nil
	}

	dbType := probeServiceType(host, port)
	if dbType == "" {
		return nil
	}

	inst := &DetectedInstance{
		Name:        fmt.Sprintf("%s:%d", strings.ToUpper(dbType), port),
		Type:        dbType,
		Host:        host,
		Port:        port,
		Source:      "宿主机",
		Status:      "未认证",
		Fingerprint: fmt.Sprintf("local:%s:%d:%s", host, port, dbType),
	}

	tryWeakPasswords(host, port, dbType, inst)
	return inst
}

// ============================================================
// 协议探测
// ============================================================

// probeServiceType 通过协议握手判断服务类型（不依赖端口号）
func probeServiceType(host string, port int) string {
	addr := fmt.Sprintf("%s:%d", host, port)

	// 1) Redis: 发 PING，看 +PONG / -NOAUTH / -ERR
	if conn, err := net.DialTimeout("tcp", addr, 2*time.Second); err == nil {
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		_, _ = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
		buf := make([]byte, 128)
		if n, err := conn.Read(buf); err == nil && n > 0 {
			resp := string(buf[:n])
			if (resp[0] == '+' || resp[0] == '-') &&
				(strings.Contains(resp, "PONG") || strings.Contains(resp, "NOAUTH") || strings.Contains(resp, "ERR")) {
				conn.Close()
				return "redis"
			}
		}
		conn.Close()
	}

	// 2) MySQL/MariaDB: 读握手包，首字节 0x0a = 协议版本 10
	if conn, err := net.DialTimeout("tcp", addr, 2*time.Second); err == nil {
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		buf := make([]byte, 256)
		if n, err := conn.Read(buf); err == nil && n > 5 && buf[0] == 0x0a {
			// 提取版本字符串
			vEnd := 1
			for vEnd < n && buf[vEnd] != 0x00 {
				vEnd++
			}
			version := strings.ToLower(string(buf[1:vEnd]))
			conn.Close()
			if strings.Contains(version, "mariadb") {
				return "mariadb"
			}
			return "mysql"
		}
		conn.Close()
	}

	// 3) PostgreSQL: 发 StartupMessage，看 'E'/'R'/'N' 响应
	if conn, err := net.DialTimeout("tcp", addr, 2*time.Second); err == nil {
		conn.SetDeadline(time.Now().Add(3 * time.Second))
		startup := []byte{0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x03}
		_, _ = conn.Write(startup)
		buf := make([]byte, 128)
		if n, err := conn.Read(buf); err == nil && n > 0 {
			conn.Close()
			if buf[0] == 'E' || buf[0] == 'N' || buf[0] == 'R' {
				return "postgresql"
			}
		}
		conn.Close()
	}

	return ""
}

// ============================================================
// 认证
// ============================================================

// extractEnvPassword 从 Docker 容器环境变量中提取密码
func extractEnvPassword(c dockerContainer) string {
	switch c.DbType {
	case "mysql", "mariadb":
		if pwd := c.EnvVars["MYSQL_ROOT_PASSWORD"]; pwd != "" {
			return pwd
		}
		if _, ok := c.EnvVars["MYSQL_ALLOW_EMPTY_PASSWORD"]; ok {
			return "" // 空密码标记
		}
	case "postgresql":
		if pwd := c.EnvVars["POSTGRES_PASSWORD"]; pwd != "" {
			return pwd
		}
	case "redis":
		if pwd := c.EnvVars["REDIS_PASSWORD"]; pwd != "" {
			return pwd
		}
	}
	return "" // 无环境变量密码
}

// tryAuth 尝试用指定密码认证，成功则更新 inst
func tryAuth(host string, port int, dbType, password string, inst *DetectedInstance) bool {
	switch dbType {
	case "mysql", "mariadb":
		user := "root"
		if tryMySQLAuth(host, port, user, password) {
			inst.Username = user
			inst.Password = password
			inst.Version = getVersionMySQL(host, port, user, password)
			inst.Status = "已认证"
			if strings.Contains(strings.ToLower(inst.Version), "mariadb") {
				inst.Type = "mariadb"
				inst.Name = fmt.Sprintf("MARIADB [%s]", inst.Container)
			}
			return true
		}
	case "postgresql":
		user := "postgres"
		if tryPostgreSQLAuth(host, port, user, password) {
			inst.Username = user
			inst.Password = password
			inst.Version = getVersionPostgreSQL(host, port, user, password)
			inst.Status = "已认证"
			return true
		}
	case "redis":
		if tryRedisAuth(host, port, password) {
			inst.Password = password
			inst.Status = "已认证"
			return true
		}
	}
	return false
}

// tryWeakPasswords 用弱密码表逐一尝试
func tryWeakPasswords(host string, port int, dbType string, inst *DetectedInstance) {
	creds, ok := weakPasswords[dbType]
	if !ok {
		return
	}
	for _, c := range creds {
		if tryAuth(host, port, dbType, c.pass, inst) {
			inst.Username = c.user
			inst.WeakPassword = true
			return
		}
	}
}

// ============================================================
// 底层认证函数
// ============================================================

func tryMySQLAuth(host string, port int, username, password string) bool {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=2s&readTimeout=2s", username, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false
	}
	defer db.Close()
	var v string
	return db.QueryRow("SELECT VERSION()").Scan(&v) == nil
}

func getVersionMySQL(host string, port int, username, password string) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=2s", username, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return ""
	}
	defer db.Close()
	var v string
	if err := db.QueryRow("SELECT VERSION()").Scan(&v); err != nil {
		return ""
	}
	return v
}

func tryPostgreSQLAuth(host string, port int, username, password string) bool {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable&connect_timeout=2", username, password, host, port)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return false
	}
	defer db.Close()
	var v string
	return db.QueryRow("SELECT version()").Scan(&v) == nil
}

func getVersionPostgreSQL(host string, port int, username, password string) string {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable&connect_timeout=2", username, password, host, port)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return ""
	}
	defer db.Close()
	var v string
	if err := db.QueryRow("SELECT version()").Scan(&v); err != nil {
		return ""
	}
	parts := strings.Split(v, ",")
	return strings.TrimSpace(parts[0])
}

func tryRedisAuth(host string, port int, password string) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))

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

// ============================================================
// 端口扫描
// ============================================================

func scanPort(host string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ============================================================
// Docker 容器发现（走 Docker Engine API，不依赖 docker CLI）
// ============================================================

type dockerContainer struct {
	ID            string
	Name          string
	Image         string
	HostPort      int
	ContainerPort int
	ContainerIP   string // 容器在 bridge 上的 IP
	DbType        string
	EnvVars       map[string]string
}

// dockerAPIContainerInfo 是 /containers/json?all=0 返回的简化字段
// （用 inline struct 即可，下面不再需要独立类型）

func getDockerContainers() []dockerContainer {
	if !isDockerAvailable() {
		return nil
	}

	// 列出所有运行中容器，附带网络信息
	var list []struct {
		ID    string   `json:"Id"`
		Names []string `json:"Names"`
		Image string   `json:"Image"`
		Ports []struct {
			IP          string `json:"IP"`
			PrivatePort int    `json:"PrivatePort"`
			PublicPort  int    `json:"PublicPort"`
			Type        string `json:"Type"`
		} `json:"Ports"`
	}
	// 限制运行中容器（避免扫到停止的）
	_, err := dockerUnixSocketGetJSON("/containers/json?all=0&size=0", &list)
	if err != nil {
		return nil
	}

	var containers []dockerContainer
	for _, c := range list {
		image := strings.ToLower(c.Image)
		dbType := imageToDbType(image)
		if dbType == "" {
			continue
		}

		// 取容器 ID（去掉 sha256: 前缀后取前 12 位，跟 docker CLI 行为一致）
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}

		// 容器名（去掉开头的 /）
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		// 解析端口映射：找属于该 dbType 默认端口的映射
		defaultPort := defaultPortForType(dbType)
		hostPort, containerPort := 0, 0
		for _, p := range c.Ports {
			if p.PrivatePort == defaultPort {
				hostPort = p.PublicPort
				containerPort = p.PrivatePort
				break
			}
		}
		if containerPort == 0 {
			containerPort = defaultPort
		}

		// 拿容器主 IP
		containerIP := getContainerPrimaryIP(id)

		containers = append(containers, dockerContainer{
			ID:            id,
			Name:          name,
			Image:         c.Image,
			HostPort:      hostPort,
			ContainerPort: containerPort,
			ContainerIP:   containerIP,
			DbType:        dbType,
			EnvVars:       getDockerEnvVars(id),
		})
	}
	return containers
}

func imageToDbType(image string) string {
	if strings.Contains(image, "mariadb") {
		return "mariadb"
	}
	if strings.Contains(image, "mysql") {
		return "mysql"
	}
	if strings.Contains(image, "redis") {
		return "redis"
	}
	if strings.Contains(image, "postgres") {
		return "postgresql"
	}
	return ""
}

func defaultPortForType(dbType string) int {
	switch dbType {
	case "redis":
		return 6379
	case "postgresql":
		return 5432
	default:
		return 3306
	}
}

func isDockerAvailable() bool {
	dockerAvailableMu.RLock()
	defer dockerAvailableMu.RUnlock()
	return dockerAvailable
}

// refreshDockerAvailability 在 cmd/main 挂载 socket 后重新探一次
func refreshDockerAvailability() {
	dockerAvailableMu.Lock()
	dockerAvailable = pingDockerDaemon()
	dockerAvailableMu.Unlock()
}

func getDockerEnvVars(containerID string) map[string]string {
	envVars := make(map[string]string)
	if !isDockerAvailable() {
		return envVars
	}
	var info struct {
		Config struct {
			Env []string `json:"Env"`
		} `json:"Config"`
	}
	if _, err := dockerUnixSocketGetJSON("/containers/"+containerID+"/json", &info); err != nil {
		return envVars
	}
	for _, env := range info.Config.Env {
		if kv := strings.SplitN(env, "=", 2); len(kv) == 2 {
			envVars[kv[0]] = kv[1]
		}
	}
	return envVars
}

func getContainerNetworkIPs(containerID string) []string {
	if !isDockerAvailable() {
		return nil
	}
	var info struct {
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	_, err := dockerUnixSocketGetJSON("/containers/"+containerID+"/json", &info)
	if err != nil {
		return nil
	}

	var ips []string
	seen := make(map[string]bool)
	for name, net := range info.NetworkSettings.Networks {
		if net.IPAddress == "" || seen[net.IPAddress] {
			continue
		}
		seen[net.IPAddress] = true
		ips = append(ips, fmt.Sprintf("%s (%s)", net.IPAddress, name))
	}
	return ips
}

// getContainerPrimaryIP 获取容器在第一个 bridge 网络上的 IP
func getContainerPrimaryIP(containerID string) string {
	if !isDockerAvailable() {
		return ""
	}
	var info struct {
		NetworkSettings struct {
			IPAddress string `json:"IPAddress"`
			Networks  map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	_, err := dockerUnixSocketGetJSON("/containers/"+containerID+"/json", &info)
	if err != nil {
		return ""
	}
	// 优先取 bridge 网络，否则取第一个非空 IP
	if br, ok := info.NetworkSettings.Networks["bridge"]; ok && br.IPAddress != "" {
		return br.IPAddress
	}
	for _, net := range info.NetworkSettings.Networks {
		if net.IPAddress != "" {
			return net.IPAddress
		}
	}
	// 最后兜底用 NetworkSettings.IPAddress
	return info.NetworkSettings.IPAddress
}

// ============================================================
// HTTP Handler
// ============================================================

func detectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 包装响应：同时返回实例列表 + Docker socket 状态（前端可显示提示）
	if r.Method == "GET" {
		detectMu.RLock()
		data := detectCache
		detectMu.RUnlock()
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": filterIgnored(data, r.URL.Query().Get("all") == "true"),
			"docker": map[string]interface{}{
				"socketExists": isDockerAvailable() || socketFileExists(),
				"canConnect":   isDockerAvailable(),
				"fixCommand":   "sudo chmod 666 /var/run/docker.sock",
			},
		})
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
	writeJSON(w, map[string]interface{}{
		"code": 0,
		"data": filterIgnored(data, r.URL.Query().Get("all") == "true"),
		"docker": map[string]interface{}{
			"socketExists": isDockerAvailable() || socketFileExists(),
			"canConnect":   isDockerAvailable(),
			"fixCommand":   "sudo chmod 666 /var/run/docker.sock",
		},
	})
}

func socketFileExists() bool {
	_, err := os.Stat(dockerSocketPath)
	return err == nil
}

func filterIgnored(data []DetectedInstance, showAll bool) []DetectedInstance {
	if showAll {
		return data
	}
	filtered := make([]DetectedInstance, 0, len(data))
	for _, inst := range data {
		if !inst.Ignored {
			filtered = append(filtered, inst)
		}
	}
	return filtered
}

// ============================================================
// 忽略列表（持久化到 ${TRIM_PKGVAR}/detect-ignore.json）
// ============================================================

var (
	ignoreMu    sync.RWMutex
	ignoreCache = make(map[string]string) // fingerprint -> label
)

func ignoreFilePath() string {
	dir := os.Getenv("TRIM_PKGVAR")
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "detect-ignore.json")
}

func loadIgnoreFromDisk() {
	data, err := os.ReadFile(ignoreFilePath())
	if err != nil {
		return
	}
	var list []struct {
		Fingerprint string `json:"fingerprint"`
		Label       string `json:"label"`
	}
	if err := json.Unmarshal(data, &list); err != nil {
		return
	}
	ignoreMu.Lock()
	for _, item := range list {
		if item.Fingerprint != "" {
			ignoreCache[item.Fingerprint] = item.Label
		}
	}
	ignoreMu.Unlock()
}

func saveIgnoreToDisk() {
	ignoreMu.RLock()
	type entry struct {
		Fingerprint string `json:"fingerprint"`
		Label       string `json:"label"`
	}
	var list []entry
	for fp, label := range ignoreCache {
		list = append(list, entry{fp, label})
	}
	ignoreMu.RUnlock()

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	path := ignoreFilePath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, data, 0644)
}

func loadIgnoreSet() map[string]bool {
	ignoreMu.RLock()
	defer ignoreMu.RUnlock()
	out := make(map[string]bool, len(ignoreCache))
	for k := range ignoreCache {
		out[k] = true
	}
	return out
}

func addIgnore(fingerprint, label string) {
	ignoreMu.Lock()
	ignoreCache[fingerprint] = label
	ignoreMu.Unlock()
	saveIgnoreToDisk()
	applyIgnoreToCache()
}

func removeIgnore(fingerprint string) {
	ignoreMu.Lock()
	delete(ignoreCache, fingerprint)
	ignoreMu.Unlock()
	saveIgnoreToDisk()
	applyIgnoreToCache()
}

func applyIgnoreToCache() {
	ignored := loadIgnoreSet()
	detectMu.Lock()
	for i := range detectCache {
		detectCache[i].Ignored = ignored[detectCache[i].Fingerprint]
	}
	detectMu.Unlock()
}

func detectIgnoreHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case "GET":
		ignoreMu.RLock()
		type item struct {
			Fingerprint string `json:"fingerprint"`
			Label       string `json:"label"`
		}
		var out []item
		for fp, label := range ignoreCache {
			out = append(out, item{fp, label})
		}
		ignoreMu.RUnlock()
		writeJSON(w, map[string]interface{}{"code": 0, "data": out})

	case "POST":
		var body struct {
			Fingerprint string `json:"fingerprint"`
			Label       string `json:"label"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "请求体格式错误"})
			return
		}
		if body.Fingerprint == "" {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "fingerprint 不能为空"})
			return
		}
		addIgnore(body.Fingerprint, body.Label)
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "已忽略"})

	case "DELETE":
		fp := r.URL.Query().Get("fingerprint")
		if fp == "" {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "fingerprint 不能为空"})
			return
		}
		removeIgnore(fp)
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "已取消忽略"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
