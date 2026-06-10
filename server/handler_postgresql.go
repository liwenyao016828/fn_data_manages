package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

func formatPostgreSQLError(prefix string, err error) string {
	errStr := err.Error()
	lower := strings.ToLower(errStr)

	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "must be owner") || strings.Contains(lower, "must be superuser") {
		return prefix + "：当前账号权限不足，请使用具有超级用户权限的 PostgreSQL 账号"
	}
	if strings.Contains(lower, "already exists") {
		return prefix + "：该对象已存在，请更换名称"
	}
	if strings.Contains(lower, "does not exist") {
		return prefix + "：指定的对象不存在"
	}
	if strings.Contains(lower, "syntax") {
		return prefix + "：SQL 语法错误，可能是 PostgreSQL 版本不兼容"
	}
	if strings.Contains(lower, "password") && (strings.Contains(lower, "authentication") || strings.Contains(lower, "failed")) {
		return prefix + "：密码认证失败，请检查密码是否符合策略要求"
	}
	if strings.Contains(lower, "connection") && strings.Contains(lower, "refused") {
		return prefix + "：无法连接到 PostgreSQL 服务器，请检查网络或服务器状态"
	}
	if strings.Contains(lower, "timeout") {
		return prefix + "：连接超时，请检查网络或服务器状态"
	}
	if strings.Contains(lower, "denied") {
		return prefix + "：操作被拒绝，请检查当前账号权限"
	}

	return prefix + "：" + errStr
}

func postgresqlDatabasesHandler(w http.ResponseWriter, r *http.Request) {
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

	db, err := openPostgreSQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
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

func postgresqlCreateDatabaseHandler(w http.ResponseWriter, r *http.Request) {
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
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[创建数据库-PostgreSQL] 解析请求体失败: %v\n", err)
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")
	fmt.Printf("[创建数据库-PostgreSQL] 收到请求: server_id=%d, name=%s, hasPassword=%v, source=%s\n", req.ServerID, req.Name, req.Password != "", source)

	if req.ServerID == 0 || req.Name == "" {
		fmt.Printf("[创建数据库-PostgreSQL] 参数错误: server_id=%d, name=%s\n", req.ServerID, req.Name)
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and name required"})
		return
	}

	if err := validateIdentifier(req.Name); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "数据库名验证失败: " + err.Error()})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		fmt.Printf("[创建数据库-PostgreSQL] 未找到服务器: server_id=%d, source=%s\n", req.ServerID, source)
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openPostgreSQL(server)
	if err != nil {
		fmt.Printf("[创建数据库-PostgreSQL] 连接PostgreSQL失败: %v\n", err)
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	createSQL := fmt.Sprintf("CREATE DATABASE %s WITH ENCODING 'UTF8'", pq.QuoteIdentifier(req.Name))
	fmt.Printf("[创建数据库-PostgreSQL] 执行SQL: ***\n")
	_, err = db.Exec(createSQL)
	if err != nil {
		fmt.Printf("[创建数据库-PostgreSQL] 创建数据库失败: %v\n", err)
		sysLogError("POSTGRESQL", fmt.Sprintf("创建数据库失败: %s (连接: %s:%d)", req.Name, server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatPostgreSQLError("创建数据库失败", err)})
		return
	}
	fmt.Printf("[创建数据库-PostgreSQL] 数据库创建成功: %s\n", req.Name)

	if req.Password != "" {
		fmt.Printf("[创建数据库-PostgreSQL] 开始创建用户: %s\n", req.Name)
		if err := validateMySQLUser(req.Name); err != nil {
			fmt.Printf("[创建数据库-PostgreSQL] 用户名验证失败: %v\n", err)
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名验证失败: " + err.Error()})
			return
		}
		if len(req.Password) < 6 {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "密码长度不能少于6位"})
			return
		}
		createUserSQL, err := buildPostgresCreateUser(req.Name)
		if err != nil {
			fmt.Printf("[创建数据库-PostgreSQL] 用户名验证失败: %v\n", err)
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名验证失败: " + err.Error()})
			return
		}
		fmt.Printf("[创建数据库-PostgreSQL] 创建用户: %s\n", req.Name)
		_, err = db.Exec(createUserSQL, req.Password)
		if err != nil {
			fmt.Printf("[创建数据库-PostgreSQL] 创建用户失败: %v\n", err)
			sysLogError("USER", fmt.Sprintf("创建用户失败: %s (连接: %s:%d)", req.Name, server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": formatPostgreSQLError("创建用户失败", err)})
			return
		}
		fmt.Printf("[创建数据库-PostgreSQL] 用户创建成功: %s\n", req.Name)

		grantSQL := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s", pq.QuoteIdentifier(req.Name), pq.QuoteIdentifier(req.Name))
		fmt.Printf("[创建数据库-PostgreSQL] 授权SQL: ***\n")
		_, err = db.Exec(grantSQL)
		if err != nil {
			fmt.Printf("[创建数据库-PostgreSQL] 授权失败: %v\n", err)
			sysLogError("USER", fmt.Sprintf("授权失败: %s (连接: %s:%d)", req.Name, server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": formatPostgreSQLError("授权失败", err)})
			return
		}
		fmt.Printf("[创建数据库-PostgreSQL] 授权成功\n")
	}

	fmt.Printf("[创建数据库-PostgreSQL] 全部完成\n")
	sysLogInfo("POSTGRESQL", fmt.Sprintf("创建数据库: %s (连接: %s:%d)", req.Name, server.Host, server.Port))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "创建成功"})
}

func postgresqlDeleteDatabaseHandler(w http.ResponseWriter, r *http.Request) {
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
		fmt.Printf("[删除数据库-PostgreSQL] 解析请求体失败: %v\n", err)
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	source := r.URL.Query().Get("source")
	fmt.Printf("[删除数据库-PostgreSQL] 收到请求: server_id=%d, name=%s, source=%s\n", req.ServerID, req.Name, source)

	if req.ServerID == 0 || req.Name == "" {
		fmt.Printf("[删除数据库-PostgreSQL] 参数错误: server_id=%d, name=%s\n", req.ServerID, req.Name)
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and name required"})
		return
	}

	if err := validateIdentifier(req.Name); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "数据库名验证失败: " + err.Error()})
		return
	}

	server := findAnyServer(uint(req.ServerID), source)
	if server == nil {
		fmt.Printf("[删除数据库-PostgreSQL] 未找到服务器: server_id=%d, source=%s\n", req.ServerID, source)
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	db, err := openPostgreSQL(server)
	if err != nil {
		fmt.Printf("[删除数据库-PostgreSQL] 连接PostgreSQL失败: %v\n", err)
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	// 检查是否有活跃连接
	var activeConns int
	err = db.QueryRow("SELECT COUNT(*) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()", req.Name).Scan(&activeConns)
	if err == nil && activeConns > 0 {
		// 终止活跃连接
		_, _ = db.Exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()", req.Name)
	}

	dropSQL := fmt.Sprintf("DROP DATABASE %s", pq.QuoteIdentifier(req.Name))
	fmt.Printf("[删除数据库-PostgreSQL] 执行SQL: %s\n", dropSQL)
	_, err = db.Exec(dropSQL)
	if err != nil {
		fmt.Printf("[删除数据库-PostgreSQL] 执行SQL失败: %v\n", err)
		sysLogError("POSTGRESQL", fmt.Sprintf("删除数据库失败: %s (连接: %s:%d)", req.Name, server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": formatPostgreSQLError("删除数据库失败", err)})
		return
	}

	fmt.Printf("[删除数据库-PostgreSQL] 成功删除数据库: %s\n", req.Name)
	sysLogInfo("POSTGRESQL", fmt.Sprintf("删除数据库: %s (连接: %s:%d)", req.Name, server.Host, server.Port))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "删除成功"})
}

func postgresqlTablesHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := validateIdentifier(dbName); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "数据库名验证失败: " + err.Error()})
		return
	}

	// 连接到指定数据库
	db, err := openPostgreSQLWithDB(server, dbName)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
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

func postgresqlColumnsHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := validateIdentifier(dbName); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "数据库名验证失败: " + err.Error()})
		return
	}
	if err := validateIdentifier(tableName); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "表名验证失败: " + err.Error()})
		return
	}

	// 连接到指定数据库
	db, err := openPostgreSQLWithDB(server, dbName)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query(
		"SELECT column_name, data_type, is_nullable, column_default, character_maximum_length FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1",
		tableName,
	)
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

func postgresqlDataHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := validateIdentifier(dbName); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "数据库名验证失败: " + err.Error()})
		return
	}
	if err := validateIdentifier(tableName); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "表名验证失败: " + err.Error()})
		return
	}

	// 连接到指定数据库
	db, err := openPostgreSQLWithDB(server, dbName)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	var total int64
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", pq.QuoteIdentifier(tableName))
	err = db.QueryRow(countSQL).Scan(&total)
	if err != nil {
		total = 0
	}

	offset := (page - 1) * pageSize
	selectSQL := fmt.Sprintf("SELECT * FROM %s LIMIT $1 OFFSET $2", pq.QuoteIdentifier(tableName))
	rows, err := db.Query(selectSQL, pageSize, offset)
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

func postgresqlExecuteHandler(w http.ResponseWriter, r *http.Request) {
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

	// 连接到指定数据库（如果提供了）
	var db *sql.DB
	var err error
	if req.Database != "" {
		if err := validateIdentifier(req.Database); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "数据库名验证失败: " + err.Error()})
			return
		}
		db, err = openPostgreSQLWithDB(server, req.Database)
	} else {
		db, err = openPostgreSQL(server)
	}
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	sqlTrimmed := strings.TrimSpace(req.SQL)
	sqlUpper := strings.ToUpper(sqlTrimmed)

	dangerousKeywords := []string{
		"DROP DATABASE", "DROP TABLE", "DROP VIEW", "DROP PROCEDURE", "DROP FUNCTION", "DROP TRIGGER", "DROP INDEX",
		"ALTER DATABASE", "ALTER USER",
		"TRUNCATE ", "RENAME ",
		"CREATE DATABASE", "CREATE USER", "CREATE TABLESPACE",
		"GRANT ", "REVOKE ", "KILL ", "SHUTDOWN",
		"LOAD DATA ", "INTO OUTFILE", "INTO DUMPFILE",
		"COPY ", "ALTER SYSTEM",
	}
	for _, kw := range dangerousKeywords {
		if strings.Contains(sqlUpper, kw) {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": fmt.Sprintf("禁止执行危险操作: %s", strings.TrimSpace(kw))})
			return
		}
	}

	sqlPreview := req.SQL
	if len(sqlPreview) > 50 {
		sqlPreview = sqlPreview[:50] + "..."
	}
	sysLogInfo("POSTGRESQL", fmt.Sprintf("执行SQL: %s (连接: %s:%d)", sqlPreview, server.Host, server.Port))

	if strings.HasPrefix(sqlUpper, "SELECT") || strings.HasPrefix(sqlUpper, "SHOW") || strings.HasPrefix(sqlUpper, "DESCRIBE") || strings.HasPrefix(sqlUpper, "EXPLAIN") {
		// 对 SELECT 添加 LIMIT
		querySQL := req.SQL
		if strings.HasPrefix(sqlUpper, "SELECT") && !strings.Contains(sqlUpper, "LIMIT") {
			querySQL = req.SQL + " LIMIT 1000"
		}
		rows, err := db.Query(querySQL)
		if err != nil {
			sysLogWarn("POSTGRESQL", fmt.Sprintf("SQL查询失败: %s (连接: %s:%d)", truncateSQL(req.SQL), server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
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
			sysLogError("POSTGRESQL", fmt.Sprintf("SQL执行失败: %s (连接: %s:%d)", truncateSQL(req.SQL), server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
			return
		}
		affected, _ := result.RowsAffected()
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"affected": affected, "msg": "执行成功"}})
	}
}

func postgresqlUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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

	db, err := openPostgreSQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()

	switch r.Method {
	case "GET":
		rows, err := db.Query("SELECT usename, usesuper, valuntil FROM pg_user")
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
			return
		}
		defer rows.Close()

		var users []map[string]interface{}
		for rows.Next() {
			var username string
			var isSuper bool
			var valUntil sql.NullString
			rows.Scan(&username, &isSuper, &valUntil)
			if username == "postgres" || username == server.Username {
				continue
			}
			users = append(users, map[string]interface{}{
				"user":       username,
				"isSuper":    isSuper,
				"valUntil":   valUntil.String,
				"privileges": "",
			})
		}

		// 获取权限信息
		privRows, err := db.Query("SELECT rolname, rolcreaterole, rolcreatedb, rolinherit, rolcanlogin FROM pg_roles WHERE rolname NOT LIKE 'pg_%'")
		if err == nil {
			defer privRows.Close()
			privMap := make(map[string][]string)
			for privRows.Next() {
				var rolName string
				var createRole, createDB, inherit, canLogin bool
				privRows.Scan(&rolName, &createRole, &createDB, &inherit, &canLogin)
				var privs []string
				if createRole {
					privs = append(privs, "CREATEROLE")
				}
				if createDB {
					privs = append(privs, "CREATEDB")
				}
				if inherit {
					privs = append(privs, "INHERIT")
				}
				if canLogin {
					privs = append(privs, "LOGIN")
				}
				privMap[rolName] = privs
			}
			for i := range users {
				if privs, ok := privMap[users[i]["user"].(string)]; ok && len(privs) > 0 {
					users[i]["privileges"] = strings.Join(privs, ", ")
				}
			}
		}

		writeJSON(w, map[string]interface{}{"code": 0, "data": users})

	case "POST":
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
			return
		}

		if req.Username == "" || req.Password == "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "username and password required"})
			return
		}

		if len(req.Password) < 6 {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "密码长度不能少于6位"})
			return
		}

		if err := validateMySQLUser(req.Username); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名验证失败: " + err.Error()})
			return
		}

		createUserSQL, err := buildPostgresCreateUser(req.Username)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "创建用户参数验证失败: " + err.Error()})
			return
		}
		sysLogInfo("POSTGRESQL", fmt.Sprintf("创建用户: %s (连接: %s:%d)", req.Username, server.Host, server.Port))
		_, err = db.Exec(createUserSQL, req.Password)
		if err != nil {
			sysLogWarn("POSTGRESQL", fmt.Sprintf("创建用户失败: %s, 错误: %s", req.Username, err.Error()))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": formatPostgreSQLError("创建用户失败", err)})
			return
		}

		writeJSON(w, map[string]interface{}{"code": 0, "msg": fmt.Sprintf("用户 %s 创建成功", req.Username)})

	case "DELETE":
		username := r.URL.Query().Get("user")
		if username == "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "user required"})
			return
		}

		if username == "postgres" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "不允许删除 postgres 用户"})
			return
		}

		if err := validateMySQLUser(username); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名验证失败: " + err.Error()})
			return
		}

		dropSQL := fmt.Sprintf("DROP USER %s", pq.QuoteIdentifier(username))
		_, err = db.Exec(dropSQL)
		if err != nil {
			sysLogError("USER", fmt.Sprintf("删除用户失败: %s (连接: %s:%d)", username, server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": formatPostgreSQLError("删除用户失败", err)})
			return
		}
		sysLogInfo("POSTGRESQL", fmt.Sprintf("删除用户: %s (连接: %s:%d)", username, server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "用户删除成功"})

	case "PUT":
		username := r.URL.Query().Get("user")
		if username == "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "user required"})
			return
		}

		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
			return
		}

		if len(req.Password) < 6 {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "密码长度不能少于6位"})
			return
		}

		if err := validateMySQLUser(username); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "用户名验证失败: " + err.Error()})
			return
		}

		alterSQL, err := buildPostgresAlterUser(username)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "修改密码参数验证失败: " + err.Error()})
			return
		}
		_, err = db.Exec(alterSQL, req.Password)
		if err != nil {
			sysLogError("USER", fmt.Sprintf("修改密码失败: %s (连接: %s:%d)", username, server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": formatPostgreSQLError("修改密码失败", err)})
			return
		}
		sysLogInfo("POSTGRESQL", fmt.Sprintf("修改用户密码: %s (连接: %s:%d)", username, server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "密码修改成功"})

	default:
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
	}
}

func postgresqlConfigHandler(w http.ResponseWriter, r *http.Request) {
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
		db, err := openPostgreSQL(server)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
			return
		}
		defer db.Close()

		rows, err := db.Query("SELECT name, setting, unit, source FROM pg_settings ORDER BY name")
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "查询配置失败: " + err.Error()})
			return
		}
		defer rows.Close()

		var settings []map[string]interface{}
		for rows.Next() {
			var name, setting, unit, sourceVal sql.NullString
			rows.Scan(&name, &setting, &unit, &sourceVal)
			settings = append(settings, map[string]interface{}{
				"name":    name.String,
				"setting": setting.String,
				"unit":    unit.String,
				"source":  sourceVal.String,
			})
		}

		// 获取版本信息
		var version string
		db.QueryRow("SELECT version()").Scan(&version)

		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"settings": settings,
				"version":  version,
				"source":   "pg_settings",
			},
		})

	} else if r.Method == "PUT" {
		var req struct {
			Variables map[string]string `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Variables) == 0 {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "variables required"})
			return
		}

		db, err := openPostgreSQL(server)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
			return
		}
		defer db.Close()

		var updated []string
		var failed []string
		for k, v := range req.Variables {
			if err := validateVariableName(k); err != nil {
				failed = append(failed, k)
				continue
			}
			setSQL := fmt.Sprintf("ALTER SYSTEM SET %s = %s", pq.QuoteIdentifier(k), quoteString(v))
			_, err := db.Exec(setSQL)
			if err != nil {
				sysLogError("POSTGRESQL", fmt.Sprintf("修改配置失败: %s = %s (连接: %s:%d)", k, v, server.Host, server.Port))
				failed = append(failed, k)
			} else {
				updated = append(updated, k)
				sysLogInfo("POSTGRESQL", fmt.Sprintf("修改PostgreSQL配置: %s = %s (连接: %s:%d)", k, v, server.Host, server.Port))
			}
		}

		// 重载配置
		if len(updated) > 0 {
			_, _ = db.Exec("SELECT pg_reload_conf()")
		}

		msg := ""
		if len(updated) > 0 {
			msg += fmt.Sprintf("已修改: %s", strings.Join(updated, ", "))
		}
		if len(failed) > 0 {
			msg += fmt.Sprintf("；修改失败(可能为只读参数): %s", strings.Join(failed, ", "))
		}
		writeJSON(w, map[string]interface{}{"code": 0, "msg": msg})
	}
}

func postgresqlPingHandler(w http.ResponseWriter, r *http.Request) {
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

	start := time.Now()
	db, err := openPostgreSQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false, "error": err.Error()}})
		return
	}
	defer db.Close()

	err = db.Ping()
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false, "latencyMs": latencyMs, "error": err.Error()}})
		return
	}

	// 获取版本信息
	var version string
	db.QueryRow("SELECT version()").Scan(&version)

	writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{
		"online":    true,
		"latencyMs": latencyMs,
		"version":   version,
	}})
}

func postgresqlRestartHandler(w http.ResponseWriter, r *http.Request) {
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

	sysLogInfo("POSTGRESQL", fmt.Sprintf("重启PostgreSQL (连接: %s:%d)", server.Host, server.Port))

	containerName := findContainerName(uint(id), source)

	if containerName != "" {
		if dockerPath, err := exec.LookPath("docker"); err == nil {
			out, err := exec.Command(dockerPath, "restart", containerName).CombinedOutput()
			if err != nil {
				sysLogError("POSTGRESQL", fmt.Sprintf("Docker重启失败: %s (连接: %s:%d)", strings.TrimSpace(string(out)), server.Host, server.Port))
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "Docker重启失败: " + strings.TrimSpace(string(out))})
				return
			}
			writeJSON(w, map[string]interface{}{"code": 0, "msg": "PostgreSQL重启成功(Docker容器 " + containerName + ")"})
			return
		}
	}

	var startErr string
	if _, err := exec.LookPath("systemctl"); err == nil {
		if err := exec.Command("systemctl", "restart", "postgresql").Run(); err != nil {
			if err := exec.Command("systemctl", "restart", "postgres").Run(); err != nil {
				startErr = err.Error()
			}
		}
	} else if _, err := exec.LookPath("service"); err == nil {
		if err := exec.Command("service", "postgresql", "restart").Run(); err != nil {
			if err := exec.Command("service", "postgres", "restart").Run(); err != nil {
				startErr = err.Error()
			}
		}
	} else {
		startErr = "找不到系统服务管理器"
	}

	if startErr != "" {
		sysLogError("POSTGRESQL", fmt.Sprintf("PostgreSQL重启失败: %s (连接: %s:%d)", startErr, server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "重启失败: " + startErr})
		return
	}
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "PostgreSQL重启成功"})
}

func postgresqlLogsHandler(w http.ResponseWriter, r *http.Request) {
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

	db, err := openPostgreSQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接PostgreSQL失败: " + err.Error()})
		return
	}
	defer db.Close()

	var logs []map[string]string
	containerName := findContainerName(uint(id), source)

	// 获取日志目录
	var logDir string
	dirRows, dirErr := db.Query("SELECT setting FROM pg_settings WHERE name = 'log_directory'")
	if dirErr == nil {
		var val string
		for dirRows.Next() {
			dirRows.Scan(&val)
		}
		dirRows.Close()
		logDir = val
	}

	// 获取数据目录
	var dataDir string
	dataRows, dataErr := db.Query("SELECT setting FROM pg_settings WHERE name = 'data_directory'")
	if dataErr == nil {
		var val string
		for dataRows.Next() {
			dataRows.Scan(&val)
		}
		dataRows.Close()
		dataDir = val
	}

	// 获取日志文件名
	var logFilename string
	fileRows, fileErr := db.Query("SELECT setting FROM pg_settings WHERE name = 'log_filename'")
	if fileErr == nil {
		var val string
		for fileRows.Next() {
			fileRows.Scan(&val)
		}
		fileRows.Close()
		logFilename = val
	}

	// 解析日志路径（使用 POSIX 路径，避免 Windows filepath 将 / 转为 \）
	resolveLogPath := func() string {
		if logDir == "" || logFilename == "" {
			return ""
		}
		if strings.HasPrefix(logDir, "/") {
			return joinPosix(logDir, logFilename)
		}
		if dataDir != "" {
			return joinPosix(dataDir, logDir, logFilename)
		}
		return joinPosix(logDir, logFilename)
	}

	// 远程服务器无法直接读取日志文件
	if source == "remote" {
		logPath := resolveLogPath()
		msg := "远程服务器日志文件无法直接读取"
		if logPath != "" {
			msg += "，日志路径: " + logPath
		}
		logs = append(logs, map[string]string{
			"time":    "",
			"level":   "Note",
			"message": msg,
		})
		writeJSON(w, map[string]interface{}{"code": 0, "data": logs})
		return
	}

	logPath := resolveLogPath()
	if logPath == "" {
		logs = append(logs, map[string]string{
			"time":    "",
			"level":   "Note",
			"message": "无法获取PostgreSQL日志路径，请检查 logging_collector 和 log_filename 配置",
		})
		writeJSON(w, map[string]interface{}{"code": 0, "data": logs})
		return
	}

	// 尝试读取日志文件：先本地，再容器
	data, readErr := readLocalOrContainerFile(containerName, logPath, server.Host, server.Port)

	// 如果文件名含日期占位符（如 %Y-%m-%d），尝试解析为当天日期
	if readErr != nil && strings.Contains(logFilename, "%") {
		now := time.Now()
		resolvedFilename := logFilename
		resolvedFilename = strings.ReplaceAll(resolvedFilename, "%Y", now.Format("2006"))
		resolvedFilename = strings.ReplaceAll(resolvedFilename, "%m", now.Format("01"))
		resolvedFilename = strings.ReplaceAll(resolvedFilename, "%d", now.Format("02"))
		resolvedFilename = strings.ReplaceAll(resolvedFilename, "%H", now.Format("15"))

		if strings.HasPrefix(logDir, "/") {
			logPath = joinPosix(logDir, resolvedFilename)
		} else if dataDir != "" {
			logPath = joinPosix(dataDir, logDir, resolvedFilename)
		} else {
			logPath = joinPosix(logDir, resolvedFilename)
		}
		data, readErr = readLocalOrContainerFile(containerName, logPath, server.Host, server.Port)
	}

	// 如果仍然失败且有容器名，尝试在容器内用 ls 找到最新的日志文件
	if readErr != nil && containerName != "" {
		dockerPath, dockerErr := exec.LookPath("docker")
		if dockerErr == nil {
			var logDirFull string
			if strings.HasPrefix(logDir, "/") {
				logDirFull = logDir
			} else if dataDir != "" {
				logDirFull = joinPosix(dataDir, logDir)
			}
			if logDirFull != "" {
				lsOut, lsErr := exec.Command(dockerPath, "exec", containerName, "ls", "-t", logDirFull).CombinedOutput()
				if lsErr == nil {
					files := strings.Split(strings.TrimSpace(string(lsOut)), "\n")
					for _, f := range files {
						f = strings.TrimSpace(f)
						if f != "" && strings.HasSuffix(f, ".log") {
							foundPath := joinPosix(logDirFull, f)
							data, readErr = readContainerFile(containerName, foundPath)
							if readErr == nil {
								break
							}
						}
					}
				}
			}
		}
	}

	if readErr != nil {
		logs = append(logs, map[string]string{
			"time":    "",
			"level":   "Warning",
			"message": "无法读取PostgreSQL日志文件: " + logPath + " (" + readErr.Error() + ")",
		})
		writeJSON(w, map[string]interface{}{"code": 0, "data": logs})
		return
	}

	for _, entry := range parseLogLines(data, 500) {
		line := entry["message"]
		level := "Note"
		upperLine := strings.ToUpper(line)
		if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "FATAL") || strings.Contains(upperLine, "PANIC") {
			level = "Error"
		} else if strings.Contains(upperLine, "WARNING") || strings.Contains(upperLine, "WARN") {
			level = "Warning"
		}

		logTime := ""
		if len(line) > 23 {
			maybeTime := line[:23]
			if _, err := time.Parse("2006-01-02 15:04:05.000", maybeTime); err == nil {
				logTime = maybeTime
			}
		}
		if logTime == "" && len(line) > 19 {
			maybeTime := line[:19]
			if _, err := time.Parse("2006-01-02 15:04:05", maybeTime); err == nil {
				logTime = maybeTime
			}
		}

		logs = append(logs, map[string]string{
			"time":    logTime,
			"level":   level,
			"message": line,
		})
	}

	if len(logs) == 0 {
		logs = append(logs, map[string]string{"time": "", "level": "Note", "message": "暂无日志数据"})
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": logs})
}
