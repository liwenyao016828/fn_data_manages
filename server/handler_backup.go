package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func backupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case "GET":
		listBackups(w, r)
	case "POST":
		createBackup(w, r)
	case "DELETE":
		batchDeleteBackups(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func backupDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := r.URL.Path
	idStr := path[len("/api/backups/"):]
	if idStr == "" || idStr == "import" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid backup id"})
		return
	}

	switch r.Method {
	case "DELETE":
		deleteBackup(w, r, idStr)
	case "GET":
		downloadBackup(w, r, idStr)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func backupImportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	importBackup(w, r)
}

func normalizeBackup(b *Backup) {
	if b.BackupLevel == "" || b.BackupLevel == "database" {
		if b.Type == "redis" {
			b.BackupLevel = "redis"
		} else {
			b.BackupLevel = "mysql"
		}
	}
	switch b.BackupLevel {
	case "redis":
		if b.Type == "" || b.Type == "mysql" {
			b.Type = "redis"
		}
	case "system":
		if b.Type == "" {
			b.Type = "system"
		}
	default:
		if b.Type == "" || b.Type == "redis" || b.Type == "system" {
			b.Type = "mysql"
		}
	}
}

func listBackups(w http.ResponseWriter, r *http.Request) {
	dbFilter := r.URL.Query().Get("database")
	levelFilter := r.URL.Query().Get("level")
	serverIDFilter := r.URL.Query().Get("server_id")
	sourceFilter := r.URL.Query().Get("source")
	searchQuery := r.URL.Query().Get("search")
	sortField := r.URL.Query().Get("sort_field")
	sortOrder := r.URL.Query().Get("sort_order")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	pageSize := 20
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
		pageSize = ps
	}
	if pageSize > 100 {
		pageSize = 100
	}

	mutex.Lock()
	var result []Backup
	for _, b := range backups {
		backup := b
		normalizeBackup(&backup)
		if dbFilter != "" && backup.Database != dbFilter {
			continue
		}
		if levelFilter != "" && backup.BackupLevel != levelFilter {
			continue
		}
		if serverIDFilter != "" {
			sid := uint(0)
			fmt.Sscanf(serverIDFilter, "%d", &sid)
			if sid > 0 && backup.ServerID != sid {
				continue
			}
			if sourceFilter == "remote" {
				server := findRemoteServer(sid)
				if server == nil {
					continue
				}
			}
		}
		if searchQuery != "" {
			q := strings.ToLower(searchQuery)
			if !strings.Contains(strings.ToLower(backup.Name), q) &&
				!strings.Contains(strings.ToLower(backup.Database), q) {
				continue
			}
		}
		result = append(result, backup)
	}

	for i := range result {
		bakDir := getDataDir() + "/backups"
		path := filepath.Join(bakDir, result[i].FileName)
		if fi, err := os.Stat(path); err == nil {
			result[i].FileSize = fi.Size()
		}
		// 显示「全部」
		if result[i].Database == "__ALL__" {
			result[i].Database = "全部"
		}
	}
	total := len(result)

	if sortField != "" {
		sort.Slice(result, func(i, j int) bool {
			var less bool
			switch sortField {
			case "name":
				less = result[i].Name < result[j].Name
			case "backupLevel":
				less = result[i].BackupLevel < result[j].BackupLevel
			case "backupType":
				less = result[i].BackupType < result[j].BackupType
			case "fileSize":
				less = result[i].FileSize < result[j].FileSize
			case "status":
				less = result[i].Status < result[j].Status
			case "createdAt":
				less = result[i].CreatedAt < result[j].CreatedAt
			case "database":
				less = result[i].Database < result[j].Database
			default:
				less = result[i].CreatedAt < result[j].CreatedAt
			}
			if sortOrder == "desc" {
				return !less
			}
			return less
		})
	}

	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	var pageItems []Backup
	if start < total {
		pageItems = result[start:end]
	} else {
		pageItems = []Backup{}
	}

	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"items":    pageItems,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func createBackup(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	name := getString(req, "name")
	if name == "" {
		name = fmt.Sprintf("backup_%s", time.Now().Format("20060102_150405"))
	}

	newBackup := Backup{
		ID:          nextBackupID,
		Name:        name,
		Type:        getString(req, "type"),
		ServerID:    getUint(req, "serverId"),
		Host:        getString(req, "host"),
		Port:        getInt(req, "port"),
		Database:    getString(req, "database"),
		FileName:    name + ".sql",
		Status:      "success",
		Description: getString(req, "description"),
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		BackupType:  "backup",
		BackupLevel: getString(req, "backupLevel"),
	}
	normalizeBackup(&newBackup)

	source := getString(req, "source")
	if source == "" {
		source = r.URL.Query().Get("source")
	}
	if source == "" {
		source = "local"
	}
	newBackup.Source = source

	// 直接使用传入的数据库名，避免连接超时
	if newBackup.BackupLevel != "system" && newBackup.BackupLevel != "redis" {
		// 如果传入的 database 是连接名或空，尝试从连接配置中获取
		if newBackup.Database == "" || strings.Contains(newBackup.Database, ":") {
			if source != "remote" {
				mutex.Lock()
				for _, db := range databases {
					if db.ID == newBackup.ServerID {
						if db.Database != "" && !strings.Contains(db.Database, ":") {
							newBackup.Database = db.Database
						}
						break
					}
				}
				mutex.Unlock()
			}
		}
		// 最后如果还是无效，返回错误（避免 mysqldump 卡住）
		if newBackup.Database == "" || strings.Contains(newBackup.Database, ":") {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "请选择具体要备份的MySQL数据库名"})
			return
		}
		// 非 root 用户不允许备份全部数据库
		if newBackup.Database == "__ALL__" {
			server := findAnyServer(newBackup.ServerID, source)
			if server != nil && server.Username != "root" {
				writeJSON(w, map[string]interface{}{"code": 403, "msg": "当前账号权限不足，仅 root 用户可以备份全部数据库"})
				return
			}
		}
	}

	bakDir := getDataDir() + "/backups"
	os.MkdirAll(bakDir, 0755)

	if newBackup.BackupLevel == "system" {
		content := doSystemBackup()
		newBackup.FileName = name + ".json"
		newBackup.FileSize = int64(len(content))
		if err := os.WriteFile(filepath.Join(bakDir, newBackup.FileName), []byte(content), 0644); err != nil {
			sysLogError("BACKUP", "写入系统备份文件失败")
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "写入系统备份文件失败: " + err.Error()})
			return
		}
	} else if newBackup.BackupLevel == "redis" {
		serverID := newBackup.ServerID
		if serverID == 0 {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "Redis备份需要指定server_id"})
			return
		}
		server := findRedisServer(serverID, source)
		if server == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "Redis服务器不存在"})
			return
		}
		if server.Host != "127.0.0.1" && server.Host != "localhost" {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "远程Redis实例不支持文件系统备份，仅支持本地Redis"})
			return
		}
		conn, err := openRedis(server)
		if err != nil {
			sysLogError("BACKUP", fmt.Sprintf("备份Redis连接失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
			return
		}
		_, err = redisDo(conn, "SAVE")
		conn.Close()
		if err != nil {
			sysLogError("BACKUP", fmt.Sprintf("备份Redis SAVE失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "Redis SAVE失败: " + err.Error()})
			return
		}
		rdbFileName, err := doRedisRdbCopy(server, name, bakDir)
		if err != nil {
			sysLogError("BACKUP", fmt.Sprintf("备份Redis RDB复制失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "复制RDB文件失败: " + err.Error()})
			return
		}
		newBackup.FileName = rdbFileName
		rdbFilePath := filepath.Join(bakDir, rdbFileName)
		if fi, err := os.Stat(rdbFilePath); err == nil {
			newBackup.FileSize = fi.Size()
		}
	} else {
		server := findAnyServer(newBackup.ServerID, source)
		if server == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "服务器不存在"})
			return
		}
		fileName, fileSize, err := doMySQLBackup(server, newBackup.Database, bakDir, name)
		if err != nil {
			sysLogError("BACKUP", fmt.Sprintf("备份MySQL失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "备份失败: " + err.Error()})
			return
		}
		newBackup.FileName = fileName
		newBackup.FileSize = fileSize
	}

	mutex.Lock()
	nextBackupID++
	backups = append(backups, newBackup)
	mutex.Unlock()

	saveData()

	sysLogInfo("BACKUP", fmt.Sprintf("创建备份 %s（级别=%s，数据库=%s）", newBackup.Name, newBackup.BackupLevel, newBackup.Database))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success", "data": newBackup})
}

func deleteBackup(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "invalid id"})
		return
	}

	mutex.Lock()
	found := false
	var backupName, backupLevel, backupDatabase string
	for i, b := range backups {
		if b.ID == uint(id) {
			bakDir := getDataDir() + "/backups"
			os.Remove(filepath.Join(bakDir, b.FileName))
			backupName = b.Name
			backupLevel = b.BackupLevel
			backupDatabase = b.Database
			backups = append(backups[:i], backups[i+1:]...)
			found = true
			break
		}
	}
	mutex.Unlock()

	if !found {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 404, "msg": "not found"})
		return
	}

	saveData()

	sysLogInfo("BACKUP", fmt.Sprintf("删除备份 %s（%s，%s）", backupName, backupLevel, backupDatabase))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
}

func batchDeleteBackups(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "ids required"})
		return
	}

	bakDir := getDataDir() + "/backups"
	deletedCount := 0
	notFoundCount := 0

	mutex.Lock()
	idSet := make(map[uint]bool)
	for _, id := range req.IDs {
		idSet[id] = true
	}
	var remaining []Backup
	for _, b := range backups {
		if idSet[b.ID] {
			os.Remove(filepath.Join(bakDir, b.FileName))
			deletedCount++
		} else {
			remaining = append(remaining, b)
		}
	}
	backups = remaining
	notFoundCount = len(req.IDs) - deletedCount
	mutex.Unlock()

	saveData()

	msg := fmt.Sprintf("成功删除 %d 个备份", deletedCount)
	if notFoundCount > 0 {
		msg = fmt.Sprintf("成功删除 %d 个备份，%d 个未找到", deletedCount, notFoundCount)
	}
	sysLogInfo("BACKUP", msg)
	writeJSON(w, map[string]interface{}{"code": 0, "msg": msg, "deleted": deletedCount})
}

func downloadBackup(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "invalid id"})
		return
	}

	mutex.Lock()
	var fileName string
	for _, b := range backups {
		if b.ID == uint(id) {
			fileName = b.FileName
			break
		}
	}
	mutex.Unlock()

	if fileName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 404, "msg": "not found"})
		return
	}

	bakDir := getDataDir() + "/backups"
	filePath := filepath.Join(bakDir, fileName)
	http.ServeFile(w, r, filePath)
}

func importBackup(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "文件解析失败: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "请选择要导入的备份文件"})
		return
	}
	defer file.Close()

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
		if len(name) > 100 {
			name = name[:100]
		}
	}
	if name == "" {
		name = fmt.Sprintf("import_%s", time.Now().Format("20060102_150405"))
	}

	serverIdStr := r.FormValue("serverId")
	serverId, _ := strconv.ParseUint(serverIdStr, 10, 32)

	fileName := fmt.Sprintf("%s_%s", time.Now().Format("20060102_150405"), filepath.Base(header.Filename))
	if len(fileName) > 200 {
		fileName = fileName[:200]
	}
	fileName = strings.ReplaceAll(fileName, "..", "")
	bakDir := getDataDir() + "/backups"
	os.MkdirAll(bakDir, 0755)

	dst, err := os.Create(filepath.Join(bakDir, fileName))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 500, "msg": "保存文件失败"})
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 500, "msg": "写入文件失败"})
		return
	}

	dbType := r.FormValue("type")
	if dbType == "" {
		dbType = "mysql"
	}
	backupLevel := r.FormValue("backupLevel")
	if backupLevel == "" {
		backupLevel = dbType
	}
	newBackup := Backup{
		ID:          nextBackupID,
		Name:        name,
		Type:        dbType,
		ServerID:    uint(serverId),
		Database:    r.FormValue("database"),
		FileName:    fileName,
		FileSize:    written,
		Status:      "success",
		Description: "导入备份",
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		BackupType:  "import",
		BackupLevel: backupLevel,
	}

	mutex.Lock()
	nextBackupID++
	backups = append(backups, newBackup)
	mutex.Unlock()

	saveData()

	sysLogInfo("BACKUP", fmt.Sprintf("导入备份: %s", newBackup.Name))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "导入成功", "data": newBackup})
}

func backupRestoreHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	restoreBackup(w, r)
}

func restoreBackup(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	backupID := getUint(req, "backup_id")
	if backupID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "backup_id required"})
		return
	}

	mutex.Lock()
	var backup *Backup
	for i := range backups {
		if backups[i].ID == backupID {
			backup = &backups[i]
			break
		}
	}
	mutex.Unlock()

	if backup == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 404, "msg": "backup not found"})
		return
	}

	normalizeBackup(backup)

	bakDir := getDataDir() + "/backups"
	filePath := filepath.Join(bakDir, backup.FileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "备份文件不存在"})
		return
	}

	switch backup.BackupLevel {
	case "system":
		content, err := os.ReadFile(filePath)
		if err != nil {
			sysLogError("BACKUP", "恢复系统备份读取失败")
			writeJSON(w, map[string]interface{}{"code": 500, "msg": "读取备份文件失败: " + err.Error()})
			return
		}
		if err := restoreSystemBackup(string(content)); err != nil {
			sysLogError("BACKUP", "系统备份恢复失败")
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "系统恢复失败: " + err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "系统配置恢复成功"})

	case "redis":
		source := getString(req, "source")
		if source == "" {
			source = r.URL.Query().Get("source")
		}
		if source == "" {
			source = backup.Source
		}
		if source == "" {
			source = "local"
		}
		server := findRedisServer(backup.ServerID, source)
		if server == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "Redis服务器不存在"})
			return
		}

		if server.Host != "127.0.0.1" && server.Host != "localhost" {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "远程Redis实例不支持文件系统恢复，请使用 Redis CLI 手动恢复"})
			return
		}

		input, err := os.ReadFile(filePath)
		if err != nil {
			sysLogError("BACKUP", "恢复Redis备份读取失败")
			writeJSON(w, map[string]interface{}{"code": 500, "msg": "读取备份文件失败: " + err.Error()})
			return
		}

		conn, err := openRedis(server)
		if err != nil {
			sysLogError("BACKUP", fmt.Sprintf("恢复Redis连接失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
			return
		}

		dirResp, _ := redisDo(conn, "CONFIG", "GET", "dir")
		dbfilenameResp, _ := redisDo(conn, "CONFIG", "GET", "dbfilename")

		rdbDir := "./data"
		rdbFile := "dump.rdb"
		if dirArr, ok := dirResp.([]interface{}); ok && len(dirArr) >= 2 {
			if d, ok := dirArr[1].(string); ok {
				rdbDir = d
			}
		}
		if dbfArr, ok := dbfilenameResp.([]interface{}); ok && len(dbfArr) >= 2 {
			if f, ok := dbfArr[1].(string); ok {
				rdbFile = f
			}
		}

		conn.Close()

		destPath := filepath.Join(rdbDir, rdbFile)
		if err := os.WriteFile(destPath, input, 0644); err != nil {
			sysLogError("BACKUP", "恢复Redis写入RDB文件失败")
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "写入RDB文件失败: " + err.Error()})
			return
		}

		writeJSON(w, map[string]interface{}{"code": 0, "msg": "Redis RDB文件已恢复，请重启Redis生效"})

	default:
		content, err := os.ReadFile(filePath)
		if err != nil {
			sysLogError("BACKUP", "恢复MySQL备份读取失败")
			writeJSON(w, map[string]interface{}{"code": 500, "msg": "读取备份文件失败: " + err.Error()})
			return
		}

		source := getString(req, "source")
		if source == "" {
			source = r.URL.Query().Get("source")
		}
		if source == "" {
			source = backup.Source
		}
		if source == "" {
			source = "local"
		}
		server := findAnyServer(backup.ServerID, source)
		if server == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "服务器不存在"})
			return
		}

		resolvedDB, resolveErr := resolveDatabaseName(backup.ServerID, source, backup.Database)
		if resolveErr != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": resolveErr.Error()})
			return
		}
		backup.Database = resolvedDB

		db, err := openMySQL(server)
		if err != nil {
			sysLogError("BACKUP", fmt.Sprintf("恢复MySQL连接失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接数据库失败: " + err.Error()})
			return
		}

		if backup.Database != "" {
			_, err := db.Exec(fmt.Sprintf("USE `%s`", escapeBacktick(backup.Database)))
			if err != nil {
				db.Close()
				sysLogError("BACKUP", fmt.Sprintf("恢复MySQL选择数据库失败: %s", backup.Database))
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "选择数据库失败: " + err.Error()})
				return
			}
		}

		droppedTables, dropErr := dropAllTablesInDB(db)
		if dropErr != nil {
			db.Close()
			sysLogError("BACKUP", fmt.Sprintf("恢复MySQL清空数据库失败: %s", backup.Database))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "清空数据库失败: " + dropErr.Error()})
			return
		}
		db.Close()

		if mysqlPath, mysqlErr := exec.LookPath("mysql"); mysqlErr == nil {
			tmpFile, tmpErr := os.CreateTemp("", "restore-*.sql")
			if tmpErr == nil {
				tmpPath := tmpFile.Name()
				defer os.Remove(tmpPath)
				tmpFile.Write(content)
				tmpFile.Close()

				cnfFile, cnfErr := os.CreateTemp("", "mysql-restore-*.cnf")
				if cnfErr == nil {
					cnfPath := cnfFile.Name()
					defer os.Remove(cnfPath)
					cnfFile.WriteString(fmt.Sprintf("[client]\npassword=%s\n", server.Password))
					cnfFile.Close()

					args := []string{
						"--defaults-extra-file=" + cnfPath,
						"-h", server.Host,
						"-P", fmt.Sprintf("%d", server.Port),
						"-u", server.Username,
					}
					if backup.Database != "" {
						args = append(args, "-D", backup.Database)
					}

					sqlFile, sqlErr := os.Open(tmpPath)
					if sqlErr == nil {
						defer sqlFile.Close()
						cmd := exec.Command(mysqlPath, args...)
						cmd.Stdin = sqlFile
						output, cmdErr := cmd.CombinedOutput()
						if cmdErr == nil {
							dropInfo := ""
							if droppedTables > 0 {
								dropInfo = fmt.Sprintf("（已清除 %d 个旧表）", droppedTables)
							}
							sysLogInfo("BACKUP", fmt.Sprintf("恢复备份 %s 到 %s（mysql客户端）%s", backup.Name, backup.Database, dropInfo))
							writeJSON(w, map[string]interface{}{"code": 0, "msg": "恢复成功" + dropInfo})
							return
						}
						fmt.Printf("mysql client restore failed: %s, %s\n", cmdErr.Error(), string(output))
						sysLogError("BACKUP", fmt.Sprintf("恢复MySQL客户端执行失败: %s", backup.Database))
					}
				}
			}
		}

		db, err = openMySQL(server)
		if err != nil {
			sysLogError("BACKUP", fmt.Sprintf("恢复MySQL回退连接失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接数据库失败: " + err.Error()})
			return
		}
		defer db.Close()

		if backup.Database != "" {
			_, err := db.Exec(fmt.Sprintf("USE `%s`", escapeBacktick(backup.Database)))
			if err != nil {
				sysLogError("BACKUP", fmt.Sprintf("恢复MySQL回退选择数据库失败: %s", backup.Database))
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "选择数据库失败: " + err.Error()})
				return
			}
		}

		statements := parseSQLStatements(string(content))
		successCount := 0
		skipCount := 0
		var errorMessages []string
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if containsDangerousDDL(stmt) {
				skipCount++
				continue
			}
			_, err := db.Exec(stmt)
			if err != nil {
				errMsg := err.Error()
				if strings.Contains(errMsg, "already exists") || strings.Contains(errMsg, "Duplicate") {
					skipCount++
					continue
				}
				errorMessages = append(errorMessages, fmt.Sprintf("SQL#%d: %s", successCount+skipCount+1, errMsg))
				continue
			}
			successCount++
		}

		if len(errorMessages) > 0 && successCount == 0 {
			sysLogError("BACKUP", fmt.Sprintf("恢复MySQL所有SQL执行失败: %s", backup.Database))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": fmt.Sprintf("恢复失败，所有SQL执行出错: %s", strings.Join(errorMessages, "; "))})
			return
		}

		msg := fmt.Sprintf("恢复成功，执行了 %d 条SQL", successCount)
		if skipCount > 0 {
			msg = fmt.Sprintf("恢复成功，执行了 %d 条SQL，跳过 %d 条", successCount, skipCount)
		}
		if len(errorMessages) > 0 {
			msg = fmt.Sprintf("恢复完成，执行 %d 条SQL，%d 条出错（%s）", successCount, len(errorMessages), strings.Join(errorMessages, "; "))
		}
		if droppedTables > 0 {
			msg = fmt.Sprintf("已清除 %d 个旧表，", droppedTables) + msg
		}
		sysLogInfo("BACKUP", fmt.Sprintf("恢复备份 %s 到 %s（清除%d个旧表，执行%d条SQL，跳过%d条，错误%d条）", backup.Name, backup.Database, droppedTables, successCount, skipCount, len(errorMessages)))
		writeJSON(w, map[string]interface{}{"code": 0, "msg": msg})
	}
}

var dangerousDDLKeywords = []string{
	"DROP DATABASE",
	"DROP USER",
	"DROP VIEW",
	"DROP PROCEDURE",
	"DROP FUNCTION",
	"DROP TRIGGER",
	"DROP EVENT",
	"ALTER DATABASE",
	"ALTER USER",
	"TRUNCATE",
	"RENAME",
	"CREATE DATABASE",
	"CREATE USER",
	"CREATE TABLESPACE",
	"GRANT",
	"REVOKE",
	"KILL",
	"SHUTDOWN",
	"SET PASSWORD",
	"FLUSH PRIVILEGES",
}

func containsDangerousDDL(sql string) bool {
	upperSQL := strings.ToUpper(strings.TrimSpace(sql))
	if strings.Contains(upperSQL, "DROP TABLE IF EXISTS") {
		return false
	}
	if strings.HasPrefix(upperSQL, "DROP TABLE") {
		return true
	}
	for _, keyword := range dangerousDDLKeywords {
		if strings.Contains(upperSQL, keyword) {
			return true
		}
	}
	return false
}

func dropAllTablesInDB(db *sql.DB) (int, error) {
	rows, err := db.Query("SHOW FULL TABLES WHERE Table_type = 'BASE TABLE'")
	if err != nil {
		return 0, fmt.Errorf("获取表列表失败: %w", err)
	}
	var tables []string
	for rows.Next() {
		var tbl, tblType string
		if err := rows.Scan(&tbl, &tblType); err != nil {
			rows.Close()
			return 0, fmt.Errorf("扫描表名失败: %w", err)
		}
		tables = append(tables, tbl)
	}
	rows.Close()

	if len(tables) == 0 {
		return 0, nil
	}

	disabled := false
	_, fkErr := db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if fkErr == nil {
		disabled = true
	}

	dropped := 0
	for _, tbl := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", escapeBacktick(tbl)))
		if err == nil {
			dropped++
		}
	}

	if disabled {
		db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}

	return dropped, nil
}

func doMySQLBackup(server *RemoteServer, dbName string, bakDir string, name string) (string, int64, error) {
	if dbName == "__ALL__" {
		// 备份所有库
		return doMySQLBackupAllDatabases(server, bakDir, name)
	}

	if mysqldumpPath, err := exec.LookPath("mysqldump"); err == nil {
		tmpFile, err := os.CreateTemp("", "mysqldump-*.cnf")
		if err != nil {
			return "", 0, fmt.Errorf("创建临时配置文件失败: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		cnfContent := fmt.Sprintf("[client]\npassword=%s\n", server.Password)
		if _, err := tmpFile.WriteString(cnfContent); err != nil {
			tmpFile.Close()
			return "", 0, fmt.Errorf("写入临时配置文件失败: %w", err)
		}
		tmpFile.Close()

		args := []string{
			"--defaults-extra-file=" + tmpPath,
			"-h", server.Host,
			"-P", fmt.Sprintf("%d", server.Port),
			"-u", server.Username,
			"--single-transaction",
			"--routines",
			"--triggers",
			"--set-charset",
			dbName,
		}
		cmd := exec.Command(mysqldumpPath, args...)
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			fileName := name + ".sql"
			filePath := filepath.Join(bakDir, fileName)
			if writeErr := os.WriteFile(filePath, output, 0644); writeErr != nil {
				return "", 0, fmt.Errorf("写入备份文件失败: %w", writeErr)
			}
			return fileName, int64(len(output)), nil
		}
		if err != nil {
			fmt.Printf("mysqldump failed: %s, output: %s\n", err.Error(), string(output))
			sysLogError("BACKUP", fmt.Sprintf("mysqldump失败 (连接: %s:%d)", server.Host, server.Port))
		}
	}

	db, err := openMySQL(server)
	if err != nil {
		return "", 0, fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	if dbName != "" {
		_, err := db.Exec(fmt.Sprintf("USE `%s`", escapeBacktick(dbName)))
		if err != nil {
			return "", 0, fmt.Errorf("选择数据库失败: %w", err)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-- MySQL Backup\n-- Database: %s\n-- Created: %s\n\n", dbName, time.Now().Format("2006-01-02 15:04:05")))

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return "", 0, fmt.Errorf("获取表列表失败: %w", err)
	}
	var tables []string
	for rows.Next() {
		var tbl string
		rows.Scan(&tbl)
		tables = append(tables, tbl)
	}
	rows.Close()

	for _, tbl := range tables {
		var tableName, createStmt string
		if err := db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", escapeBacktick(tbl))).Scan(&tableName, &createStmt); err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n%s;\n\n", escapeBacktick(tbl), createStmt))

		dataRows, err := db.Query(fmt.Sprintf("SELECT * FROM `%s`", escapeBacktick(tbl)))
		if err != nil {
			continue
		}
		cols, _ := dataRows.Columns()
		for dataRows.Next() {
			vals := make([]interface{}, len(cols))
			valPtrs := make([]interface{}, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			dataRows.Scan(valPtrs...)
			var valStrs []string
			for _, v := range vals {
				if v == nil {
					valStrs = append(valStrs, "NULL")
				} else if b, ok := v.([]byte); ok {
					valStrs = append(valStrs, "'"+escapeSQLString(string(b))+"'")
				} else {
					valStrs = append(valStrs, fmt.Sprintf("'%v'", v))
				}
			}
			colNames := make([]string, len(cols))
			for i, c := range cols {
				colNames[i] = "`" + escapeBacktick(c) + "`"
			}
			sb.WriteString(fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s);\n", escapeBacktick(tbl), strings.Join(colNames, ", "), strings.Join(valStrs, ", ")))
		}
		dataRows.Close()
		sb.WriteString("\n")
	}

	content := sb.String()
	fileName := name + ".sql"
	filePath := filepath.Join(bakDir, fileName)
	if writeErr := os.WriteFile(filePath, []byte(content), 0644); writeErr != nil {
		return "", 0, fmt.Errorf("写入备份文件失败: %w", writeErr)
	}
	return fileName, int64(len(content)), nil
}

func parseSQLStatements(content string) []string {
	var statements []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(content); i++ {
		ch := content[i]

		if inBlockComment {
			if ch == '*' && i+1 < len(content) && content[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				current.WriteByte(ch)
			}
			continue
		}

		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar {
				if i+1 < len(content) && content[i+1] == quoteChar {
					current.WriteByte(content[i+1])
					i++
					continue
				}
				inQuote = false
			} else if ch == '\\' {
				if i+1 < len(content) {
					current.WriteByte(content[i+1])
					i++
				}
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
			continue
		}

		if ch == '-' && i+1 < len(content) && content[i+1] == '-' {
			inLineComment = true
			current.WriteByte(ch)
			current.WriteByte(content[i+1])
			i++
			continue
		}

		if ch == '/' && i+1 < len(content) && content[i+1] == '*' {
			inBlockComment = true
			i++
			continue
		}

		if ch == ';' {
			stmt := strings.TrimSpace(current.String())
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			isCommentOnly := true
			for _, line := range strings.Split(stmt, "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
					isCommentOnly = false
					break
				}
			}
			if !isCommentOnly && stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	remaining := strings.TrimSpace(current.String())
	remaining = strings.TrimSuffix(remaining, ";")
	remaining = strings.TrimSpace(remaining)
	if remaining != "" {
		isCommentOnly := true
		for _, line := range strings.Split(remaining, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
				isCommentOnly = false
				break
			}
		}
		if !isCommentOnly {
			statements = append(statements, remaining)
		}
	}

	return statements
}

func resolveDatabaseName(serverID uint, source string, providedDB string) (string, error) {
	if providedDB != "" && !strings.Contains(providedDB, ":") {
		return providedDB, nil
	}

	if source != "remote" {
		mutex.Lock()
		for _, db := range databases {
			if db.ID == serverID {
				if db.Database != "" && !strings.Contains(db.Database, ":") {
					mutex.Unlock()
					return db.Database, nil
				}
				break
			}
		}
		mutex.Unlock()
	}

	server := findAnyServer(serverID, source)
	if server == nil {
		if providedDB != "" {
			return providedDB, nil
		}
		return "", fmt.Errorf("无法确定数据库名称，请选择具体的数据库")
	}

	db, err := openMySQLWithTimeout(server, 3*time.Second)
	if err != nil {
		if providedDB != "" {
			return providedDB, nil
		}
		return "", fmt.Errorf("连接数据库失败，无法获取数据库列表: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		if providedDB != "" {
			return providedDB, nil
		}
		return "", fmt.Errorf("获取数据库列表失败: %w", err)
	}
	defer rows.Close()

	systemDBs := map[string]bool{
		"information_schema": true,
		"performance_schema": true,
		"mysql":              true,
		"sys":                true,
	}

	var userDBs []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if systemDBs[name] {
			continue
		}
		userDBs = append(userDBs, name)
	}

	if providedDB != "" {
		for _, name := range userDBs {
			if strings.EqualFold(name, providedDB) {
				return name, nil
			}
		}
		return providedDB, nil
	}

	if len(userDBs) > 0 {
		return userDBs[0], nil
	}

	return "", fmt.Errorf("未找到可用的数据库，请选择具体的数据库")
}

func backupRetentionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	dataDir := getDataDir()
	configPath := filepath.Join(dataDir, "backup_retention.json")

	if r.Method == "GET" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"days": 30}})
			return
		}
		var config map[string]interface{}
		json.Unmarshal(data, &config)
		writeJSON(w, map[string]interface{}{"code": 0, "data": config})
		return
	}

	if r.Method == "PUT" {
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
			return
		}
		data, _ := json.Marshal(config)
		if err := atomicWriteFile(configPath, data, 0644); err != nil {
			writeJSON(w, map[string]interface{}{"code": 500, "msg": "配置保存失败: " + err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "配置已保存"})
	}
}

func doMySQLBackupAllDatabases(server *RemoteServer, bakDir string, name string) (string, int64, error) {
	// 先获取所有数据库列表
	db, err := openMySQLWithTimeout(server, 3*time.Second)
	if err != nil {
		return "", 0, fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return "", 0, fmt.Errorf("获取数据库列表失败: %w", err)
	}
	defer rows.Close()

	systemDBs := map[string]bool{
		"information_schema": true,
		"performance_schema": true,
		"mysql":              true,
		"sys":                true,
	}

	var userDBs []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if !systemDBs[name] {
			userDBs = append(userDBs, name)
		}
	}
	if len(userDBs) == 0 {
		return "", 0, fmt.Errorf("没有可用的用户数据库")
	}

	// 尝试使用 mysqldump 一次性备份所有库
	if mysqldumpPath, err := exec.LookPath("mysqldump"); err == nil {
		tmpFile, err := os.CreateTemp("", "mysqldump-*.cnf")
		if err != nil {
			return "", 0, fmt.Errorf("创建临时配置文件失败: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		cnfContent := fmt.Sprintf("[client]\npassword=%s\n", server.Password)
		if _, err := tmpFile.WriteString(cnfContent); err != nil {
			tmpFile.Close()
			return "", 0, fmt.Errorf("写入临时配置文件失败: %w", err)
		}
		tmpFile.Close()

		args := []string{
			"--defaults-extra-file=" + tmpPath,
			"-h", server.Host,
			"-P", fmt.Sprintf("%d", server.Port),
			"-u", server.Username,
			"--single-transaction",
			"--routines",
			"--triggers",
			"--set-charset",
			"--databases",
		}
		args = append(args, userDBs...)

		cmd := exec.Command(mysqldumpPath, args...)
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			fileName := name + ".sql"
			filePath := filepath.Join(bakDir, fileName)
			if writeErr := os.WriteFile(filePath, output, 0644); writeErr != nil {
				return "", 0, fmt.Errorf("写入备份文件失败: %w", writeErr)
			}
			return fileName, int64(len(output)), nil
		}
		if err != nil {
			fmt.Printf("mysqldump (all databases) failed: %s, output: %s\n", err.Error(), string(output))
			sysLogError("BACKUP", fmt.Sprintf("mysqldump全库备份失败 (连接: %s:%d)", server.Host, server.Port))
		}
	}

	// 回退方案：逐个库备份并合并
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-- MySQL Backup (All Databases)\n-- Created: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	for _, dbName := range userDBs {
		sb.WriteString(fmt.Sprintf("-- ===== Database: %s =====\n\n", dbName))

		// 切换到该库
		_, _ = db.Exec(fmt.Sprintf("USE `%s`", escapeBacktick(dbName)))

		// 获取表列表
		tblRows, err := db.Query("SHOW TABLES")
		if err != nil {
			continue
		}

		var tables []string
		for tblRows.Next() {
			var tbl string
			tblRows.Scan(&tbl)
			tables = append(tables, tbl)
		}
		tblRows.Close()

		for _, tbl := range tables {
			var tableName, createStmt string
			if err := db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", escapeBacktick(tbl))).Scan(&tableName, &createStmt); err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`;\n%s;\n\n", escapeBacktick(dbName), escapeBacktick(tbl), createStmt))

			dataRows, err := db.Query(fmt.Sprintf("SELECT * FROM `%s`.`%s`", escapeBacktick(dbName), escapeBacktick(tbl)))
			if err != nil {
				continue
			}

			cols, _ := dataRows.Columns()
			vals := make([]interface{}, len(cols))
			valPointers := make([]interface{}, len(cols))
			for i := range vals {
				valPointers[i] = &vals[i]
			}

			for dataRows.Next() {
				if err := dataRows.Scan(valPointers...); err != nil {
					continue
				}

				var valueStrings []string
				for _, v := range vals {
					if v == nil {
						valueStrings = append(valueStrings, "NULL")
					} else {
						switch val := v.(type) {
						case []byte:
							valueStrings = append(valueStrings, fmt.Sprintf("'%s'", escapeQuote(string(val))))
						case string:
							valueStrings = append(valueStrings, fmt.Sprintf("'%s'", escapeQuote(val)))
						default:
							valueStrings = append(valueStrings, fmt.Sprintf("%v", v))
						}
					}
				}

				sb.WriteString(fmt.Sprintf("INSERT INTO `%s`.`%s` VALUES (%s);\n", escapeBacktick(dbName), escapeBacktick(tbl), strings.Join(valueStrings, ", ")))
			}
			sb.WriteString("\n")
			dataRows.Close()
		}
	}

	content := sb.String()
	fileName := name + ".sql"
	filePath := filepath.Join(bakDir, fileName)
	if writeErr := os.WriteFile(filePath, []byte(content), 0644); writeErr != nil {
		return "", 0, fmt.Errorf("写入备份文件失败: %w", writeErr)
	}
	return fileName, int64(len(content)), nil
}
