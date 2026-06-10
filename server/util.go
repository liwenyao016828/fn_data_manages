package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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

func fsBrowseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		if runtime.GOOS == "windows" {
			reqPath = "C:\\"
		} else {
			reqPath = "/"
		}
	}

	reqPath = filepath.Clean(reqPath)
	if runtime.GOOS == "windows" && len(reqPath) == 2 && reqPath[1] == ':' {
		reqPath = reqPath + "\\"
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
		writeJSON(w, map[string]interface{}{
			"code": 1,
			"msg":   "无法读取目录: " + err.Error(),
			"path":  reqPath,
		})
		return
	}

	var dirs []map[string]interface{}
	excludedDirs := map[string]bool{
		"proc": true, "sys": true, "dev": true,
		"boot": true, "sbin": true, "lib": true, "lib64": true,
		"lost+found": true, "snap": true,
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