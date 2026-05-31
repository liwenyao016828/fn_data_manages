package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func testMySQLConnection(host string, port int, username, password string) string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=3s&readTimeout=3s&allowNativePasswords=true",
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

func testRedisConnection(host string, port int, username, password string) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	if username != "" && password != "" {
		resp, err := redisDo(conn, "AUTH", username, password)
		if err != nil {
			return false
		}
		if s, ok := resp.(string); ok && s != "OK" {
			return false
		}
	} else if password != "" {
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
		if dbType == "mysql" {
			version = testMySQLConnection(host, port, username, password)
		} else if dbType == "redis" {
			if testRedisConnection(host, port, username, password) {
				version = "connected"
			}
		}
		if version != "" {
			sysLogInfo("CONNECTION", fmt.Sprintf("连接测试成功 %s (%s:%d, %s)", req.Name, host, port, dbType))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success", "version": version})
		} else {
			sysLogWarn("CONNECTION", fmt.Sprintf("连接测试失败 %s (%s:%d, %s)", req.Name, host, port, dbType))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "连接失败"})
		}
		return
	}

	mutex.Lock()
	for _, db := range databases {
		if db.Name == req.Name {
			mutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "名称已存在"})
			return
		}
		if db.Host == req.Host && db.Port == req.Port && db.Username == req.Username {
			mutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的连接已存在"})
			return
		}
	}
	for _, rs := range remoteServers {
		if rs.Host == req.Host && rs.Port == req.Port && rs.Username == req.Username {
			mutex.Unlock()
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
		ver, dsk := fetchServerInfo(server)
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
					break
				}
			}
			mutex.Unlock()
		}
	}()

	saveData()

	sysLogInfo("CONNECTION", fmt.Sprintf("添加本地连接 %s (%s:%d)", req.Name, req.Host, req.Port))

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
	maskedResult := *result
	maskedResult.Password = maskPassword(result.Password)
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
	for i := range filtered {
		filtered[i].Password = maskPassword(filtered[i].Password)
	}
	mutex.Unlock()

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

func fetchServerInfo(server *RemoteServer) (string, string) {
	version, disk := "", ""

	if strings.ToLower(server.Type) == "redis" {
		conn, err := openRedis(server)
		if err != nil {
			return "", ""
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
	} else {
		db, err := openMySQL(server)
		if err != nil {
			return "", ""
		}
		defer db.Close()
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
	}

	return version, disk
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

	version, disk := fetchServerInfo(server)
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
				break
			}
		}
	}
	mutex.Unlock()
	saveData()

	writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]string{"version": version, "disk": disk}})
}