package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func testMySQLConnection(host string, port int, username, password string) (string, string) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=3s&readTimeout=3s&allowNativePasswords=true",
		username, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", fmt.Sprintf("打开连接失败: %v", err)
	}
	defer db.Close()

	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		return "", fmt.Sprintf("执行VERSION()查询失败: %v", err)
	}
	return version, ""
}

func testPostgreSQLConnection(host string, port int, username, password string) (string, string) {
	sslmode := "disable"
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=%s&connect_timeout=3",
		username, password, host, port, sslmode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return "", fmt.Sprintf("打开连接失败: %v", err)
	}
	defer db.Close()

	var version string
	if err := db.QueryRow("SELECT version()").Scan(&version); err != nil {
		return "", fmt.Sprintf("执行version()查询失败: %v", err)
	}
	parts := strings.Split(version, ",")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0]), ""
	}
	return version, ""
}

func testSQLiteConnection(filePath string) (string, string) {
	if filePath == "" {
		return "", "文件路径为空"
	}
	dsn := filePath + "?mode=ro&_busy_timeout=3000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return "", fmt.Sprintf("打开SQLite文件失败: %v", err)
	}
	defer db.Close()

	var version string
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&version); err != nil {
		return "", fmt.Sprintf("执行sqlite_version()查询失败: %v", err)
	}
	return "SQLite " + version, ""
}

func testRedisConnection(host string, port int, username, password string) (bool, string) {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false, fmt.Sprintf("TCP连接失败: %v", err)
	}
	defer conn.Close()

	if username != "" && password != "" {
		resp, err := redisDo(conn, "AUTH", username, password)
		if err != nil {
			return false, fmt.Sprintf("AUTH认证失败: %v", err)
		}
		if s, ok := resp.(string); ok && s != "OK" {
			return false, fmt.Sprintf("AUTH认证返回非OK: %s", s)
		}
	} else if password != "" {
		resp, err := redisDo(conn, "AUTH", password)
		if err != nil {
			return false, fmt.Sprintf("AUTH认证失败: %v", err)
		}
		if s, ok := resp.(string); ok && s != "OK" {
			return false, fmt.Sprintf("AUTH认证返回非OK: %s", s)
		}
	}

	resp, err := redisDo(conn, "PING")
	if err != nil {
		return false, fmt.Sprintf("PING命令失败: %v", err)
	}
	if pong, ok := resp.(string); ok && pong == "PONG" {
		return true, ""
	}
	return false, fmt.Sprintf("PING返回非PONG: %v", resp)
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case "POST":
		createHandler(w, r)
	case "PUT":
		updateHandler(w, r)
	case "DELETE":
		deleteHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	var req DatabaseCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	if req.TestOnly {
		host := req.Host
		port := req.Port
		username := req.Username
		password := req.Password
		dbType := req.Type

		if req.TestID > 0 {
			server := findAnyServer(req.TestID, req.TestSource)
			if server != nil {
				if host == "" {
					host = server.Host
					port = server.Port
					username = server.Username
					dbType = server.Type
				}
				if password == "" {
					password = server.Password
				}
			}
		}

		var version string
		var errMsg string
		if dbType == "mysql" || dbType == "mariadb" {
			version, errMsg = testMySQLConnection(host, port, username, password)
		} else if dbType == "redis" {
			var ok bool
			ok, errMsg = testRedisConnection(host, port, username, password)
			if ok {
				version = "connected"
			}
		} else if dbType == "postgresql" {
			version, errMsg = testPostgreSQLConnection(host, port, username, password)
		} else if dbType == "sqlite" {
			version, errMsg = testSQLiteConnection(host)
		}
		if version != "" {
			sysLogInfo("CONNECTION", fmt.Sprintf("连接测试成功 %s (%s:%d, %s)", req.Name, host, port, dbType))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success", "version": version})
		} else {
			detail := errMsg
			if detail == "" {
				detail = "连接失败"
			}
			sysLogError("CONNECTION", fmt.Sprintf("连接测试失败 %s (%s:%d, %s): %s", req.Name, host, port, dbType, detail))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": detail})
		}
		return
	}

	mutex.Lock()
	for _, db := range databases {
		if db.Name == req.Name {
			mutex.Unlock()
			errMsg := fmt.Sprintf("添加连接失败: 名称 '%s' 已存在 (ID=%d)", req.Name, db.ID)
			sysLogError("CONNECTION", errMsg)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "名称已存在"})
			return
		}
		if db.Host == req.Host && db.Port == req.Port && db.Username == req.Username {
			mutex.Unlock()
			errMsg := fmt.Sprintf("添加连接失败: 相同主机(%s)、端口(%d)和用户名(%s)的本地连接已存在 (名称='%s', ID=%d)", req.Host, req.Port, req.Username, db.Name, db.ID)
			sysLogError("CONNECTION", errMsg)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的连接已存在"})
			return
		}
	}
	for _, rs := range remoteServers {
		if rs.Host == req.Host && rs.Port == req.Port && rs.Username == req.Username {
			mutex.Unlock()
			errMsg := fmt.Sprintf("添加连接失败: 相同主机(%s)、端口(%d)和用户名(%s)的远程连接已存在 (名称='%s', ID=%d)", req.Host, req.Port, req.Username, rs.Name, rs.ID)
			sysLogError("CONNECTION", errMsg)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的远程连接已存在"})
			return
		}
	}

	newDB := Database{
		ID:          nextID,
		Name:        req.Name,
		Type:        req.Type,
		Version:     req.Version,
		Host:        req.Host,
		Port:        req.Port,
		Username:    req.Username,
		Password:    req.Password,
		Database:    req.Database,
		SSL:         req.SSL,
		Description: req.Description,
		Permission:  req.Permission,
		Container:   req.Container,
		Status:      "running",
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	nextID++
	databases = append(databases, newDB)
	newID := newDB.ID
	mutex.Unlock()

	go func() {
		server := &RemoteServer{
			Host:     newDB.Host,
			Port:     newDB.Port,
			Username: newDB.Username,
			Password: newDB.Password,
			Type:     newDB.Type,
		}
		ver, dsk, dbs := fetchServerInfo(server)
		if ver != "" || dsk != "" {
			mutex.Lock()
			for i := range databases {
				if databases[i].ID == newID {
					if ver != "" {
						databases[i].Version = ver
					}
					if dsk != "" {
						databases[i].Disk = dsk
					}
					if len(dbs) > 0 {
						databases[i].Databases = dbs
					}
					break
				}
			}
			mutex.Unlock()
		}
		if ver == "" && dsk == "" {
			sysLogWarn("CONNECTION", fmt.Sprintf("添加连接后获取服务器信息失败 %s (%s:%d, %s)", req.Name, req.Host, req.Port, req.Type))
		}
	}()

	saveData()

	sysLogInfo("CONNECTION", fmt.Sprintf("添加本地连接成功 %s (%s:%d, 类型=%s, 用户=%s)", req.Name, req.Host, req.Port, req.Type, req.Username))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	id, ok := req["id"].(float64)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "invalid id"})
		return
	}

	mutex.Lock()
	found := false
	for i, db := range databases {
		if db.ID == uint(id) {
			found = true
			if name, ok := req["name"].(string); ok && name != "" && name != db.Name {
				for _, other := range databases {
					if other.ID != db.ID && other.Name == name {
						mutex.Unlock()
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "名称已存在"})
						return
					}
				}
				databases[i].Name = name
			}
			if host, ok := req["host"].(string); ok && host != "" {
				databases[i].Host = host
			}
			if port, ok := req["port"].(float64); ok && port != 0 {
				databases[i].Port = int(port)
			}
			if username, ok := req["username"].(string); ok && username != "" {
				databases[i].Username = username
			}
			newHost := databases[i].Host
			newPort := databases[i].Port
			newUsername := databases[i].Username
			for _, other := range databases {
				if other.ID != db.ID && other.Host == newHost && other.Port == newPort && other.Username == newUsername {
					mutex.Unlock()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的连接已存在"})
					return
				}
			}
			for _, rs := range remoteServers {
				if rs.Host == newHost && rs.Port == newPort && rs.Username == newUsername {
					mutex.Unlock()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的远程连接已存在"})
					return
				}
			}
			if password, ok := req["password"].(string); ok && password != "" && password != "••••••••" {
				databases[i].Password = password
			}
			if ssl, ok := req["ssl"].(bool); ok {
				databases[i].SSL = ssl
			}
			if desc, ok := req["description"].(string); ok {
				databases[i].Description = desc
			}
			if container, ok := req["container"].(string); ok {
				databases[i].Container = container
			}
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

	sysLogInfo("CONNECTION", fmt.Sprintf("更新连接 #%d", uint(id)))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	id, ok := req["id"].(float64)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "invalid id"})
		return
	}

	mutex.Lock()
	found := false
	var delName, delHost string
	var delPort int
	for i, db := range databases {
		if db.ID == uint(id) {
			delName = db.Name
			delHost = db.Host
			delPort = db.Port
			databases = append(databases[:i], databases[i+1:]...)
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

	sysLogInfo("CONNECTION", fmt.Sprintf("删除连接 %s (%s:%d)", delName, delHost, delPort))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req DatabaseSearch
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	mutex.Lock()
	var filtered []Database
	for _, db := range databases {
		if req.Type != "" && db.Type != req.Type {
			continue
		}
		if req.Info != "" && !contains(db.Name, req.Info) {
			continue
		}
		filtered = append(filtered, db)
	}

	total := int64(len(filtered))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(filtered) {
		filtered = []Database{}
	} else if end > len(filtered) {
		filtered = filtered[start:]
	} else {
		filtered = filtered[start:end]
	}
	for i := range filtered {
		filtered[i].Password = maskPassword(filtered[i].Password)
	}
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": PageResult{Items: filtered, Total: total}})
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	idStr := r.URL.Path[len("/api/databases/db/"):]
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "invalid id"})
		return
	}

	mutex.Lock()
	var result *Database
	for i, db := range databases {
		if db.ID == uint(id) {
			result = &databases[i]
			break
		}
	}
	if result == nil {
		mutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 404, "msg": "not found"})
		return
	}
	reveal := r.URL.Query().Get("reveal") == "true"
	maskedResult := *result
	if reveal {
		maskedResult.Password = result.Password // 明文返回
	} else {
		maskedResult.Password = maskPassword(result.Password)
	}
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": maskedResult})
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	dbType := r.URL.Path[len("/api/databases/db/list/"):]

	mutex.Lock()
	var filtered []Database
	for _, db := range databases {
		if dbType != "" && dbType != "all" && db.Type != dbType {
			continue
		}
		filtered = append(filtered, db)
	}
	mutex.Unlock()

	// 掩码化密码后再返回
	for i := range filtered {
		filtered[i].Password = maskPassword(filtered[i].Password)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": filtered})
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req DatabaseCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	mutex.Lock()
	exists := false
	for _, db := range databases {
		if db.Name == req.Name {
			exists = true
			break
		}
	}
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": !exists})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func fetchServerInfo(server *RemoteServer) (string, string, []string) {
	version, disk := "", ""
	var databases []string

	serverType := strings.ToLower(server.Type)

	if serverType == "redis" {
		conn, err := openRedis(server)
		if err != nil {
			return "", "", nil
		}
		infoResp, _ := redisDo(conn, "INFO", "server")
		if infoStr, ok := infoResp.(string); ok {
			for _, line := range strings.Split(infoStr, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "redis_version:") {
					version = strings.TrimSpace(strings.TrimPrefix(line, "redis_version:"))
				}
			}
		}
		memResp, _ := redisDo(conn, "INFO", "memory")
		if memStr, ok := memResp.(string); ok {
			for _, line := range strings.Split(memStr, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "used_memory_human:") {
					disk = strings.TrimSpace(strings.TrimPrefix(line, "used_memory_human:"))
				}
			}
		}
		conn.Close()
	} else if serverType == "postgresql" {
		db, err := openPostgreSQL(server)
		if err != nil {
			return "", "", nil
		}
		defer db.Close()
		db.QueryRow("SELECT version()").Scan(&version)
		parts := strings.Split(version, ",")
		if len(parts) > 0 {
			version = strings.TrimSpace(parts[0])
		}
		var dbSize sql.NullFloat64
		db.QueryRow("SELECT ROUND(SUM(pg_database_size(datname)) / 1024.0 / 1024.0, 2) FROM pg_database WHERE datistemplate = false").Scan(&dbSize)
		if dbSize.Valid && dbSize.Float64 > 0 {
			if dbSize.Float64 > 1024 {
				disk = fmt.Sprintf("%.2fG", dbSize.Float64/1024)
			} else {
				disk = fmt.Sprintf("%.2fM", dbSize.Float64)
			}
		}
		rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
		if err == nil {
			defer rows.Close()
			var dbName string
			for rows.Next() {
				if err := rows.Scan(&dbName); err == nil {
					databases = append(databases, dbName)
				}
			}
		}
	} else if serverType == "sqlite" {
		db, err := openSQLite(server)
		if err != nil {
			return "", "", nil
		}
		defer db.Close()
		var ver string
		if err := db.QueryRow("SELECT sqlite_version()").Scan(&ver); err == nil {
			version = "SQLite " + ver
		}
		// SQLite 文件大小
		if fi, err := os.Stat(server.Host); err == nil {
			sizeMB := float64(fi.Size()) / 1024.0 / 1024.0
			if sizeMB > 1024 {
				disk = fmt.Sprintf("%.2fG", sizeMB/1024)
			} else {
				disk = fmt.Sprintf("%.2fM", sizeMB)
			}
		}
	} else {
		// MySQL / MariaDB
		fmt.Printf("[fetchServerInfo] 连接MySQL: host=%s, port=%d, user=%s\n", server.Host, server.Port, server.Username)
		db, err := openMySQL(server)
		if err != nil {
			fmt.Printf("[fetchServerInfo] 连接失败: %v\n", err)
			return "", "", nil
		}
		defer db.Close()
		fmt.Printf("[fetchServerInfo] 连接成功\n")
		db.QueryRow("SELECT VERSION()").Scan(&version)
		var dataSize sql.NullFloat64
		db.QueryRow("SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) FROM information_schema.TABLES").Scan(&dataSize)
		if dataSize.Valid && dataSize.Float64 > 0 {
			if dataSize.Float64 > 1024 {
				disk = fmt.Sprintf("%.2fG", dataSize.Float64/1024)
			} else {
				disk = fmt.Sprintf("%.2fM", dataSize.Float64)
			}
		} else {
			var dbCount int
			db.QueryRow("SELECT COUNT(*) FROM information_schema.SCHEMATA").Scan(&dbCount)
			disk = fmt.Sprintf("%d个库", dbCount)
		}

		rows, err := db.Query("SHOW DATABASES")
		if err != nil {
			fmt.Printf("[fetchServerInfo] SHOW DATABASES失败: %v\n", err)
		} else {
			defer rows.Close()
			var dbName string
			for rows.Next() {
				if err := rows.Scan(&dbName); err == nil {
					lower := strings.ToLower(dbName)
					if lower != "information_schema" && lower != "mysql" && lower != "performance_schema" && lower != "sys" {
						databases = append(databases, dbName)
					}
				}
			}
			fmt.Printf("[fetchServerInfo] 获取到%d个数据库\n", len(databases))
		}
	}
	return version, disk, databases
}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     uint   `json:"id"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	server := findAnyServer(req.ID, req.Source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "not found"})
		return
	}

	version, disk, dbs := fetchServerInfo(server)
	if version == "" && disk == "" {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败，无法获取信息"})
		return
	}

	mutex.Lock()
	if req.Source == "remote" {
		for i := range remoteServers {
			if remoteServers[i].ID == req.ID {
				if version != "" {
					remoteServers[i].Version = version
				}
				if disk != "" {
					remoteServers[i].Disk = disk
				}
				break
			}
		}
	} else {
		for i := range databases {
			if databases[i].ID == req.ID {
				if version != "" {
					databases[i].Version = version
				}
				if disk != "" {
					databases[i].Disk = disk
				}
				if len(dbs) > 0 {
					databases[i].Databases = dbs
				}
				break
			}
		}
	}
	mutex.Unlock()
	saveData()

	writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"version": version, "disk": disk, "databases": dbs}})
}
