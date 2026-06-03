package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func scheduledBackupDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := r.URL.Path
	idStr := path[len("/api/backups/scheduled/"):]
	if idStr == "" {
		return
	}

	if r.Method == "DELETE" {
		deleteScheduledBackup(w, r, idStr)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func doSystemBackup() string {
	dataDir := getDataDir()
	backupData := map[string]interface{}{
		"type":    "system_backup",
		"created": time.Now().Format("2006-01-02 15:04:05"),
		"version": "1.0",
		"data":    map[string]json.RawMessage{},
	}
	files := []string{"databases.json", "remote_servers.json", "backups.json", "scheduled_backups.json"}
	dataMap := make(map[string]json.RawMessage)
	for _, f := range files {
		content, err := os.ReadFile(filepath.Join(dataDir, f))
		if err != nil {
			continue
		}
		dataMap[f] = json.RawMessage(content)
	}
	backupData["data"] = dataMap
	result, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(result)
}

func restoreSystemBackup(content string) error {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") {
		return restoreSystemBackupJSON(content)
	}
	return restoreSystemBackupLegacy(content)
}

func restoreSystemBackupJSON(content string) error {
	var backupData struct {
		Type    string                     `json:"type"`
		Created string                     `json:"created"`
		Version string                     `json:"version"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(content), &backupData); err != nil {
		return fmt.Errorf("解析系统备份失败: %w", err)
	}
	if backupData.Type != "system_backup" {
		return fmt.Errorf("无效的系统备份格式")
	}
	dataDir := getDataDir()
	for filename, rawContent := range backupData.Data {
		if !json.Valid(rawContent) {
			continue
		}
		targetPath := filepath.Join(dataDir, filename)
		if err := atomicWriteFile(targetPath, rawContent, 0644); err != nil {
			return fmt.Errorf("恢复 %s 失败: %w", filename, err)
		}
	}
	loadData()
	sysLogInfo("BACKUP", "系统配置恢复成功，已重新加载配置")
	return nil
}

func restoreSystemBackupLegacy(content string) error {
	dataDir := getDataDir()
	files := []string{"databases.json", "remote_servers.json", "backups.json", "scheduled_backups.json"}
	for _, filename := range files {
		startMark := fmt.Sprintf("-- ## %s START", filename)
		endMark := fmt.Sprintf("-- ## %s END", filename)
		startIdx := strings.Index(content, startMark)
		if startIdx < 0 {
			continue
		}
		jsonStart := strings.Index(content[startIdx:], "\n")
		if jsonStart < 0 {
			continue
		}
		jsonStart += startIdx + 1
		endIdx := strings.Index(content[jsonStart:], endMark)
		if endIdx < 0 {
			continue
		}
		jsonContent := strings.TrimSpace(content[jsonStart : jsonStart+endIdx])
		if !json.Valid([]byte(jsonContent)) {
			continue
		}
		targetPath := filepath.Join(dataDir, filename)
		if err := atomicWriteFile(targetPath, []byte(jsonContent), 0644); err != nil {
			return fmt.Errorf("恢复 %s 失败: %w", filename, err)
		}
	}
	loadData()
	sysLogInfo("BACKUP", "系统配置恢复成功，已重新加载配置")
	return nil
}

func scheduledBackupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case "GET":
		listScheduledBackups(w, r)
	case "POST":
		createScheduledBackup(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listScheduledBackups(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr != "" {
		id, _ := strconv.ParseUint(idStr, 10, 32)
		mutex.Lock()
		for _, s := range scheduledBackups {
			if s.ID == uint(id) {
				mutex.Unlock()
				writeJSON(w, map[string]interface{}{"code": 0, "data": s})
				return
			}
		}
		mutex.Unlock()
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "not found"})
		return
	}

	mutex.Lock()
	result := make([]ScheduledBackup, len(scheduledBackups))
	copy(result, scheduledBackups)
	mutex.Unlock()

	writeJSON(w, map[string]interface{}{"code": 0, "data": result})
}

func createScheduledBackup(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	idFromReq := getUint(req, "id")
	if idFromReq > 0 {
		mutex.Lock()
		for i, s := range scheduledBackups {
			if s.ID == idFromReq {
				if name, ok := req["name"].(string); ok && name != "" {
					scheduledBackups[i].Name = name
				}
				if cron, ok := req["cron"].(string); ok {
					scheduledBackups[i].Cron = cron
				}
				if label, ok := req["label"].(string); ok {
					scheduledBackups[i].Label = label
				}
				if enabled, ok := req["enabled"].(bool); ok {
					scheduledBackups[i].Enabled = enabled
				}
				if rc, ok := req["retainCount"].(float64); ok {
					scheduledBackups[i].RetainCount = int(rc)
				}
				mutex.Unlock()
				saveData()
				writeJSON(w, map[string]interface{}{"code": 0, "msg": "updated"})
				return
			}
		}
		mutex.Unlock()
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "not found"})
		return
	}

	name := getString(req, "name")
	if name == "" {
		name = fmt.Sprintf("定时备份_%s", time.Now().Format("20060102_150405"))
	}

	s := ScheduledBackup{
		ID:          nextSchedID,
		Name:        name,
		BackupLevel: getString(req, "backupLevel"),
		ServerID:    getUint(req, "serverId"),
		Source:      getString(req, "source"),
		Database:    getString(req, "database"),
		Cron:        getString(req, "cron"),
		Label:       getString(req, "label"),
		Enabled:     true,
		RetainCount: getInt(req, "retainCount"),
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
	}
	if s.RetainCount == 0 {
		s.RetainCount = 7
	}

	// 非 root 用户不允许备份全部数据库
	if s.Database == "__ALL__" && s.BackupLevel != "system" && s.BackupLevel != "redis" {
		server := findAnyServer(s.ServerID, s.Source)
		if server != nil && server.Username != "root" {
			writeJSON(w, map[string]interface{}{"code": 403, "msg": "当前账号权限不足，仅 root 用户可以备份全部数据库"})
			return
		}
	}

	mutex.Lock()
	nextSchedID++
	scheduledBackups = append(scheduledBackups, s)
	mutex.Unlock()

	saveData()
	sysLogInfo("BACKUP", fmt.Sprintf("创建定时备份计划: %s (%s, %s)", s.Name, s.Cron, s.BackupLevel))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "success", "data": s})
}

func deleteScheduledBackup(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid id"})
		return
	}

	mutex.Lock()
	for i, s := range scheduledBackups {
		if s.ID == uint(id) {
			scheduledBackups = append(scheduledBackups[:i], scheduledBackups[i+1:]...)
			mutex.Unlock()
			saveData()
			sysLogInfo("BACKUP", fmt.Sprintf("删除定时备份计划: %s", s.Name))
			writeJSON(w, map[string]interface{}{"code": 0, "msg": "deleted"})
			return
		}
	}
	mutex.Unlock()
	writeJSON(w, map[string]interface{}{"code": 404, "msg": "not found"})
}

func checkScheduledBackups() {
	mutex.Lock()
	now := time.Now()
	var toRun []ScheduledBackup
	for _, s := range scheduledBackups {
		if !s.Enabled {
			continue
		}
		toRun = append(toRun, s)
	}
	mutex.Unlock()

	for _, s := range toRun {
		lastRun, _ := time.Parse("2006-01-02 15:04:05", s.LastRun)
		period := parseSchedule(s.Cron)
		if period > 0 && now.Sub(lastRun) >= period {
			runScheduledBackup(s)
		}
	}

	cleanupExpiredBackups()
}

func parseSchedule(cron string) time.Duration {
	switch cron {
	case "daily":
		return 24 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	case "monthly":
		return 30 * 24 * time.Hour
	default:
		if h, err := strconv.Atoi(cron); err == nil && h > 0 {
			return time.Duration(h) * time.Hour
		}
	}
	return 0
}

func runScheduledBackup(s ScheduledBackup) {
	sysLogInfo("BACKUP", fmt.Sprintf("执行定时备份: %s", s.Name))
	name := fmt.Sprintf("auto_%s_%s", s.Name, time.Now().Format("20060102_150405"))
	bakDir := getDataDir() + "/backups"
	fmt.Printf("Backup directory: %s\n", bakDir)
	os.MkdirAll(bakDir, 0755)

	var content string
	fileName := name + ".json"
	backupType := s.BackupLevel

	if s.BackupLevel == "system" {
		content = doSystemBackup()
		fileName = name + ".json"
	} else if s.BackupLevel == "redis" {
		server := findRedisServer(s.ServerID, s.Source)
		if server != nil {
			if server.Host != "127.0.0.1" && server.Host != "localhost" {
				sysLogWarn("BACKUP", "定时备份Redis远程实例不支持RDB备份")
				content = fmt.Sprintf("-- Redis Auto Backup FAILED: remote Redis not supported\n")
				fileName = name + ".rdb"
			} else {
				conn, err := openRedis(server)
				if err == nil {
					redisDo(conn, "SAVE")
					conn.Close()
					rdbFileName, rdbErr := doRedisRdbCopy(server, name, bakDir)
					if rdbErr != nil {
						sysLogError("BACKUP", fmt.Sprintf("定时备份Redis RDB复制失败 (连接: %s:%d)", server.Host, server.Port))
						content = fmt.Sprintf("-- Redis Auto Backup FAILED: %s\n", rdbErr.Error())
						fileName = name + ".rdb"
					} else {
						fileName = rdbFileName
					}
				} else {
					sysLogError("BACKUP", fmt.Sprintf("定时备份Redis连接失败 (连接: %s:%d)", server.Host, server.Port))
					content = fmt.Sprintf("-- Redis Auto Backup FAILED: %s\n", err.Error())
					fileName = name + ".rdb"
				}
			}
		} else {
			sysLogWarn("BACKUP", fmt.Sprintf("定时备份Redis服务器未找到: ID=%d", s.ServerID))
			content = fmt.Sprintf("-- Redis Auto Backup FAILED: server not found\n")
			fileName = name + ".rdb"
		}
	} else {
		server := findAnyServer(s.ServerID, s.Source)
		if server != nil {
			var fileSize int64
			var err error
			fileName, fileSize, err = doMySQLBackup(server, s.Database, bakDir, name)
			if err != nil {
				sysLogError("BACKUP", fmt.Sprintf("定时备份MySQL失败 (连接: %s:%d)", server.Host, server.Port))
				content = fmt.Sprintf("-- MySQL Auto Backup FAILED: %s\n-- Error: %s\n", s.Name, err.Error())
				fileName = name + ".sql"
			}
			_ = fileSize
		} else {
			sysLogWarn("BACKUP", fmt.Sprintf("定时备份MySQL服务器未找到: ID=%d", s.ServerID))
			content = fmt.Sprintf("-- MySQL Auto Backup FAILED: server not found\n")
			fileName = name + ".sql"
		}
	}
	if content != "" {
		os.WriteFile(filepath.Join(bakDir, fileName), []byte(content), 0644)
	}

	backupStatus := "success"
	if content != "" && strings.Contains(content, "FAILED") {
		backupStatus = "failed"
	}

	filePath := filepath.Join(bakDir, fileName)
	var fileSize int64
	if fi, err := os.Stat(filePath); err == nil {
		fileSize = fi.Size()
	}

	newBak := Backup{
		ID:          nextBackupID,
		Name:        name,
		Type:        backupType,
		BackupLevel: backupType,
		ServerID:    s.ServerID,
		Database:    s.Database,
		FileName:    fileName,
		FileSize:    fileSize,
		Status:      backupStatus,
		Description: "定时备份",
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		BackupType:  "scheduled",
		Source:      s.Source,
	}
	normalizeBackup(&newBak)

	mutex.Lock()
	nextBackupID++
	backups = append(backups, newBak)

	for i := range scheduledBackups {
		if scheduledBackups[i].ID == s.ID {
			scheduledBackups[i].LastRun = time.Now().Format("2006-01-02 15:04:05")
			break
		}
	}

	if s.RetainCount > 0 {
		count := 0
		for _, b := range backups {
			if b.Database == s.Database && b.BackupType == "scheduled" && b.ServerID == s.ServerID && b.BackupLevel == s.BackupLevel {
				count++
			}
		}
		toDelete := count - s.RetainCount
		if toDelete > 0 {
			deleted := 0
			newBackups := make([]Backup, 0, len(backups))
			for _, b := range backups {
				if b.Database == s.Database && b.BackupType == "scheduled" && b.ServerID == s.ServerID && b.BackupLevel == s.BackupLevel && deleted < toDelete {
					os.Remove(filepath.Join(bakDir, b.FileName))
					deleted++
					continue
				}
				newBackups = append(newBackups, b)
			}
			backups = newBackups
		}
	}
	mutex.Unlock()
	saveData()
}

func runScheduledBackupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	id := getUint(req, "id")
	if id == 0 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "id required"})
		return
	}

	mutex.Lock()
	var target *ScheduledBackup
	for i := range scheduledBackups {
		if scheduledBackups[i].ID == id {
			target = &scheduledBackups[i]
			break
		}
	}
	mutex.Unlock()

	if target == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "计划不存在"})
		return
	}

	go runScheduledBackup(*target)

	writeJSON(w, map[string]interface{}{"code": 0, "msg": "已开始执行备份"})
}

func cleanupExpiredBackups() {
	dataDir := getDataDir()
	configPath := filepath.Join(dataDir, "backup_retention.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return
	}

	daysVal, ok := config["days"]
	if !ok {
		return
	}

	if strVal, ok := daysVal.(string); ok && strVal == "never" {
		return
	}

	var days int
	switch v := daysVal.(type) {
	case float64:
		days = int(v)
	case string:
		days, _ = strconv.Atoi(v)
	default:
		return
	}

	if days <= 0 {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -days)

	mutex.Lock()
	bakDir := dataDir + "/backups"
	var remaining []Backup
	deleted := 0
	for _, b := range backups {
		createdAt, err := time.Parse("2006-01-02 15:04:05", b.CreatedAt)
		if err != nil || createdAt.After(cutoff) {
			remaining = append(remaining, b)
			continue
		}
		os.Remove(filepath.Join(bakDir, b.FileName))
		deleted++
	}
	if deleted > 0 {
		backups = remaining
		mutex.Unlock()
		saveData()
		sysLogInfo("BACKUP", fmt.Sprintf("清理了 %d 个过期备份（保留天数: %d）", deleted, days))
	} else {
		mutex.Unlock()
	}
}
