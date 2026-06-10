package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func getDataDir() string {
	dir := os.Getenv("TRIM_PKGVAR")
	if dir == "" {
		dir = "./data"
	}
	os.MkdirAll(dir, 0755)
	return dir
}

var logMutex sync.Mutex

func sysLog(level, source, message string) {
	logMutex.Lock()
	defer logMutex.Unlock()
	dir := getDataDir()
	os.MkdirAll(dir, 0755)
	logPath := filepath.Join(dir, "app.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	timestamp := time.Now().Format("2006-01-02T15:04:05-07:00")
	line := fmt.Sprintf("%s [%s] [%s] %s\n", timestamp, strings.ToUpper(level), source, message)
	f.WriteString(line)
}

func sysLogInfo(source, message string)  { sysLog("info", source, message) }
func sysLogWarn(source, message string)  { sysLog("warning", source, message) }
func sysLogError(source, message string) { sysLog("error", source, message) }

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
		str := fmt.Sprintf("%v", v)
		b, err := strconv.ParseBool(str)
		if err == nil {
			return b
		}
	}
	return false
}

func getUint(m map[string]interface{}, key string) uint {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return uint(n)
		case int:
			return uint(n)
		case uint:
			return n
		}
	}
	return 0
}

// #3 fsBrowse 允许的根目录白名单
// - 优先使用环境变量 DM_FSBROWSE_ROOTS（逗号分隔）
// - 默认包含: ./data, /var, /tmp, /home, /vol, /mnt, /opt, /srv
// - Windows: 默认允许所有盘符根 + TRIM_PKGVAR
func defaultFsBrowseRoots() []string {
	env := os.Getenv("DM_FSBROWSE_ROOTS")
	if env != "" {
		parts := strings.Split(env, ",")
		roots := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				roots = append(roots, p)
			}
		}
		if len(roots) > 0 {
			return roots
		}
	}
	if runtime.GOOS == "windows" {
		return []string{"./data"}
	}
	return []string{"./data", "/var", "/tmp", "/home", "/vol", "/mnt", "/opt", "/srv"}
}

// isPathUnderRoots 检查目标路径是否在任一允许根目录之下
func isPathUnderRoots(targetPath string, roots []string) bool {
	cleaned := filepath.Clean(targetPath)
	for _, root := range roots {
		rootClean := filepath.Clean(root)
		rel, err := filepath.Rel(rootClean, cleaned)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		return true
	}
	return false
}

func fsBrowseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	allowedRoots := defaultFsBrowseRoots()
	// 总是把 TRIM_PKGVAR（数据目录）加入允许列表
	if dir := os.Getenv("TRIM_PKGVAR"); dir != "" {
		allowedRoots = append(allowedRoots, dir)
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		// 默认显示允许的根目录列表
		if runtime.GOOS == "windows" {
			// Windows: 列出存在的盘符
			var drives []map[string]interface{}
			for _, d := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
				drivePath := string(d) + ":\\"
				_, err := os.ReadDir(drivePath)
				if err == nil {
					drives = append(drives, map[string]interface{}{
						"name":  string(d) + ":\\",
						"path":  drivePath,
						"isDir": true,
						"drive": true,
					})
				}
			}
			writeJSON(w, map[string]interface{}{
				"code":   0,
				"path":   "(drives)",
				"parent": "",
				"dirs":   drives,
				"isRoot": true,
			})
			return
		}
		// Linux/Unix: 把所有允许的根作为"虚拟根目录"展示
		var roots []map[string]interface{}
		seen := map[string]bool{}
		for _, root := range allowedRoots {
			root = filepath.Clean(root)
			if _, err := os.Stat(root); err != nil {
				continue
			}
			if seen[root] {
				continue
			}
			seen[root] = true
			roots = append(roots, map[string]interface{}{
				"name":  root,
				"path":  root,
				"isDir": true,
				"drive": true,
			})
		}
		// 补充一个 home 目录
		if home, err := os.UserHomeDir(); err == nil && !seen[home] {
			roots = append(roots, map[string]interface{}{
				"name":  home + " (home)",
				"path":  home,
				"isDir": true,
				"drive": true,
			})
		}
		writeJSON(w, map[string]interface{}{
			"code":   0,
			"path":   "(roots)",
			"parent": "",
			"dirs":   roots,
			"isRoot": true,
		})
		return
	}

	reqPath = filepath.Clean(reqPath)
	if runtime.GOOS == "windows" && len(reqPath) == 2 && reqPath[1] == ':' {
		reqPath = reqPath + "\\"
	}

	// #3 白名单校验：路径必须在允许的根目录之下
	if !isPathUnderRoots(reqPath, allowedRoots) {
		writeJSON(w, map[string]interface{}{
			"code": 1,
			"msg":  "该目录不在允许的浏览范围内，请联系管理员配置 DM_FSBROWSE_ROOTS",
			"path": reqPath,
		})
		return
	}

	// 安全检查：限制可浏览的目录范围（黑名单兜底）
	if runtime.GOOS != "windows" {
		forbiddenPaths := []string{"/proc", "/sys", "/dev", "/boot", "/root", "/etc/ssh", "/etc/security"}
		for _, fp := range forbiddenPaths {
			if strings.HasPrefix(reqPath, fp) {
				writeJSON(w, map[string]interface{}{
					"code": 1,
					"msg":  "不允许浏览该系统目录",
					"path": reqPath,
				})
				return
			}
		}
	}

	isRoot := false
	if runtime.GOOS == "windows" && len(reqPath) == 3 && reqPath[1] == ':' && reqPath[2] == '\\' {
		parent := filepath.Dir(reqPath)
		if parent == reqPath {
			isRoot = true
		}
	}
	if !isRoot && (reqPath == "/" || reqPath == ".") {
		isRoot = true
	}

	if isRoot && runtime.GOOS == "windows" {
		var drives []map[string]interface{}
		for _, d := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			drivePath := string(d) + ":\\"
			_, err := os.ReadDir(drivePath)
			if err == nil {
				drives = append(drives, map[string]interface{}{
					"name":  string(d) + ":\\",
					"path":  drivePath,
					"isDir": true,
					"drive": true,
				})
			}
		}
		writeJSON(w, map[string]interface{}{
			"code":   0,
			"path":   reqPath,
			"parent": "",
			"dirs":   drives,
			"isRoot": true,
		})
		return
	}

	entries, err := os.ReadDir(reqPath)
	if err != nil {
		// 目录不存在时尝试自动创建（日志目录通常是安全可创建的）
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(reqPath, 0755); mkErr == nil {
				entries, err = os.ReadDir(reqPath)
			} else {
				// 创建失败则向上回退到最近的已存在父目录
				fallback := reqPath
				for i := 0; i < 32; i++ {
					parent := filepath.Dir(fallback)
					if parent == fallback || parent == "" || parent == "." {
						break
					}
					if _, statErr := os.Stat(parent); statErr == nil {
						reqPath = parent
						entries, err = os.ReadDir(reqPath)
						break
					}
					fallback = parent
				}
			}
		}
		if err != nil {
			writeJSON(w, map[string]interface{}{
				"code": 1,
				"msg":  "无法读取目录: " + err.Error(),
				"path": reqPath,
			})
			return
		}
	}

	var dirs []map[string]interface{}
	excludedDirs := map[string]bool{
		"proc": true, "sys": true, "dev": true,
		"boot": true, "sbin": true, "lib": true, "lib64": true,
		"lost+found": true, "snap": true,
		"root": true, "ssh": true, "ssl": true, "pam.d": true, "security": true,
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() {
			continue
		}
		if excludedDirs[name] {
			continue
		}
		dirs = append(dirs, map[string]interface{}{
			"name":  name,
			"path":  filepath.Join(reqPath, name),
			"isDir": true,
		})
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i]["name"].(string)) < strings.ToLower(dirs[j]["name"].(string))
	})

	parent := filepath.Dir(reqPath)
	if parent == reqPath {
		parent = ""
	}

	writeJSON(w, map[string]interface{}{
		"code":   0,
		"path":   reqPath,
		"parent": parent,
		"dirs":   dirs,
		"isRoot": isRoot,
	})
}

// joinPosix 使用 / 拼接路径，避免 Windows 下 filepath.Join 产生反斜杠
func joinPosix(parts ...string) string {
	return strings.Join(parts, "/")
}

// #2 docker exec 容器名校验
// 容器名直接拼入 docker exec 命令，必须严格白名单
var containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)

// validateContainerName 校验容器名合法性，避免 docker exec 参数注入
func validateContainerName(name string) error {
	if name == "" {
		return fmt.Errorf("容器名不能为空")
	}
	if len(name) > 64 {
		return fmt.Errorf("容器名长度不能超过64个字符")
	}
	if !containerNameRegex.MatchString(name) {
		return fmt.Errorf("容器名包含非法字符: %s", name)
	}
	return nil
}

// #2 日志路径白名单：读取容器/本地日志时只允许常见日志目录
// 可通过环境变量 DM_LOG_PATH_PREFIXES 扩展（逗号分隔绝对路径前缀）
var defaultLogPathPrefixes = []string{
	"/var/log/",
	"/var/lib/mysql/",
	"/var/lib/postgresql/",
	"/var/lib/redis/",
	"/data/",
	"/opt/",
	"/tmp/",
	"/vol1/",
	"/vol2/",
	"/vol3/",
	"/vol4/",
	"/vol5/",
}

func validateLogPath(p string) error {
	if p == "" {
		return fmt.Errorf("日志路径不能为空")
	}
	// Windows 路径直接放行（本地配置路径场景）
	if runtime.GOOS == "windows" && (len(p) >= 2 && p[1] == ':') {
		return nil
	}
	prefixes := defaultLogPathPrefixes
	if env := os.Getenv("DM_LOG_PATH_PREFIXES"); env != "" {
		for _, x := range strings.Split(env, ",") {
			x = strings.TrimSpace(x)
			if x != "" {
				prefixes = append(prefixes, x)
			}
		}
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(p, prefix) {
			return nil
		}
	}
	return fmt.Errorf("日志路径不在白名单内: %s", p)
}

// validateFilePathUnderDir 检查目标路径是否在指定根目录之下。
// - 用 EvalSymlinks 解析符号链接后比较，防止 symlink 逃逸
// - 拒绝 ".." 等相对路径组件
// - Linux: 根必须为绝对路径
// - Windows: 允许盘符路径
func validateFilePathUnderDir(targetPath, rootDir string) error {
	if targetPath == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	if rootDir == "" {
		return fmt.Errorf("根目录未指定")
	}
	// 拼接并规范化
	cleaned := filepath.Clean(targetPath)
	rootCleaned := filepath.Clean(rootDir)
	// 解析符号链接
	resolvedTarget, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		// 文件不存在时直接用 cleaned 比较（不强制要求预存）
		resolvedTarget = cleaned
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootCleaned)
	if err != nil {
		resolvedRoot = rootCleaned
	}
	// 大小写处理（Windows 路径不区分大小写）
	if runtime.GOOS == "windows" {
		resolvedTarget = strings.ToLower(resolvedTarget)
		resolvedRoot = strings.ToLower(resolvedRoot)
	}
	// 必须有合法前缀
	rel, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil {
		return fmt.Errorf("路径不在允许的目录范围内")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("路径越界: %s", targetPath)
	}
	return nil
}

// readContainerFile 通过 docker exec 从容器内读取文件内容（最后512KB）
func readContainerFile(containerName, filePath string) ([]byte, error) {
	if err := validateContainerName(containerName); err != nil {
		return nil, fmt.Errorf("容器名校验失败: %w", err)
	}
	if !filepath.IsAbs(filePath) {
		return nil, fmt.Errorf("容器内文件路径必须为绝对路径: %s", filePath)
	}
	// 拒绝明显危险的路径特征
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("文件路径不允许包含 ..: %s", filePath)
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("未找到docker命令")
	}

	// #2 用 -- 分隔 docker 参数与子命令参数，避免 filePath 以 - 开头被解释为选项
	out, err := exec.Command(dockerPath, "exec", containerName, "--", "tail", "-c", "524288", filePath).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
	}
	return out, nil
}

// findContainerByPort 通过 docker ps 根据端口映射查找容器名
func findContainerByPort(host string, port int) string {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return ""
	}

	out, err := exec.Command(dockerPath, "ps", "--format", "{{.Names}}\t{{.Ports}}").CombinedOutput()
	if err != nil {
		return ""
	}

	portStr := fmt.Sprintf(":%d->", port)
	if host != "" && host != "127.0.0.1" && host != "localhost" {
		portStr = fmt.Sprintf("%s:%d->", host, port)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		ports := parts[1]
		if strings.Contains(ports, portStr) {
			return name
		}
	}
	return ""
}

// readLocalOrContainerFile 读取日志文件，优先本地，失败后尝试 docker exec
// containerName 为空时自动通过 host:port 查找 Docker 容器
func readLocalOrContainerFile(containerName, filePath string, host string, port int) ([]byte, error) {
	// #2 容器名校验（如提供）
	if containerName != "" {
		if err := validateContainerName(containerName); err != nil {
			return nil, fmt.Errorf("容器名校验失败: %w", err)
		}
	}
	// 文件路径基本检查：拒绝 ".."
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("文件路径不允许包含 ..: %s", filePath)
	}
	// 路径白名单校验：只允许读常见日志目录，避免误读敏感路径
	if err := validateLogPath(filePath); err != nil {
		return nil, fmt.Errorf("日志路径校验失败: %w", err)
	}

	// 先尝试本地读取（仅对 Windows 路径或相对路径尝试）
	// Linux 绝对路径在 Windows 上无法直接打开，跳过本地读取直接走 docker
	shouldTryLocal := true
	if runtime.GOOS == "windows" && strings.HasPrefix(filePath, "/") {
		shouldTryLocal = false
	}

	if shouldTryLocal {
		f, err := os.Open(filePath)
		if err == nil {
			defer f.Close()
			fi, statErr := f.Stat()
			if statErr != nil {
				return nil, statErr
			}
			const maxReadSize int64 = 512 * 1024
			if fi.Size() > maxReadSize {
				offset := fi.Size() - maxReadSize
				if _, seekErr := f.Seek(offset, io.SeekStart); seekErr != nil {
					return nil, seekErr
				}
				data, readErr := io.ReadAll(f)
				if readErr != nil {
					return nil, readErr
				}
				if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
					data = data[idx+1:]
				}
				return data, nil
			}
			return io.ReadAll(f)
		}
	}

	// 本地读取失败，尝试 docker exec
	if containerName == "" {
		containerName = findContainerByPort(host, port)
	}
	if containerName != "" {
		// 二次校验 findContainerByPort 的返回值
		if err := validateContainerName(containerName); err != nil {
			return nil, fmt.Errorf("查找到的容器名非法: %w", err)
		}
		return readContainerFile(containerName, filePath)
	}

	return nil, fmt.Errorf("无法读取日志文件: %s", filePath)
}

// parseLogLines 将原始日志文本按行解析为结构化日志条目
func parseLogLines(data []byte, maxLines int) []map[string]string {
	lines := strings.Split(string(data), "\n")
	start := 0
	if len(lines) > maxLines {
		start = len(lines) - maxLines
	}
	var result []map[string]string
	for _, line := range lines[start:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, map[string]string{
			"time":    "",
			"level":   "Note",
			"message": line,
		})
	}
	return result
}
