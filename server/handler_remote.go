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

func maskPassword(s string) string {
	if s == "" {
		return ""
	}
	return "••••••••"
}

func remoteServerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case "GET":
		listRemoteServers(w, r)
	case "POST":
		createRemoteServer(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func remoteServerDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := r.URL.Path
	idStr := path[len("/api/remote-servers/"):]
	if idStr == "" || idStr == "test" {
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "invalid id"})
		return
	}

	switch r.Method {
	case "GET":
		getRemoteServer(w, r, uint(id))
	case "PUT":
		updateRemoteServer(w, r, uint(id))
	case "DELETE":
		deleteRemoteServer(w, r, uint(id))
	case "POST":
		if r.URL.Path[len("/api/remote-servers/"):] == fmt.Sprintf("%d/test", id) {
			testRemoteServer(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func remoteServerTestHandler(w http.ResponseWriter, r *http.Request) {
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

	testRemoteServer(w, r)
}

func createRemoteServer(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	mutex.Lock()
	for _, s := range remoteServers {
		if name, ok := req["name"].(string); ok && s.Name == name {
			mutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "名称已存在"})
			return
		}
	}
	host := getString(req, "host")
	port := getInt(req, "port")
	username := getString(req, "username")
	for _, s := range remoteServers {
		if s.Host == host && s.Port == port && s.Username == username {
			mutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的远程连接已存在"})
			return
		}
	}
	for _, db := range databases {
		if db.Host == host && db.Port == port && db.Username == username {
			mutex.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的本地连接已存在"})
			return
		}
	}

	newServer := RemoteServer{
		ID:          nextRemoteID,
		Name:        getString(req, "name"),
		Type:        getString(req, "type"),
		Version:     getString(req, "version"),
		Host:        getString(req, "host"),
		Port:        getInt(req, "port"),
		Username:    getString(req, "username"),
		Password:    getString(req, "password"),
		SSL:         getBool(req, "ssl"),
		Description: getString(req, "description"),
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	nextRemoteID++
	remoteServers = append(remoteServers, newServer)
	newID := newServer.ID
	mutex.Unlock()

	go func() {
		ver, dsk := fetchServerInfo(&newServer)
		if ver != "" || dsk != "" {
			mutex.Lock()
			for i := range remoteServers {
				if remoteServers[i].ID == newID {
					if ver != "" {
						remoteServers[i].Version = ver
					}
					if dsk != "" {
						remoteServers[i].Disk = dsk
					}
					break
				}
			}
			mutex.Unlock()
		}
	}()

	saveData()

	sysLogInfo("CONNECTION", fmt.Sprintf("添加远程连接 %s (%s:%d)", newServer.Name, newServer.Host, newServer.Port))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
}

func getRemoteServer(w http.ResponseWriter, r *http.Request, id uint) {
	mutex.Lock()
	var result *RemoteServer
	for i, s := range remoteServers {
		if s.ID == id {
			result = &remoteServers[i]
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

func listRemoteServers(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	result := make([]RemoteServer, len(remoteServers))
	copy(result, remoteServers)
	for i := range result {
		result[i].Password = maskPassword(result[i].Password)
	}
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": result})
}

func updateRemoteServer(w http.ResponseWriter, r *http.Request, id uint) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	mutex.Lock()
	found := false
	for i, s := range remoteServers {
		if s.ID == id {
			found = true
			if name, ok := req["name"].(string); ok && name != "" {
				for _, other := range remoteServers {
					if other.ID != s.ID && other.Name == name {
						mutex.Unlock()
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "名称已存在"})
						return
					}
				}
				remoteServers[i].Name = name
			}
			if typ, ok := req["type"].(string); ok && typ != "" {
				remoteServers[i].Type = typ
			}
			if ver, ok := req["version"].(string); ok && ver != "" {
				remoteServers[i].Version = ver
			}
			if host, ok := req["host"].(string); ok && host != "" {
				remoteServers[i].Host = host
			}
			if port, ok := req["port"].(float64); ok && port != 0 {
				remoteServers[i].Port = int(port)
			}
			if username, ok := req["username"].(string); ok && username != "" {
				remoteServers[i].Username = username
			}
			newHost := remoteServers[i].Host
			newPort := remoteServers[i].Port
			newUsername := remoteServers[i].Username
			for _, other := range remoteServers {
				if other.ID != s.ID && other.Host == newHost && other.Port == newPort && other.Username == newUsername {
					mutex.Unlock()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的远程连接已存在"})
					return
				}
			}
			for _, db := range databases {
				if db.Host == newHost && db.Port == newPort && db.Username == newUsername {
					mutex.Unlock()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]interface{}{"code": 400, "msg": "相同主机、端口和用户名的本地连接已存在"})
					return
				}
			}
			if password, ok := req["password"].(string); ok && password != "" && password != "••••••••" {
				remoteServers[i].Password = password
			}
			if ssl, ok := req["ssl"].(bool); ok {
				remoteServers[i].SSL = ssl
			}
			if desc, ok := req["description"].(string); ok {
				remoteServers[i].Description = desc
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

	sysLogInfo("CONNECTION", fmt.Sprintf("更新远程连接 ID=%d", id))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
}

func deleteRemoteServer(w http.ResponseWriter, r *http.Request, id uint) {
	mutex.Lock()
	found := false
	var delName, delHost string
	var delPort int
	for i, s := range remoteServers {
		if s.ID == id {
			delName = s.Name
			delHost = s.Host
			delPort = s.Port
			remoteServers = append(remoteServers[:i], remoteServers[i+1:]...)
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

	sysLogInfo("CONNECTION", fmt.Sprintf("删除远程连接 %s (%s:%d)", delName, delHost, delPort))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "msg": "success"})
}

func testRemoteServer(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	host := getString(req, "host")
	port := getInt(req, "port")
	username := getString(req, "username")
	password := getString(req, "password")
	dbType := getString(req, "type")
	if dbType == "" {
		dbType = "mysql"
	}

	if host == "" || port == 0 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "host and port are required"})
		return
	}

	if dbType == "redis" {
		addr := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "TCP 连接失败: " + err.Error()})
			return
		}
		var redisVer string
		var redisMem string
		if password != "" {
			authArgs := []string{"AUTH"}
			if username != "" {
				authArgs = append(authArgs, username, password)
			} else {
				authArgs = append(authArgs, password)
			}
			resp, authErr := redisDo(conn, authArgs...)
			if authErr != nil {
				conn.Close()
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "认证失败: " + authErr.Error()})
				return
			}
			if errStr, ok := resp.(string); ok && errStr != "OK" {
				conn.Close()
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "认证失败: " + errStr})
				return
			}
		}
		infoResp, infoErr := redisDo(conn, "INFO", "server")
		if infoErr == nil {
			if infoStr, ok := infoResp.(string); ok {
				for _, line := range strings.Split(infoStr, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "redis_version:") {
						redisVer = strings.TrimPrefix(line, "redis_version:")
					}
				}
			}
		}
		memResp, memErr := redisDo(conn, "INFO", "memory")
		if memErr == nil {
			if memStr, ok := memResp.(string); ok {
				for _, line := range strings.Split(memStr, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "used_memory_human:") {
						redisMem = strings.TrimPrefix(line, "used_memory_human:")
					}
				}
			}
		}
		conn.Close()
		result := map[string]interface{}{"code": 0, "msg": "Redis 连接成功"}
		if redisVer != "" {
			result["version"] = redisVer
		}
		if redisMem != "" {
			result["disk"] = redisMem
		}
		writeJSON(w, result)
		return
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=5s&allowNativePasswords=true",
		username, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()
	db.SetConnMaxLifetime(5 * time.Second)

	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "认证失败: " + err.Error()})
		return
	}

	var diskSize string
	var dataSize float64
	db.QueryRow("SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) FROM information_schema.TABLES").Scan(&dataSize)
	if dataSize > 0 {
		if dataSize > 1024 {
			diskSize = fmt.Sprintf("%.2fG", dataSize/1024)
		} else {
			diskSize = fmt.Sprintf("%.2fM", dataSize)
		}
	}

	result := map[string]interface{}{"code": 0, "msg": "连接成功", "version": version}
	if diskSize != "" {
		result["disk"] = diskSize
	}
	writeJSON(w, result)
}