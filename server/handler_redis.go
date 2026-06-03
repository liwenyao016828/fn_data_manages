package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func redisInfoHandler(w http.ResponseWriter, r *http.Request) {
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

	server := findRedisServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	conn, err := openRedis(server)
	if err != nil {
		if source != "remote" {
			containerName := findContainerName(uint(id), source)
			if containerName != "" {
				dockerPortMap := getDockerPortMapping()
				if info, ok := dockerPortMap[containerName]; ok && info.RedisPort > 0 && info.RedisPort != server.Port {
					mutex.Lock()
					for i := range databases {
						if databases[i].ID == uint(id) {
							databases[i].Port = info.RedisPort
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
							"newPort":     info.RedisPort,
						},
					})
					return
				}
			}
			newPort := detectRedisPort(server.Host, server.Password, server.Port)
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
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
		return
	}
	defer conn.Close()

	resp, err := redisDo(conn, "INFO")
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "获取信息失败: " + err.Error()})
		return
	}

	infoStr, ok := resp.(string)
	if !ok {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "解析信息失败"})
		return
	}

	info := make(map[string]string)
	for _, line := range strings.Split(infoStr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			info[parts[0]] = parts[1]
		}
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": info})
}

func redisKeysHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	pattern := r.URL.Query().Get("pattern")
	cursorStr := r.URL.Query().Get("cursor")
	countStr := r.URL.Query().Get("count")
	source := r.URL.Query().Get("source")

	if serverID == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}
	if pattern == "" {
		pattern = "*"
	}
	count := 50
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 && c <= 500 {
			count = c
		}
	}
	cursor := "0"
	if cursorStr != "" {
		cursor = cursorStr
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	server := findRedisServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	conn, err := openRedis(server)
	if err != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis命令执行连接失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
		return
	}
	defer conn.Close()

	resp, err := redisDo(conn, "SCAN", cursor, "MATCH", pattern, "COUNT", strconv.Itoa(count))
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "获取keys失败: " + err.Error()})
		return
	}

	arr, ok := resp.([]interface{})
	if !ok || len(arr) != 2 {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "解析keys结果失败"})
		return
	}

	nextCursor, _ := arr[0].(string)
	keyList, _ := arr[1].([]interface{})

	type KeyInfo struct {
		Key  string `json:"key"`
		Type string `json:"type"`
		TTL  int64  `json:"ttl"`
		Size int64  `json:"size"`
	}

	var keyInfos []KeyInfo
	for _, k := range keyList {
		key, _ := k.(string)
		if key == "" {
			continue
		}

		typeResp, _ := redisDo(conn, "TYPE", key)
		keyType, _ := typeResp.(string)

		ttlResp, _ := redisDo(conn, "TTL", key)
		ttl, _ := ttlResp.(int64)

		size := int64(0)
		switch keyType {
		case "string":
			if sResp, err := redisDo(conn, "STRLEN", key); err == nil {
				size, _ = sResp.(int64)
			}
		case "list":
			if sResp, err := redisDo(conn, "LLEN", key); err == nil {
				size, _ = sResp.(int64)
			}
		case "set":
			if sResp, err := redisDo(conn, "SCARD", key); err == nil {
				size, _ = sResp.(int64)
			}
		case "hash":
			if sResp, err := redisDo(conn, "HLEN", key); err == nil {
				size, _ = sResp.(int64)
			}
		case "zset":
			if sResp, err := redisDo(conn, "ZCARD", key); err == nil {
				size, _ = sResp.(int64)
			}
		}

		keyInfos = append(keyInfos, KeyInfo{
			Key:  key,
			Type: keyType,
			TTL:  ttl,
			Size: size,
		})
	}

	writeJSON(w, map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"keys":       keyInfos,
			"nextCursor": nextCursor,
			"total":      len(keyInfos),
		},
	})
}

func redisKeyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	key := r.URL.Query().Get("key")
	source := r.URL.Query().Get("source")

	if serverID == "" || key == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and key required"})
		return
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	server := findRedisServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	conn, err := openRedis(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
		return
	}
	defer conn.Close()

	typeResp, err := redisDo(conn, "TYPE", key)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "获取key类型失败: " + err.Error()})
		return
	}
	keyType, _ := typeResp.(string)

	ttlResp, _ := redisDo(conn, "TTL", key)
	ttl, _ := ttlResp.(int64)

	var value interface{}

	switch keyType {
	case "string":
		val, err := redisDo(conn, "GET", key)
		if err == nil {
			value = val
		} else {
			value = nil
		}
	case "list":
		val, err := redisDo(conn, "LRANGE", key, "0", "-1")
		if err == nil {
			value = val
		} else {
			value = []interface{}{}
		}
	case "set":
		val, err := redisDo(conn, "SMEMBERS", key)
		if err == nil {
			value = val
		} else {
			value = []interface{}{}
		}
	case "hash":
		val, err := redisDo(conn, "HGETALL", key)
		if err == nil {
			arr, ok := val.([]interface{})
			if ok {
				hash := make(map[string]string)
				for i := 0; i+1 < len(arr); i += 2 {
					k, _ := arr[i].(string)
					v, _ := arr[i+1].(string)
					hash[k] = v
				}
				value = hash
			} else {
				value = map[string]string{}
			}
		} else {
			value = map[string]string{}
		}
	case "zset":
		val, err := redisDo(conn, "ZRANGE", key, "0", "-1", "WITHSCORES")
		if err == nil {
			arr, ok := val.([]interface{})
			if ok {
				var items []map[string]interface{}
				for i := 0; i+1 < len(arr); i += 2 {
					member, _ := arr[i].(string)
					scoreStr, _ := arr[i+1].(string)
					score, _ := strconv.ParseFloat(scoreStr, 64)
					items = append(items, map[string]interface{}{
						"member": member,
						"score":  score,
					})
				}
				value = items
			} else {
				value = []map[string]interface{}{}
			}
		} else {
			value = []map[string]interface{}{}
		}
	default:
		value = fmt.Sprintf("(type: %s)", keyType)
	}

	writeJSON(w, map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"key":   key,
			"type":  keyType,
			"value": value,
			"ttl":   ttl,
		},
	})
}

var redisAllowedCommands = map[string]bool{
	"GET": true, "SET": true, "DEL": true, "EXISTS": true,
	"TYPE": true, "TTL": true, "PTTL": true, "EXPIRE": true,
	"PEXPIRE": true, "EXPIREAT": true, "PERSIST": true,
	"KEYS": true, "SCAN": true, "RANDOMKEY": true,
	"RENAME": true, "RENAMENX": true, "DUMP": true, "RESTORE": true,
	"APPEND": true, "STRLEN": true, "INCR": true, "DECR": true,
	"INCRBY": true, "DECRBY": true, "INCRBYFLOAT": true,
	"MGET": true, "MSET": true, "MSETNX": true,
	"LPUSH": true, "RPUSH": true, "LPOP": true, "RPOP": true,
	"LLEN": true, "LRANGE": true, "LINDEX": true, "LSET": true,
	"LREM": true, "LTRIM": true, "RPOPLPUSH": true,
	"SADD": true, "SREM": true, "SMEMBERS": true, "SISMEMBER": true,
	"SCARD": true, "SDIFF": true, "SINTER": true, "SUNION": true,
	"HSET": true, "HGET": true, "HDEL": true, "HGETALL": true,
	"HEXISTS": true, "HKEYS": true, "HVALS": true, "HLEN": true,
	"HMSET": true, "HMGET": true, "HINCRBY": true,
	"ZADD": true, "ZREM": true, "ZRANGE": true, "ZREVRANGE": true,
	"ZRANK": true, "ZREVRANK": true, "ZCARD": true, "ZSCORE": true,
	"ZRANGEBYSCORE": true, "ZREVRANGEBYSCORE": true, "ZREMANGEBYRANK": true,
	"ZREMANGEBYSCORE": true, "ZCOUNT": true,
	"SELECT": true, "DBSIZE": true, "INFO": true, "PING": true,
	"ECHO": true, "TIME": true, "CLIENT": true,
	"CONFIG": true, "OBJECT": true, "SLOWLOG": true, "MEMORY": true,
	"SRANDMEMBER": true, "SPOP": true, "SMOVE": true,
	"LINSERT": true, "BRPOP": true, "BLPOP": true,
	"PFADD": true, "PFCOUNT": true, "PFMERGE": true,
	"GEOADD": true, "GEODIST": true, "GEOHASH": true, "GEOPOS": true,
	"GEORADIUS": true, "GEORADIUSBYMEMBER": true,
	"XADD": true, "XLEN": true, "XRANGE": true, "XREVRANGE": true,
	"XREAD": true, "XGROUP": true, "XACK": true, "XPENDING": true,
	"XCLAIM": true, "XINFO": true, "XTRIM": true, "XDEL": true,
	"BITCOUNT": true, "BITOP": true, "GETBIT": true, "SETBIT": true,
	"BITPOS": true, "BITFIELD": true,
}

var redisDangerousCommands = map[string]bool{
	"FLUSHALL": true, "FLUSHDB": true, "SHUTDOWN": true,
	"DEBUG": true, "SYNC": true, "PSYNC": true,
	"SAVE": true, "BGSAVE": true, "BGREWRITEAOF": true,
	"SLAVEOF": true, "REPLICAOF": true, "CLUSTER": true,
	"MONITOR": true, "MIGRATE": true, "RESTORE-ASKING": true,
}

func redisExecuteHandler(w http.ResponseWriter, r *http.Request) {
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
		ServerID int      `json:"server_id"`
		Command  string   `json:"command"`
		Args     []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	if req.ServerID == 0 || req.Command == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id and command required"})
		return
	}

	cmd := strings.ToUpper(req.Command)
	if redisDangerousCommands[cmd] {
		writeJSON(w, map[string]interface{}{"code": 403, "msg": fmt.Sprintf("禁止执行危险命令: %s", cmd)})
		return
	}
	if !redisAllowedCommands[cmd] {
		writeJSON(w, map[string]interface{}{"code": 403, "msg": fmt.Sprintf("命令不在允许列表中: %s，如需执行请联系管理员", cmd)})
		return
	}

	source := r.URL.Query().Get("source")

	server := findRedisServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	conn, err := openRedis(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
		return
	}
	defer conn.Close()

	args := []string{req.Command}
	args = append(args, req.Args...)

	result, err := redisDo(conn, args...)
	if err != nil {
		sysLogWarn("REDIS", fmt.Sprintf("Redis命令执行失败: %s (连接: %s:%d)", cmd, server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "命令执行失败: " + err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"result": result}})
}

func redisConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	param := r.URL.Query().Get("param")
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

	server := findRedisServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	conn, err := openRedis(server)
	if err != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis配置连接失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
		return
	}
	defer conn.Close()

	if r.Method == "GET" {
		if param == "" {
			param = "*"
		}
		resp, err := redisDo(conn, "CONFIG", "GET", param)
		if err != nil {
			sysLogWarn("REDIS", fmt.Sprintf("Redis配置获取失败 (连接: %s:%d)", server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "获取配置失败: " + err.Error()})
			return
		}
		arr, ok := resp.([]interface{})
		if !ok {
			writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]string{}})
			return
		}
		result := make(map[string]string)
		sensitiveKeys := map[string]bool{
			"requirepass": true, "masterauth": true, "requirepass-file": true,
		}
		for i := 0; i+1 < len(arr); i += 2 {
			key, _ := arr[i].(string)
			val, _ := arr[i+1].(string)
			if sensitiveKeys[key] && val != "" {
				result[key] = "******"
			} else {
				result[key] = val
			}
		}
		writeJSON(w, map[string]interface{}{"code": 0, "data": result})
	} else if r.Method == "PUT" {
		var req struct {
			Param string `json:"param"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
			return
		}
		// 安全检查：限制可修改的配置参数
		dangerousParams := map[string]bool{
			"dir": true, "dbfilename": true, "appendfilename": true,
			"requirepass": true, "masterauth": true,
			"slave-read-only": true, "replica-read-only": true,
		}
		paramLower := strings.ToLower(req.Param)
		if dangerousParams[paramLower] {
			writeJSON(w, map[string]interface{}{"code": 403, "msg": "不允许修改该配置参数: " + req.Param})
			return
		}
		_, err := redisDo(conn, "CONFIG", "SET", req.Param, req.Value)
		if err != nil {
			sysLogError("REDIS", fmt.Sprintf("Redis配置修改失败: %s = %s (连接: %s:%d)", req.Param, req.Value, server.Host, server.Port))
			writeJSON(w, map[string]interface{}{"code": 1, "msg": "设置配置失败: " + err.Error()})
			return
		}
		sysLogInfo("REDIS", fmt.Sprintf("修改Redis配置: %s = %s (连接: %s:%d)", req.Param, req.Value, server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "配置已更新"})
	}
}

func redisLogsHandler(w http.ResponseWriter, r *http.Request) {
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

	server := findRedisServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	conn, err := openRedis(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
		return
	}
	defer conn.Close()

	resp, err := redisDo(conn, "CONFIG", "GET", "logfile")
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
			{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "Redis日志文件未配置或无法读取"},
		}})
		return
	}

	arr, ok := resp.([]interface{})
	if !ok || len(arr) < 2 {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
			{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "Redis日志文件未配置或无法读取"},
		}})
		return
	}

	logPath, _ := arr[1].(string)
	if logPath == "" {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
			{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "Redis日志输出到stdout，无法直接读取"},
		}})
		return
	}

	if source == "remote" {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
			{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "远程服务器Redis日志文件无法直接读取，日志路径: " + logPath},
		}})
		return
	}

	var data []byte
	fi, err := os.Stat(logPath)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
			{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "无法读取日志文件: " + err.Error()},
		}})
		return
	}
	const maxReadSize int64 = 512 * 1024
	f, err := os.Open(logPath)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
			{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "无法打开日志文件: " + err.Error()},
		}})
		return
	}
	defer f.Close()
	if fi.Size() > maxReadSize {
		if _, err := f.Seek(fi.Size()-maxReadSize, io.SeekStart); err != nil {
			writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
				{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "读取日志文件失败: " + err.Error()},
			}})
			return
		}
		data, err = io.ReadAll(f)
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			data = data[idx+1:]
		}
	} else {
		data, err = io.ReadAll(f)
	}
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []map[string]string{
			{"time": time.Now().Format("2006-01-02 15:04:05"), "level": "Note", "message": "读取日志文件失败: " + err.Error()},
		}})
		return
	}

	var logs []map[string]string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 尝试解析 Redis 日志时间戳
		logTime := time.Now().Format("2006-01-02 15:04:05")

		// 尝试解析标准时间格式 (Y-M-D H:M:S)
		if len(line) > 19 {
			if _, err := time.Parse("2006-01-02 15:04:05", line[:19]); err == nil {
				logTime = line[:19]
			} else if len(line) > 26 {
				if _, err := time.Parse("2006-01-02 15:04:05.000000", line[:26]); err == nil {
					logTime = line[:26]
				}
			} else if line[0] >= '0' && line[0] <= '9' && len(line) > 9 {
				// 尝试解析带年月日的日期
				if strings.Contains(line[:10], "-") && strings.Contains(line[:10], "-") {
					if _, err := time.Parse("2006-01-02", line[:10]); err == nil {
						logTime = line[:10]
					}
				}
			}
		}

		level := "Note"
		if strings.Contains(strings.ToUpper(line), "ERROR") || strings.Contains(strings.ToUpper(line), "CRITICAL") {
			level = "Error"
		} else if strings.Contains(strings.ToUpper(line), "WARNING") || strings.Contains(strings.ToUpper(line), "WARN") {
			level = "Warning"
		} else if strings.Contains(strings.ToUpper(line), "INFO") || strings.Contains(strings.ToUpper(line), "NOTICE") {
			level = "Info"
		}

		logs = append(logs, map[string]string{
			"time":    logTime,
			"level":   level,
			"message": line,
		})
	}

	maxLogLines := 500
	if len(logs) > maxLogLines {
		logs = logs[len(logs)-maxLogLines:]
	}

	if len(logs) == 0 {
		logs = append(logs, map[string]string{
			"time":    time.Now().Format("2006-01-02 15:04:05"),
			"level":   "Note",
			"message": "暂无日志数据",
		})
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": logs})
}

func systemLogsClearHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "方法不允许"})
		return
	}

	dataDir := getDataDir()
	logPath := filepath.Join(dataDir, "app.log")

	if err := os.Truncate(logPath, 0); err != nil {
		writeJSON(w, map[string]interface{}{"code": 500, "msg": "清空失败"})
		return
	}

	sysLogInfo("SYSTEM", "清空系统日志")
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "日志已清空"})
}

func systemLogWriteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "方法不允许"})
		return
	}

	var req struct {
		Level   string `json:"level"`
		Source  string `json:"source"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "无效请求"})
		return
	}

	if req.Message == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "消息不能为空"})
		return
	}

	if len(req.Message) > 500 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "消息长度不能超过500字符"})
		return
	}

	allowedSources := map[string]bool{
		"SYSTEM": true, "USER": true, "CONNECTION": true,
		"HEALTH": true, "BACKUP": true, "MYSQL": true, "REDIS": true,
	}
	source := req.Source
	if source == "" {
		source = "USER"
	}
	if !allowedSources[source] {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "无效的source值"})
		return
	}

	allowedLevels := map[string]bool{
		"info": true, "warning": true, "error": true,
	}
	level := req.Level
	if level == "" {
		level = "info"
	}
	if !allowedLevels[level] {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "无效的level值"})
		return
	}

	sysLog(level, source, req.Message)
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "已记录"})
}

func doRedisRdbCopy(server *RemoteServer, name string, bakDir string) (string, error) {
	if server.Host != "127.0.0.1" && server.Host != "localhost" {
		return "", fmt.Errorf("远程Redis实例不支持文件系统备份，仅支持本地Redis")
	}

	conn, err := openRedis(server)
	if err != nil {
		return "", fmt.Errorf("连接Redis失败: %w", err)
	}
	defer conn.Close()

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

	sourcePath := filepath.Join(rdbDir, rdbFile)
	input, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("读取RDB文件失败(%s): %w", sourcePath, err)
	}

	os.MkdirAll(bakDir, 0755)
	backupFileName := name + ".rdb"
	err = os.WriteFile(filepath.Join(bakDir, backupFileName), input, 0644)
	if err != nil {
		return "", fmt.Errorf("保存备份文件失败: %w", err)
	}
	return backupFileName, nil
}

func redisBackupHandler(w http.ResponseWriter, r *http.Request) {
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
		Database int    `json:"database"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	if req.ServerID == 0 {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}

	source := r.URL.Query().Get("source")

	server := findRedisServer(uint(req.ServerID), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	conn, connErr := openRedis(server)
	if connErr != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis备份连接失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + connErr.Error()})
		return
	}

	_, err := redisDo(conn, "SAVE")
	conn.Close()
	if err != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis SAVE失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "Redis SAVE失败: " + err.Error()})
		return
	}

	bakDir := getDataDir() + "/backups"
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("redis_backup_%s", time.Now().Format("20060102_150405"))
	}

	backupFileName, err := doRedisRdbCopy(server, name, bakDir)
	if err != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis RDB文件复制失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": err.Error()})
		return
	}

	if source == "" {
		source = "local"
	}

	newBackup := Backup{
		ID:          nextBackupID,
		Name:        name,
		Type:        "redis",
		ServerID:    uint(req.ServerID),
		Host:        server.Host,
		Port:        server.Port,
		Database:    fmt.Sprintf("db%d", req.Database),
		FileName:    backupFileName,
		FileSize:    0,
		Status:      "success",
		Description: "Redis RDB备份",
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
		BackupType:  "backup",
		BackupLevel: "redis",
		Source:      source,
	}
	normalizeBackup(&newBackup)

	mutex.Lock()
	nextBackupID++
	backups = append(backups, newBackup)
	mutex.Unlock()

	saveData()

	sysLogInfo("BACKUP", fmt.Sprintf("创建Redis备份: %s (连接: %s:%d)", newBackup.Name, server.Host, server.Port))
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "备份成功", "data": newBackup})
}

func redisRestoreHandler(w http.ResponseWriter, r *http.Request) {
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
		BackupID int `json:"backup_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
		return
	}

	mutex.Lock()
	var backup *Backup
	for i := range backups {
		if backups[i].ID == uint(req.BackupID) {
			backup = &backups[i]
			break
		}
	}
	mutex.Unlock()

	if backup == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "备份记录不存在"})
		return
	}

	source := r.URL.Query().Get("source")
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

	sysLogInfo("BACKUP", fmt.Sprintf("恢复Redis备份: %s (连接: %s:%d)", backup.Name, server.Host, server.Port))

	if server.Host != "127.0.0.1" && server.Host != "localhost" {
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "远程Redis实例不支持文件系统恢复，请使用 Redis CLI 手动恢复"})
		return
	}

	bakDir := getDataDir() + "/backups"
	backupPath := filepath.Join(bakDir, backup.FileName)

	input, err := os.ReadFile(backupPath)
	if err != nil {
		sysLogError("REDIS", "Redis恢复读取备份文件失败")
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "读取备份文件失败: " + err.Error()})
		return
	}

	conn, err := openRedis(server)
	if err != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis恢复连接失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + err.Error()})
		return
	}
	defer conn.Close()

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

	destPath := filepath.Join(rdbDir, rdbFile)
	err = os.WriteFile(destPath, input, 0644)
	if err != nil {
		sysLogError("REDIS", "Redis恢复写入RDB文件失败")
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "写入RDB文件失败: " + err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{"code": 0, "msg": "RDB文件已恢复，请重启Redis实例生效"})
}

func redisRestartHandler(w http.ResponseWriter, r *http.Request) {
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

	server := findRedisServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	sysLogInfo("REDIS", fmt.Sprintf("重启Redis (连接: %s:%d)", server.Host, server.Port))

	containerName := findContainerName(uint(id), source)

	if containerName != "" {
		if dockerPath, err := exec.LookPath("docker"); err == nil {
			out, err := exec.Command(dockerPath, "restart", containerName).CombinedOutput()
			if err != nil {
				sysLogError("REDIS", fmt.Sprintf("Docker重启Redis失败: %s (连接: %s:%d)", containerName, server.Host, server.Port))
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "Docker重启失败: " + strings.TrimSpace(string(out))})
				return
			}
			writeJSON(w, map[string]interface{}{"code": 0, "msg": "Redis重启成功(Docker容器 " + containerName + ")"})
			return
		}
	}

	conn, connErr := openRedis(server)
	if connErr != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis重启连接失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + connErr.Error()})
		return
	}

	dirResp, _ := redisDo(conn, "CONFIG", "GET", "dir")
	dbfilenameResp, _ := redisDo(conn, "CONFIG", "GET", "dbfilename")
	conn.Close()

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

	conn2, connErr2 := openRedis(server)
	if connErr2 != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis重启连接失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + connErr2.Error()})
		return
	}
	redisDo(conn2, "SHUTDOWN", "SAVE")
	conn2.Close()

	time.Sleep(2 * time.Second)

	var startErr string
	if _, err := exec.LookPath("systemctl"); err == nil {
		if err := exec.Command("systemctl", "start", "redis").Run(); err != nil {
			if err := exec.Command("systemctl", "start", "redis-server").Run(); err != nil {
				startErr = err.Error()
			}
		}
	} else if _, err := exec.LookPath("service"); err == nil {
		if err := exec.Command("service", "redis", "start").Run(); err != nil {
			if err := exec.Command("service", "redis-server", "start").Run(); err != nil {
				startErr = err.Error()
			}
		}
	} else if _, err := exec.LookPath("redis-server"); err == nil {
		rdbPath := filepath.Join(rdbDir, rdbFile)
		if _, statErr := os.Stat(rdbPath); statErr == nil {
			cmd := exec.Command("redis-server", "--dir", rdbDir, "--dbfilename", rdbFile)
			if err := cmd.Start(); err != nil {
				startErr = err.Error()
			}
		} else {
			cmd := exec.Command("redis-server")
			if err := cmd.Start(); err != nil {
				startErr = err.Error()
			}
		}
	} else {
		startErr = "找不到Redis启动命令，请手动启动"
	}

	if startErr != "" {
		sysLogError("REDIS", fmt.Sprintf("Redis自动启动失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "Redis已安全关闭，自动重启失败(" + startErr + ")，请手动启动"})
		return
	}
	writeJSON(w, map[string]interface{}{"code": 0, "msg": "Redis重启成功"})
}

func redisStopHandler(w http.ResponseWriter, r *http.Request) {
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

	server := findRedisServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	sysLogInfo("REDIS", fmt.Sprintf("停止Redis (连接: %s:%d)", server.Host, server.Port))

	containerName := findContainerName(uint(id), source)

	if containerName != "" {
		if dockerPath, err := exec.LookPath("docker"); err == nil {
			out, err := exec.Command(dockerPath, "stop", containerName).CombinedOutput()
			if err != nil {
				sysLogError("REDIS", fmt.Sprintf("Docker停止Redis失败: %s (连接: %s:%d)", containerName, server.Host, server.Port))
				writeJSON(w, map[string]interface{}{"code": 1, "msg": "Docker停止失败: " + strings.TrimSpace(string(out))})
				return
			}
			writeJSON(w, map[string]interface{}{"code": 0, "msg": "Redis已停止(Docker容器 " + containerName + ")"})
			return
		}
	}

	conn, connErr := openRedis(server)
	if connErr != nil {
		sysLogError("REDIS", fmt.Sprintf("Redis停止连接失败 (连接: %s:%d)", server.Host, server.Port))
		writeJSON(w, map[string]interface{}{"code": 1, "msg": "连接Redis失败: " + connErr.Error()})
		return
	}
	redisDo(conn, "SHUTDOWN", "SAVE")
	conn.Close()

	writeJSON(w, map[string]interface{}{"code": 0, "msg": "Redis已安全关闭（数据已保存）"})
}

func logConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	dataDir := getDataDir()
	configPath := filepath.Join(dataDir, "log_config.json")

	if r.Method == "GET" {
		data, err := os.ReadFile(configPath)
		defaultPath := filepath.Join(getDataDir(), "logs")
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"enabled": true, "path": defaultPath, "retentionDays": 30}})
			return
		}
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"enabled": true, "path": defaultPath, "retentionDays": 30}})
			return
		}
		if _, ok := config["path"]; !ok {
			config["path"] = defaultPath
		}
		if _, ok := config["retentionDays"]; !ok {
			config["retentionDays"] = 30
		}
		writeJSON(w, map[string]interface{}{"code": 0, "data": config})
		return
	}

	if r.Method == "PUT" {
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": err.Error()})
			return
		}
		allowedKeys := map[string]bool{"enabled": true, "path": true, "retentionDays": true}
		filtered := make(map[string]interface{})
		for k, v := range config {
			if allowedKeys[k] {
				filtered[k] = v
			}
		}
		data, err := json.Marshal(filtered)
		if err != nil {
			writeJSON(w, map[string]interface{}{"code": 500, "msg": "配置序列化失败"})
			return
		}
		if err := atomicWriteFile(configPath, data, 0644); err != nil {
			writeJSON(w, map[string]interface{}{"code": 500, "msg": "配置保存失败: " + err.Error()})
			return
		}
		sysLogInfo("SYSTEM", fmt.Sprintf("修改日志配置: enabled=%v, path=%s, retentionDays=%v", filtered["enabled"], filtered["path"], filtered["retentionDays"]))
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "配置已保存"})
		return
	}

	writeJSON(w, map[string]interface{}{"code": 405, "msg": "方法不允许"})
}

func systemLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != "GET" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "方法不允许"})
		return
	}

	_ = r.URL.Query().Get("server_id")
	_ = r.URL.Query().Get("source")

	dataDir := getDataDir()
	logPath := filepath.Join(dataDir, "app.log")

	f, err := os.Open(logPath)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []interface{}{}})
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": []interface{}{}})
		return
	}

	const maxReadSize int64 = 512 * 1024
	var scanner *bufio.Scanner
	if fi.Size() > maxReadSize {
		offset := fi.Size() - maxReadSize
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			writeJSON(w, map[string]interface{}{"code": 0, "data": []interface{}{}})
			return
		}
		scanner = bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		if !scanner.Scan() {
			writeJSON(w, map[string]interface{}{"code": 0, "data": []interface{}{}})
			return
		}
	} else {
		scanner = bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	}

	queryLogPatterns := []string{"获取MySQL日志", "获取Redis日志", "获取系统日志"}

	var logs []map[string]string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		isQueryLog := false
		for _, pattern := range queryLogPatterns {
			if strings.Contains(line, pattern) {
				isQueryLog = true
				break
			}
		}
		if isQueryLog {
			continue
		}
		entry := map[string]string{
			"time":    "",
			"level":   "info",
			"source":  "SYSTEM",
			"message": line,
		}
		if len(line) > 19 && line[4] == '-' && line[7] == '-' && line[10] == 'T' {
			tsEnd := 19
			if len(line) > 25 && (line[19] == '+' || line[19] == '-') && line[22] == ':' {
				tsEnd = 25
			}
			entry["time"] = line[:tsEnd]
			rest := strings.TrimSpace(line[tsEnd:])
			parts := strings.SplitN(rest, " ", 3)
			if len(parts) >= 1 {
				lvl := strings.ToUpper(strings.Trim(parts[0], " []"))
				switch lvl {
				case "ERROR", "ERR":
					entry["level"] = "error"
				case "WARNING", "WARN":
					entry["level"] = "warning"
				case "DEBUG":
					entry["level"] = "debug"
				default:
					entry["level"] = "info"
				}
			}
			if len(parts) >= 2 {
				src := strings.Trim(parts[1], " []")
				if src != "" {
					entry["source"] = src
				}
			}
			if len(parts) >= 3 {
				entry["message"] = parts[2]
			}
		}
		logs = append(logs, entry)
	}

	if logs == nil {
		logs = []map[string]string{}
	}

	maxLogs := 500
	if len(logs) > maxLogs {
		logs = logs[len(logs)-maxLogs:]
	}

	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	writeJSON(w, map[string]interface{}{"code": 0, "data": logs})
}
