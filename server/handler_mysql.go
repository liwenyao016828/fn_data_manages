package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func mysqlDatabasesHandler(w http.ResponseWriter, r *http.Request) {
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

	containerName := findContainerName(uint(id), source)

	if containerName != "" {
		if dockerPath, err := exec.LookPath("docker"); err == nil {
			out, err := exec.Command(dockerPath, "stop", containerName).CombinedOutput()
			if err != nil {
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "Docker停止失败: " + strings.TrimSpace(string(out))})
				return
			}
			writeJSON(w, map[string]interface{}{"code": 0, "msg": "MySQL已停止(Docker容器 " + containerName + ")"})
			return
		}
	}

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		databases = append(databases, name)
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": databases})
}

func mysqlCreateDatabaseHandler(w http.ResponseWriter, r *http.Request) {
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
		ServerID   int    `json:"server_id"`
		Name       string `json:"name"`
		Password   string `json:"password"`
		Charset    string `json:"charset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")

	if req.ServerID == 0 || req.Name == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and name required"})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	charset := req.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s_general_ci", escapeBacktick(req.Name), charset, charset))
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "创建数据库失败: " + err.Error()})
		return
	}

	if req.Password != "" {
		_, err = db.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", escapeQuote(req.Name), escapeQuote(req.Password)))
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "创建用户失败: " + err.Error()})
			return
		}
		_, err = db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", escapeBacktick(req.Name), escapeQuote(req.Name)))
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "授权失败: " + err.Error()})
			return
		}
		_, _ = db.Exec("FLUSH PRIVILEGES")
	}

	writeJSON(w, map[string]interface{}{"code": 0, "msg": "创建成功"})
}

func escapeBacktick(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

func escapeQuote(s string) string {
	return strings.ReplaceAll(s, "'", "\\'")
}

func mysqlTablesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	dbName := r.URL.Query().Get("database")
	source := r.URL.Query().Get("source")
	if serverID == "" || dbName == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and database required"})
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf("SHOW TABLES FROM `%s`", escapeBacktick(dbName)))
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
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

func mysqlColumnsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	dbName := r.URL.Query().Get("database")
	tableName := r.URL.Query().Get("table")
	source := r.URL.Query().Get("source")
	if serverID == "" || dbName == "" || tableName == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id, database and table required"})
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`.`%s`", escapeBacktick(dbName), escapeBacktick(tableName)))
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
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

func mysqlDataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	dbName := r.URL.Query().Get("database")
	tableName := r.URL.Query().Get("table")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("pageSize")
	source := r.URL.Query().Get("source")

	if serverID == "" || dbName == "" || tableName == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id, database and table required"})
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	var total int64
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", escapeBacktick(dbName), escapeBacktick(tableName))).Scan(&total)
	if err != nil {
		total = 0
	}

	offset := (page - 1) * pageSize
	sqlQuery := fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT %d OFFSET %d", escapeBacktick(dbName), escapeBacktick(tableName), pageSize, offset)
	rows, err := db.Query(sqlQuery)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
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

func mysqlExecuteHandler(w http.ResponseWriter, r *http.Request) {
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
		Database string `json:"database"`
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	if req.Database != "" {
		_, err := db.Exec(fmt.Sprintf("USE `%s`", escapeBacktick(req.Database)))
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "选择数据库失败: " + err.Error()})
			return
		}
	}

	sqlTrimmed := strings.TrimSpace(req.SQL)
	sqlUpper := strings.ToUpper(sqlTrimmed)

	dangerousKeywords := []string{
		"DROP DATABASE", "DROP VIEW", "DROP PROCEDURE", "DROP FUNCTION", "DROP TRIGGER", "DROP EVENT", "DROP USER",
		"ALTER DATABASE", "ALTER USER",
		"TRUNCATE ", "RENAME ",
		"CREATE DATABASE", "CREATE USER", "CREATE TABLESPACE",
		"GRANT ", "REVOKE ", "KILL ", "SHUTDOWN",
		"SET PASSWORD", "FLUSH PRIVILEGES", "FLUSH TABLES",
	}
	for _, kw := range dangerousKeywords {
		if strings.Contains(sqlUpper, kw) {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": fmt.Sprintf("禁止执行危险操作: %s", strings.TrimSpace(kw))})
			return
		}
	}

	if strings.HasPrefix(sqlUpper, "SELECT") || strings.HasPrefix(sqlUpper, "SHOW") || strings.HasPrefix(sqlUpper, "DESCRIBE") || strings.HasPrefix(sqlUpper, "EXPLAIN") {
		rows, err := db.Query(req.SQL)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
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

		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"columns": cols, "rows": data}})
	} else {
		result, err := db.Exec(req.SQL)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
			return
		}
		affected, _ := result.RowsAffected()
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"affected": affected, "msg": "执行成功"}})
	}
}

var mysqlConfigPaths = []string{
	"/etc/my.cnf",
	"/etc/mysql/my.cnf",
	"/etc/mysql/mysql.conf.d/mysqld.cnf",
	"/usr/local/etc/my.cnf",
	"/opt/homebrew/etc/my.cnf",
}

func findMySQLConfigFile(server *RemoteServer) (string, error) {
	db, err := openMySQL(server)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var basedir, version string
	db.QueryRow("SHOW VARIABLES LIKE 'basedir'").Scan(new(string), &basedir)
	db.QueryRow("SHOW VARIABLES LIKE 'version'").Scan(new(string), &version)

	if basedir != "" {
		baseLower := strings.ToLower(basedir)
		if strings.Contains(baseLower, "mysql server") {
			verPart := ""
			parts := strings.Split(strings.ReplaceAll(basedir, "\\", "/"), "/")
			for _, p := range parts {
				if strings.Contains(p, "MySQL Server") {
					verPart = strings.TrimPrefix(p, "MySQL Server ")
					verPart = strings.TrimSpace(verPart)
					break
				}
			}
			if verPart == "" && version != "" {
				dotIdx := strings.Index(version, ".")
				if dotIdx > 0 {
					secondDot := strings.Index(version[dotIdx+1:], ".")
					if secondDot > 0 {
						verPart = version[:dotIdx+1+secondDot]
					} else {
						verPart = version[:dotIdx]
					}
				}
			}
			myIni := filepath.Join("C:\\ProgramData\\MySQL", "MySQL Server "+verPart, "my.ini")
			if _, err := os.Stat(myIni); err == nil {
				return myIni, nil
			}
		}
		myIni := filepath.Join(basedir, "my.ini")
		if _, err := os.Stat(myIni); err == nil {
			return myIni, nil
		}
		myCnf := filepath.Join(basedir, "my.cnf")
		if _, err := os.Stat(myCnf); err == nil {
			return myCnf, nil
		}
	}

	for _, p := range mysqlConfigPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("找不到MySQL配置文件，请检查 %s", strings.Join(mysqlConfigPaths, ", "))
}

func mysqlConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
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

	if r.Method == "GET" {
		cfgPath, cfgErr := findMySQLConfigFile(server)
		if cfgErr == nil {
			data, err := os.ReadFile(cfgPath)
			if err == nil {
				writeJSON(w, map[string]interface{}{
					"code": 0,
					"data": map[string]interface{}{
						"content":  string(data),
						"filePath": cfgPath,
						"source":   "file",
					},
				})
				return
			}
		}

		content, err := getMySQLConfigFromSQL(server)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{
				"content":  "# 无法获取配置: " + err.Error() + "\n",
				"filePath": "",
				"source":   "none",
			}})
			return
		}
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"content":  content,
				"filePath": "",
				"source":   "sql",
			},
		})

	} else if r.Method == "PUT" {
		var req struct {
			Content  string `json:"content"`
			FilePath string `json:"filePath"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
			return
		}

		if req.FilePath != "" {
			cfgPath := req.FilePath
			backupPath := cfgPath + ".backup"
			origData, _ := os.ReadFile(cfgPath)
			os.WriteFile(backupPath, origData, 0644)
			if err := os.WriteFile(cfgPath, []byte(req.Content), 0644); err != nil {
				writeJSON(w, map[string]interface{}{"code": 500, "msg": "写入配置失败(权限不足): " + err.Error()})
				return
			}
			writeJSON(w, map[string]interface{}{"code": 0, "msg": "配置已保存至 " + cfgPath + "，旧配置备份至 " + backupPath + "，请重启MySQL使配置生效"})
			return
		}

		var reqVars struct {
			Variables map[string]string `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqVars); err != nil || len(reqVars.Variables) == 0 {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "无法定位配置文件，且未提供变量修改。可通过'常用配置项'修改在线变量"})
			return
		}
		db, err := openMySQL(server)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
			return
		}
		defer db.Close()
		var updated []string
		var failed []string
		for k, v := range reqVars.Variables {
			var valStr string
			if _, err := strconv.ParseFloat(v, 64); err == nil {
				valStr = v
			} else {
				valStr = "'" + escapeSQLString(v) + "'"
			}
			_, err := db.Exec(fmt.Sprintf("SET GLOBAL %s = %s", escapeBacktick(k), valStr))
			if err != nil {
				failed = append(failed, k)
			} else {
				updated = append(updated, k)
			}
		}
		msg := ""
		if len(updated) > 0 {
			msg += fmt.Sprintf("已在线修改: %s", strings.Join(updated, ", "))
		}
		if len(failed) > 0 {
			msg += fmt.Sprintf("；修改失败(可能为只读变量): %s", strings.Join(failed, ", "))
		}
		writeJSON(w, map[string]interface{}{"code": 0, "msg": msg})
	}
}

func getMySQLConfigFromSQL(server *RemoteServer) (string, error) {
	db, err := openMySQL(server)
	if err != nil {
		return "", err
	}
	defer db.Close()

	variables := []string{
		"max_connections", "innodb_buffer_pool_size", "port",
		"datadir", "long_query_time", "slow_query_log",
		"bind_address", "character_set_server", "collation_server",
		"sql_mode", "default_storage_engine", "wait_timeout",
		"interactive_timeout", "max_allowed_packet", "sort_buffer_size",
		"join_buffer_size", "read_buffer_size", "read_rnd_buffer_size",
		"thread_cache_size", "table_open_cache", "log_error",
		"general_log", "slow_query_log_file", "sync_binlog",
		"innodb_flush_log_at_trx_commit", "innodb_log_file_size",
		"key_buffer_size", "query_cache_size", "query_cache_type",
		"lower_case_table_names", "explicit_defaults_for_timestamp",
		"autocommit", "transaction_isolation",
	}

	var sb strings.Builder
	sb.WriteString("# MySQL 配置 (通过 SHOW VARIABLES 获取)\n")
	var version, basedir string
	db.QueryRow("SELECT VERSION()").Scan(&version)
	db.QueryRow("SHOW VARIABLES LIKE 'basedir'").Scan(new(string), &basedir)
	if version != "" {
		sb.WriteString(fmt.Sprintf("# 版本: %s\n", version))
	}
	if basedir != "" {
		sb.WriteString(fmt.Sprintf("# basedir: %s\n", basedir))
	}
	sb.WriteString("# 注意: 此为运行时变量，非配置文件。修改需通过'常用配置项'在线修改\n")
	sb.WriteString("[mysqld]\n\n")

	for _, v := range variables {
		var name, val string
		if err := db.QueryRow(fmt.Sprintf("SHOW VARIABLES LIKE '%s'", escapeSQLString(v))).Scan(&name, &val); err == nil {
			if isNumericValue(val) {
				sb.WriteString(fmt.Sprintf("%s = %s\n", name, val))
			} else {
				sb.WriteString(fmt.Sprintf("%s = %s\n", name, val))
			}
		}
	}

	return sb.String(), nil
}

func isNumericValue(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func mysqlLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	logType := r.URL.Query().Get("type")
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()

	var logs []map[string]string

	var mysqlDataDir string
	dirRows, dirErr := db.Query("SHOW VARIABLES LIKE 'datadir'")
	if dirErr == nil {
		var dn, dv string
		for dirRows.Next() {
			dirRows.Scan(&dn, &dv)
		}
		dirRows.Close()
		mysqlDataDir = dv
	}

	resolveLogPath := func(logFile string) string {
		if logFile == "" {
			return ""
		}
		if filepath.IsAbs(logFile) {
			return logFile
		}
		if mysqlDataDir != "" {
			return filepath.Join(mysqlDataDir, logFile)
		}
		return logFile
	}

	readLogFile := func(path string) ([]map[string]string, string) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err.Error()
		}
		var result []map[string]string
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				result = append(result, map[string]string{
					"time":    "",
					"level":   "Note",
					"message": line,
				})
			}
		}
		return result, ""
	}

	if logType == "slow" {
		rows, err := db.Query("SHOW VARIABLES LIKE 'slow_query_log_file'")
		if err == nil {
			defer rows.Close()
			var name, logFile string
			for rows.Next() {
				rows.Scan(&name, &logFile)
			}
			if logFile != "" {
				resolvedPath := resolveLogPath(logFile)
				fileLogs, readErr := readLogFile(resolvedPath)
				if readErr == "" {
					for _, l := range fileLogs {
						l["time"] = extractLogTime(l["message"], "slow")
						l["level"] = "slow"
						logs = append(logs, l)
					}
				} else {
					slowRows, slowErr := db.Query("SELECT start_time, sql_text, rows_examined, query_time FROM mysql.slow_log ORDER BY start_time DESC LIMIT 200")
					if slowErr == nil {
						defer slowRows.Close()
						for slowRows.Next() {
							var startTime, sqlText, rowsExamined, queryTime string
							slowRows.Scan(&startTime, &sqlText, &rowsExamined, &queryTime)
							logs = append(logs, map[string]string{
								"time":    startTime,
								"level":   "slow",
								"message": sqlText + " (扫描行:" + rowsExamined + " 耗时:" + queryTime + ")",
							})
						}
					}
					if len(logs) == 0 {
						logs = append(logs, map[string]string{
							"time":    "",
							"level":   "Warning",
							"message": "无法读取慢查询日志 (文件权限不足且日志表未启用): " + resolvedPath,
						})
					}
				}
			} else {
				logs = append(logs, map[string]string{
					"time":    "",
					"level":   "Note",
					"message": "慢查询日志未启用，请先开启 slow_query_log",
				})
			}
		}
	} else if logType == "general" {
		rows, err := db.Query("SHOW VARIABLES LIKE 'general_log_file'")
		if err == nil {
			defer rows.Close()
			var name, logFile string
			for rows.Next() {
				rows.Scan(&name, &logFile)
			}
			if logFile != "" {
				resolvedPath := resolveLogPath(logFile)
				fileLogs, readErr := readLogFile(resolvedPath)
				if readErr == "" {
					for _, l := range fileLogs {
						l["time"] = extractLogTime(l["message"], "general")
						l["level"] = "query"
						logs = append(logs, l)
					}
				} else {
					genRows, genErr := db.Query("SELECT event_time, user_host, argument FROM mysql.general_log ORDER BY event_time DESC LIMIT 200")
					if genErr == nil {
						defer genRows.Close()
						for genRows.Next() {
							var eventTime, userHost, argument string
							genRows.Scan(&eventTime, &userHost, &argument)
							logs = append(logs, map[string]string{
								"time":    eventTime,
								"level":   "query",
								"message": "[" + userHost + "] " + argument,
							})
						}
					}
					if len(logs) == 0 {
						logs = append(logs, map[string]string{
							"time":    "",
							"level":   "Warning",
							"message": "无法读取通用日志 (文件权限不足且日志表未启用): " + resolvedPath,
						})
					}
				}
			} else {
				logs = append(logs, map[string]string{
					"time":    "",
					"level":   "Note",
					"message": "通用日志未启用，请先开启 general_log",
				})
			}
		}
	} else {
		rows, err := db.Query("SHOW VARIABLES LIKE 'log_error'")
		if err == nil {
			defer rows.Close()
			var name, logFile string
			for rows.Next() {
				rows.Scan(&name, &logFile)
			}
			if logFile != "" {
				resolvedPath := resolveLogPath(logFile)
				fileLogs, readErr := readLogFile(resolvedPath)
				if readErr == "" {
					for _, l := range fileLogs {
						l["time"] = extractLogTime(l["message"], "error")
						line := l["message"]
						level := "Note"
						if strings.Contains(strings.ToUpper(line), "ERROR") {
							level = "Error"
						} else if strings.Contains(strings.ToUpper(line), "WARNING") || strings.Contains(strings.ToUpper(line), "WARN") {
							level = "Warning"
						}
						l["level"] = level
						logs = append(logs, l)
					}
				} else {
					statusRows, statusErr := db.Query("SHOW ENGINE INNODB STATUS")
					if statusErr == nil {
						defer statusRows.Close()
						for statusRows.Next() {
							var typ, nameVal, statusText string
							statusRows.Scan(&typ, &nameVal, &statusText)
							lines := strings.Split(statusText, "\n")
							for _, line := range lines {
								line = strings.TrimSpace(line)
								if line == "" || strings.HasPrefix(line, "=") {
									continue
								}
								level := "Note"
								upperLine := strings.ToUpper(line)
								if strings.Contains(upperLine, "ERROR") {
									level = "Error"
								} else if strings.Contains(upperLine, "WARNING") || strings.Contains(upperLine, "WARN") {
									level = "Warning"
								}
								logs = append(logs, map[string]string{
									"time":    "",
									"level":   level,
									"message": line,
								})
							}
						}
					}
					if len(logs) == 0 {
						logs = append(logs, map[string]string{
							"time":    "",
							"level":   "Warning",
							"message": "无法读取错误日志 (文件权限不足): " + resolvedPath,
						})
					}
				}
			} else {
				logs = append(logs, map[string]string{
					"time":    "",
					"level":   "Note",
					"message": "MySQL 错误日志路径为空，请检查 log_error 变量",
				})
			}
		}
	}

	if len(logs) == 0 {
		logs = append(logs, map[string]string{
			"time":    "",
			"level":   "Note",
			"message": "暂无日志数据",
		})
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": logs})
}

func mysqlLogsClearHandler(w http.ResponseWriter, r *http.Request) {
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()

	db.Exec("FLUSH ERROR LOGS")
	db.Exec("FLUSH SLOW LOGS")
	db.Exec("FLUSH GENERAL LOGS")

	writeJSON(w, map[string]interface{}{"code": 0, "msg": "日志已清空"})
}

func mysqlPortHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
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

	if r.Method == "GET" {
		cfgPath, _ := findMySQLConfigFile(server)
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{
			"port":    server.Port,
			"cfgPath": cfgPath,
		}})
		return
	}

	var req struct {
		Port      int `json:"port"`
		DockerPort int `json:"dockerPort"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	if req.Port < 1024 || req.Port > 65535 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "端口必须在1024-65535之间"})
		return
	}

	containerName := findContainerName(uint(id), source)

	if containerName != "" {
		if req.DockerPort > 0 {
			mutex.Lock()
			for i, d := range databases {
				if d.ID == uint(id) {
					databases[i].Port = req.DockerPort
					break
				}
			}
			mutex.Unlock()
			saveData()
			writeJSON(w, map[string]interface{}{"code": 0, "msg": fmt.Sprintf("Docker映射端口已更新为 %d，请确认容器端口映射已修改", req.DockerPort)})
			return
		}
		mutex.Lock()
		for i, d := range databases {
			if d.ID == uint(id) {
				databases[i].Port = req.Port
				break
			}
		}
		mutex.Unlock()
		saveData()
		writeJSON(w, map[string]interface{}{"code": 0, "msg": fmt.Sprintf("端口已更新为 %d，Docker容器需手动修改端口映射后重启容器", req.Port)})
		return
	}

	cfgPath, cfgErr := findMySQLConfigFile(server)
	if cfgErr != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "无法定位配置文件: " + cfgErr.Error()})
		return
	}

	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "无法读取配置文件: " + err.Error()})
		return
	}

	backupPath := cfgPath + ".port_backup"
	os.WriteFile(backupPath, cfgData, 0644)

	newContent := string(cfgData)
	replaced := false
	lines := strings.Split(newContent, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "port") && !strings.Contains(trimmed, "report") {
			hasComment := strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";")
			eqIdx := strings.Index(trimmed, "=")
			var prefix string
			if eqIdx > 0 {
				prefix = line[:strings.Index(line, "=")+1]
			} else {
				wsIdx := strings.Index(trimmed, " ")
				if wsIdx > 0 && strings.Index(line, " ") > 0 {
					prefix = line[:strings.Index(line, " ")+1]
				} else {
					prefix = "port = "
				}
			}
			lines[i] = prefix + fmt.Sprintf("%d", req.Port)
			if hasComment {
				lines[i] = "# " + lines[i]
			}
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, fmt.Sprintf("\nport = %d", req.Port))
	}
	newContent = strings.Join(lines, "\n")

	if err := os.WriteFile(cfgPath, []byte(newContent), 0644); err != nil {
		writeJSON(w, map[string]interface{}{"code": 500, "msg": "写入配置失败(权限不足): " + err.Error()})
		return
	}

	mutex.Lock()
	found := false
	for i, d := range databases {
		if d.ID == uint(id) {
			databases[i].Port = req.Port
			found = true
			break
		}
	}
	if !found {
		for i, s := range remoteServers {
			if s.ID == uint(id) {
				remoteServers[i].Port = req.Port
				found = true
				break
			}
		}
	}
	mutex.Unlock()
	saveData()

	writeJSON(w, map[string]interface{}{"code": 0, "msg": fmt.Sprintf("配置文件中端口已修改为 %d，存储端口已同步，请重启MySQL使新端口生效", req.Port)})
}

func mysqlUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()

	switch r.Method {
	case "GET":
		rows, err := db.Query("SELECT user, host FROM mysql.user")
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "denied") || strings.Contains(errStr, "1142") {
				var currentUser string
				db.QueryRow("SELECT CURRENT_USER()").Scan(&currentUser)
				writeJSON(w, map[string]interface{}{
					"code": 1,
					"msg":  fmt.Sprintf("当前用户 %s 没有查看用户列表的权限，请使用具有SELECT ON mysql.user权限的账号", currentUser),
					"data": []interface{}{
						map[string]interface{}{"user": currentUser, "host": "", "privileges": "(当前登录用户)"},
					},
				})
				return
			}
			writeJSON(w, map[string]interface{}{"code": 1, "msg": errStr})
			return
		}
		defer rows.Close()

		var users []map[string]interface{}
		for rows.Next() {
			var user, host string
			rows.Scan(&user, &host)
			if user == "mysql.infoschema" || user == "mysql.session" || user == "mysql.sys" || user == server.Username {
				continue
			}
			users = append(users, map[string]interface{}{
				"user":       user,
				"host":       host,
				"privileges": "",
			})
		}
		writeJSON(w, map[string]interface{}{"code": 0, "data": users})

	case "POST":
		var req struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			Host       string `json:"host"`
			Privileges string `json:"privileges"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
			return
		}

		if req.Host == "" {
			req.Host = "%"
		}

		if len(req.Password) > 0 && len(req.Password) < 6 {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "密码长度不能少于6位"})
			return
		}

		safeUser := validateIdentity(req.Username)
		safeHost := validateIdentity(req.Host)
		if safeUser == "" || safeHost == "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名或主机名包含非法字符"})
			return
		}

		safePass := escapeSQLString(req.Password)
		_, err := db.Exec(fmt.Sprintf("CREATE USER '%s'@'%s' IDENTIFIED BY '%s'", safeUser, safeHost, safePass))
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "创建用户失败: " + err.Error()})
			return
		}

		if req.Privileges != "" {
			privs := strings.ToUpper(req.Privileges)
			if privs == "ALL" {
				privs = "ALL PRIVILEGES"
			}
			_, err = db.Exec(fmt.Sprintf("GRANT %s ON *.* TO '%s'@'%s'", privs, safeUser, safeHost))
			if err != nil {
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "授权失败: " + err.Error()})
				return
			}
		}

		db.Exec("FLUSH PRIVILEGES")
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "用户创建成功"})

	case "DELETE":
		user := r.URL.Query().Get("user")
		host := r.URL.Query().Get("host")
		if user == "" || host == "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "user and host required"})
			return
		}

		if user == "root" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "不允许删除 root 用户"})
			return
		}

		safeUser := validateIdentity(user)
		safeHost := validateIdentity(host)
		if safeUser == "" || safeHost == "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名或主机名包含非法字符"})
			return
		}

		_, err := db.Exec(fmt.Sprintf("DROP USER '%s'@'%s'", safeUser, safeHost))
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "删除用户失败: " + err.Error()})
			return
		}
		db.Exec("FLUSH PRIVILEGES")
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "用户删除成功"})
	}
}

func mysqlGrantHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "PUT, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	user := r.URL.Query().Get("user")
	host := r.URL.Query().Get("host")
	source := r.URL.Query().Get("source")
	if serverID == "" || user == "" || host == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id, user and host required"})
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()

	var req struct {
		Privileges string `json:"privileges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	safeUser := validateIdentity(user)
	safeHost := validateIdentity(host)
	if safeUser == "" || safeHost == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名或主机名包含非法字符"})
		return
	}

	privs := strings.ToUpper(req.Privileges)
	if privs == "ALL" || privs == "ALL PRIVILEGES" {
		privs = "ALL PRIVILEGES"
	}

	_, err = db.Exec(fmt.Sprintf("GRANT %s ON *.* TO '%s'@'%s'", privs, safeUser, safeHost))
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "授权失败: " + err.Error()})
		return
	}

	db.Exec("FLUSH PRIVILEGES")
	msg := "权限修改成功"
	if privs == "ALL PRIVILEGES" {
		msg += "（注意：已授予全部权限，请确认操作意图）"
	}
	writeJSON(w, map[string]interface{}{"code": 0, "msg": msg})
}

func mysqlRestartHandler(w http.ResponseWriter, r *http.Request) {
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

	containerName := findContainerName(uint(id), source)

	if containerName != "" {
		if dockerPath, err := exec.LookPath("docker"); err == nil {
			out, err := exec.Command(dockerPath, "restart", containerName).CombinedOutput()
			if err != nil {
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "Docker重启失败: " + strings.TrimSpace(string(out))})
				return
			}
			writeJSON(w, map[string]interface{}{"code": 0, "msg": "MySQL重启成功(Docker容器 " + containerName + ")"})
			return
		}
	}

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}

	var basedir string
	db.QueryRow("SHOW VARIABLES LIKE 'basedir'").Scan(new(string), &basedir)

	_, err = db.Exec("SHUTDOWN")
	db.Close()
	if err != nil && !strings.Contains(err.Error(), "server closed") {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "关闭失败: " + err.Error()})
		return
	}

	time.Sleep(2 * time.Second)

	var startErr string
	if _, err := exec.LookPath("systemctl"); err == nil {
		if err := exec.Command("systemctl", "start", "mysql").Run(); err != nil {
			if err := exec.Command("systemctl", "start", "mysqld").Run(); err != nil {
				if err := exec.Command("systemctl", "start", "mariadb").Run(); err != nil {
					startErr = err.Error()
				}
			}
		}
	} else if _, err := exec.LookPath("service"); err == nil {
		if err := exec.Command("service", "mysql", "start").Run(); err != nil {
			if err := exec.Command("service", "mysqld", "start").Run(); err != nil {
				if err := exec.Command("service", "mariadb", "start").Run(); err != nil {
					startErr = err.Error()
				}
			}
		}
	} else if basedir != "" {
		mysqldPath := filepath.Join(basedir, "bin", "mysqld")
		if _, err := os.Stat(mysqldPath); err == nil {
			if err := exec.Command(mysqldPath, "--daemonize").Start(); err != nil {
				startErr = err.Error()
			}
		} else {
			startErr = "找不到mysqld启动命令"
		}
	} else {
		startErr = "找不到系统服务管理器"
	}

	if startErr != "" {
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "MySQL已关闭，自动启动失败(" + startErr + ")，请手动启动"})
		return
	}
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "MySQL重启成功"})
}

func mysqlStopHandler(w http.ResponseWriter, r *http.Request) {
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

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[recover] mysql stop panic: %v\n", r)
			}
		}()
		db.Exec("SHUTDOWN")
	}()
	db.Close()

	writeJSON(w, map[string]interface{}{"code": 0, "msg": "停止指令已发送"})
}

func mysqlPingHandler(w http.ResponseWriter, r *http.Request) {
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

	svc := GetHealthService()
	uid := fmt.Sprintf("%s:%d", map[string]string{"remote": "r", "local": "l"}[source], id)
	if svc.IsCacheValid(uid, 30) {
		if st := svc.GetStatus(uid); st != nil {
			if st.Online {
				writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": true, "latencyMs": st.LatencyMs, "checkedAt": st.CheckedAt}})
			} else {
				writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false, "latencyMs": st.LatencyMs, "checkedAt": st.CheckedAt}})
			}
			return
		}
	}

	server := findAnyServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	online := checkMySQLOnline(server)
	if online {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": true}})
	} else {
		if source != "remote" {
			containerName := findContainerName(uint(id), source)
			if containerName != "" {
				dockerPortMap := getDockerPortMapping()
				if info, ok := dockerPortMap[containerName]; ok && info.MysqlPort > 0 && info.MysqlPort != server.Port {
					mutex.Lock()
					for i := range databases {
						if databases[i].ID == uint(id) {
							databases[i].Port = info.MysqlPort
							break
						}
					}
					mutex.Unlock()
					saveData()
					writeJSON(w, map[string]interface{}{
						"code": 0,
						"data": map[string]interface{}{
							"online":      true,
							"portChanged": true,
							"newPort":     info.MysqlPort,
						},
					})
					return
				}
			}
			newPort := detectMySQLPort(server.Host, server.Username, server.Password, server.Port)
			if newPort > 0 {
				mutex.Lock()
				for i := range databases {
					if databases[i].ID == uint(id) {
						databases[i].Port = newPort
						break
					}
				}
				mutex.Unlock()
				saveData()
				writeJSON(w, map[string]interface{}{
					"code": 0,
					"data": map[string]interface{}{
						"online":      true,
						"portChanged": true,
						"newPort":     newPort,
					},
				})
				return
			}
		}
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false}})
	}
}

func escapeSQLString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\x00", "\\0")
	return s
}

func extractLogTime(line string, logType string) string {
	if strings.HasPrefix(line, "# Time:") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			return strings.TrimSpace(parts[2])
		}
	}
	if len(line) > 19 {
		maybeTime := line[:19]
		if _, err := time.Parse("2006-01-02T15:04:05", maybeTime); err == nil {
			return maybeTime
		}
	}
	if len(line) > 26 {
		maybeTime := line[:26]
		if _, err := time.Parse("2006-01-02T15:04:05.000000", maybeTime); err == nil {
			return maybeTime
		}
	}
	// 如果没有找到时间戳，返回当前时间
	return time.Now().Format("2006-01-02 15:04:05")
}