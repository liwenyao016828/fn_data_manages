package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func formatSQLiteError(prefix string, err error) string {
	errStr := err.Error()
	lower := strings.ToLower(errStr)

	if strings.Contains(lower, "unable to open") || strings.Contains(lower, "no such file") {
		return prefix + "：无法打开数据库文件，请检查文件路径是否正确"
	}
	if strings.Contains(lower, "disk i/o error") {
		return prefix + "：磁盘I/O错误，请检查文件权限或磁盘空间"
	}
	if strings.Contains(lower, "database is locked") {
		return prefix + "：数据库被锁定，请稍后重试"
	}
	if strings.Contains(lower, "readonly database") {
		return prefix + "：数据库为只读，请检查文件权限"
	}
	if strings.Contains(lower, "syntax") || strings.Contains(lower, "near ") {
		return prefix + "：SQL 语法错误"
	}
	if strings.Contains(lower, "no such table") {
		return prefix + "：表不存在"
	}
	if strings.Contains(lower, "no such column") {
		return prefix + "：列不存在"
	}
	if strings.Contains(lower, "unique constraint") || strings.Contains(lower, "constraint failed") {
		return prefix + "：约束冲突，数据可能已存在"
	}
	if strings.Contains(lower, "not null constraint") {
		return prefix + "：非空约束冲突，必填字段不能为空"
	}
	if strings.Contains(lower, "foreign key") {
		return prefix + "：外键约束冲突"
	}

	return prefix + "：" + errStr
}

func validateSQLiteTableName(name string) error {
	if name == "" {
		return fmt.Errorf("表名不能为空")
	}
	if len(name) > 64 {
		return fmt.Errorf("表名长度不能超过64个字符")
	}
	if !identifierRegex.MatchString(name) {
		return fmt.Errorf("表名包含非法字符，只允许字母、数字和下划线")
	}
	return nil
}

func validateSQLiteFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("文件路径不允许包含父目录引用")
	}
	return nil
}

func sqliteTablesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	source := r.URL.Query().Get("source")
	if serverID == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	server := findAnyServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openSQLite(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("连接SQLite失败", err)})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("查询表列表失败", err)})
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": tables})
}

func sqliteColumnsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	tableName := r.URL.Query().Get("table")
	source := r.URL.Query().Get("source")
	if serverID == "" || tableName == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and table required"})
		return
	}

	if err := validateSQLiteTableName(tableName); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "表名验证失败: " + err.Error()})
		return
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	server := findAnyServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openSQLite(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("连接SQLite失败", err)})
		return
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", quoteIdentifier(tableName)))
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("查询列信息失败", err)})
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var columns []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		rows.Scan(valPtrs...)
		col := make(map[string]interface{})
		for i, name := range cols {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				col[name] = string(b)
			} else {
				col[name] = v
			}
		}
		columns = append(columns, col)
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": columns})
}

func sqliteDataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	tableName := r.URL.Query().Get("table")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")
	source := r.URL.Query().Get("source")

	if serverID == "" || tableName == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and table required"})
		return
	}

	if err := validateSQLiteTableName(tableName); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "表名验证失败: " + err.Error()})
		return
	}

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize < 1 || pageSize > 500 {
		pageSize = 50
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	server := findAnyServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openSQLite(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("连接SQLite失败", err)})
		return
	}
	defer db.Close()

	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdentifier(tableName))
	err = db.QueryRow(countSQL).Scan(&total)
	if err != nil {
		total = 0
	}

	offset := (page - 1) * pageSize
	selectSQL := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", quoteIdentifier(tableName))
	rows, err := db.Query(selectSQL, pageSize, offset)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("查询数据失败", err)})
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var data []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		rows.Scan(valPtrs...)
		row := make(map[string]interface{})
		for i, name := range cols {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				row[name] = string(b)
			} else {
				row[name] = v
			}
		}
		data = append(data, row)
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{
		"columns":  cols,
		"rows":     data,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	}})
}

func sqliteExecuteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
		return
	}

	var req struct {
		ServerID int    `json:"server_id"`
		SQL      string `json:"sql"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")

	if req.ServerID == 0 || req.SQL == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and sql required"})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openSQLite(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("连接SQLite失败", err)})
		return
	}
	defer db.Close()

	sqlTrimmed := strings.TrimSpace(req.SQL)
	sqlUpper := strings.ToUpper(sqlTrimmed)

	// 安全检查：阻止危险操作
	dangerousKeywords := []string{
		"ATTACH DATABASE",
		"DETACH DATABASE",
	}
	for _, kw := range dangerousKeywords {
		if strings.Contains(sqlUpper, kw) {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": fmt.Sprintf("禁止执行危险操作: %s", kw)})
			return
		}
	}

	sqlPreview := req.SQL
	if len(sqlPreview) > 50 {
		sqlPreview = sqlPreview[:50] + "..."
	}
	sysLogInfo("SQLITE", fmt.Sprintf("执行SQL: %s (文件: %s)", sqlPreview, server.Host))

	if strings.HasPrefix(sqlUpper, "SELECT") || strings.HasPrefix(sqlUpper, "PRAGMA") {
		querySQL := req.SQL
		// 对 SELECT 添加 LIMIT（PRAGMA 不需要）
		if strings.HasPrefix(sqlUpper, "SELECT") && !strings.Contains(sqlUpper, "LIMIT") {
			querySQL = req.SQL + " LIMIT 1000"
		}
		rows, err := db.Query(querySQL)
		if err != nil {
			sysLogWarn("SQLITE", fmt.Sprintf("SQL查询失败: %s (文件: %s)", truncateSQL(req.SQL), server.Host))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("查询失败", err)})
			return
		}
		defer rows.Close()

		cols, _ := rows.Columns()
		var data []map[string]interface{}
		maxRows := 1000
		rowCount := 0
		for rows.Next() {
			rowCount++
			if rowCount > maxRows {
				break
			}
			vals := make([]interface{}, len(cols))
			valPtrs := make([]interface{}, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}
			rows.Scan(valPtrs...)
			row := make(map[string]interface{})
			for i, name := range cols {
				v := vals[i]
				if b, ok := v.([]byte); ok {
					row[name] = string(b)
				} else {
					row[name] = v
				}
			}
			data = append(data, row)
		}

		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"columns": cols, "rows": data}})
	} else {
		result, err := db.Exec(req.SQL)
		if err != nil {
			sysLogError("SQLITE", fmt.Sprintf("SQL执行失败: %s (文件: %s)", truncateSQL(req.SQL), server.Host))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("执行失败", err)})
			return
		}
		affected, _ := result.RowsAffected()
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"affected": affected, "msg": "执行成功"}})
	}
}

func sqliteCreateTableHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
		return
	}

	var req struct {
		ServerID int    `json:"server_id"`
		Table    string `json:"table"`
		Columns  string `json:"columns"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")

	if req.ServerID == 0 || req.Table == "" || req.Columns == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id, table and columns required"})
		return
	}

	if err := validateSQLiteTableName(req.Table); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "表名验证失败: " + err.Error()})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openSQLite(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("连接SQLite失败", err)})
		return
	}
	defer db.Close()

	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", quoteIdentifier(req.Table), req.Columns)
	_, err = db.Exec(createSQL)
	if err != nil {
		sysLogError("SQLITE", fmt.Sprintf("创建表失败: %s (文件: %s)", req.Table, server.Host))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("创建表失败", err)})
		return
	}

	sysLogInfo("SQLITE", fmt.Sprintf("创建表: %s (文件: %s)", req.Table, server.Host))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "创建成功"})
}

func sqliteDropTableHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
		return
	}

	var req struct {
		ServerID int    `json:"server_id"`
		Table    string `json:"table"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")

	if req.ServerID == 0 || req.Table == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and table required"})
		return
	}

	if err := validateSQLiteTableName(req.Table); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "表名验证失败: " + err.Error()})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openSQLite(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("连接SQLite失败", err)})
		return
	}
	defer db.Close()

	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteIdentifier(req.Table))
	_, err = db.Exec(dropSQL)
	if err != nil {
		sysLogError("SQLITE", fmt.Sprintf("删除表失败: %s (文件: %s)", req.Table, server.Host))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatSQLiteError("删除表失败", err)})
		return
	}

	sysLogInfo("SQLITE", fmt.Sprintf("删除表: %s (文件: %s)", req.Table, server.Host))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "删除成功"})
}

func sqlitePingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	source := r.URL.Query().Get("source")
	if serverID == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	server := findAnyServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(server.Host); os.IsNotExist(err) {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false, "msg": "文件不存在"}})
		return
	}

	// 尝试打开并验证是否为有效的 SQLite 文件
	db, err := openSQLite(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false, "msg": formatSQLiteError("无法打开", err)}})
		return
	}
	defer db.Close()

	// 执行简单查询验证连接
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false, "msg": "查询验证失败"}})
		return
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": true}})
}

func sqliteBackupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
		return
	}

	var req struct {
		ServerID int    `json:"server_id"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")

	if req.ServerID == 0 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	// 检查源文件是否存在
	if _, err := os.Stat(server.Host); os.IsNotExist(err) {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "SQLite数据库文件不存在"})
		return
	}

	name := req.Name
	if name == "" {
		name = fmt.Sprintf("sqlite_backup_%s", time.Now().Format("20060102_150405"))
	}

	bakDir := filepath.Join(getDataDir(), "backups")
	os.MkdirAll(bakDir, 0755)

	backupFileName := name + filepath.Ext(server.Host)
	if backupFileName == name {
		backupFileName = name + ".db"
	}
	backupPath := filepath.Join(bakDir, backupFileName)

	// 复制文件
	srcFile, err := os.Open(server.Host)
	if err != nil {
		sysLogError("SQLITE", fmt.Sprintf("备份-打开源文件失败 (文件: %s)", server.Host))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "打开数据库文件失败: " + err.Error()})
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(backupPath)
	if err != nil {
		sysLogError("SQLITE", fmt.Sprintf("备份-创建备份文件失败 (文件: %s)", backupPath))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "创建备份文件失败: " + err.Error()})
		return
	}
	defer dstFile.Close()

	// 先锁定数据库确保一致性
	db, err := openSQLite(server)
	if err == nil {
		_, _ = db.Exec("BEGIN IMMEDIATE")
	}

	written, err := copyFileData(dstFile, srcFile)
	if db != nil {
		_, _ = db.Exec("COMMIT")
		db.Close()
	}

	if err != nil {
		sysLogError("SQLITE", fmt.Sprintf("备份-复制文件失败 (文件: %s)", server.Host))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "复制文件失败: " + err.Error()})
		return
	}

	// 记录备份信息
	newBackup := Backup{
		ID:          nextBackupID,
		Name:        name,
		Type:        "sqlite",
		ServerID:    uint(req.ServerID),
		Host:        server.Host,
		Database:    filepath.Base(server.Host),
		FileName:    backupFileName,
		FileSize:    written,
		Status:      "success",
		Description: "SQLite文件备份",
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		BackupType:  "backup",
		BackupLevel: "sqlite",
		Source:      source,
	}

	mutex.Lock()
	nextBackupID++
	backups = append(backups, newBackup)
	mutex.Unlock()
	saveData()

	sysLogInfo("SQLITE", fmt.Sprintf("备份SQLite: %s -> %s", server.Host, backupPath))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "备份成功", "data": newBackup})
}

func sqliteRestoreHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
		return
	}

	var req struct {
		ServerID  int    `json:"server_id"`
		BackupID  uint   `json:"backup_id"`
		FilePath  string `json:"file_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")

	if req.ServerID == 0 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	var backupFilePath string

	if req.BackupID > 0 {
		// 从备份记录中查找
		mutex.Lock()
		var backup *Backup
		for i := range backups {
			if backups[i].ID == req.BackupID {
				backup = &backups[i]
				break
			}
		}
		mutex.Unlock()

		if backup == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "备份记录不存在"})
			return
		}

		bakDir := filepath.Join(getDataDir(), "backups")
		backupFilePath = filepath.Join(bakDir, backup.FileName)
	} else if req.FilePath != "" {
		backupFilePath = req.FilePath
	} else {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "backup_id or file_path required"})
		return
	}

	// 检查备份文件是否存在
	if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "备份文件不存在"})
		return
	}

	// 打开备份文件
	srcFile, err := os.Open(backupFilePath)
	if err != nil {
		sysLogError("SQLITE", fmt.Sprintf("恢复-打开备份文件失败 (文件: %s)", backupFilePath))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "打开备份文件失败: " + err.Error()})
		return
	}
	defer srcFile.Close()

	// 先备份当前数据库文件
	currentBackup := server.Host + ".restore_backup"
	if _, err := os.Stat(server.Host); err == nil {
		copyFile(server.Host, currentBackup)
	}

	// 写入恢复文件
	dstFile, err := os.Create(server.Host)
	if err != nil {
		sysLogError("SQLITE", fmt.Sprintf("恢复-写入数据库文件失败 (文件: %s)", server.Host))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "写入数据库文件失败: " + err.Error()})
		return
	}

	_, err = copyFileData(dstFile, srcFile)
	dstFile.Close()
	if err != nil {
		// 恢复失败，尝试还原之前的备份
		if _, restoreErr := os.Stat(currentBackup); restoreErr == nil {
			copyFile(currentBackup, server.Host)
		}
		sysLogError("SQLITE", fmt.Sprintf("恢复-复制文件失败 (文件: %s)", server.Host))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "恢复失败: " + err.Error()})
		return
	}

	// 清理临时备份
	os.Remove(currentBackup)

	sysLogInfo("SQLITE", fmt.Sprintf("恢复SQLite: %s <- %s", server.Host, backupFilePath))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "恢复成功"})
}

func copyFileData(dst *os.File, src *os.File) (int64, error) {
	return io.Copy(dst, src)
}

// copyFile 简单复制文件
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
