package main

import (
	"fmt"
	"net/http"
	"strings"
)

func syncPortsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != "POST" {
		writeJSON(w, map[string]interface{}{"code": 405, "msg": "method not allowed"})
		return
	}

	changes := syncAllPorts()
	writeJSON(w, map[string]interface{}{
		"code": 0,
		"msg":  fmt.Sprintf("端口同步完成，检测到 %d 个变更", len(changes)),
		"data": changes,
	})
}

func syncAllPorts() []map[string]interface{} {
	var changes []map[string]interface{}

	dockerPortMap := getDockerPortMapping()

	mutex.Lock()
	for i := range databases {
		if databases[i].ID >= 100000 {
			continue
		}

		oldPort := databases[i].Port
		newPort := oldPort

		if databases[i].Container != "" {
			if mapped, ok := dockerPortMap[databases[i].Container]; ok {
				if databases[i].Type == "mysql" && mapped.MysqlPort > 0 {
					newPort = mapped.MysqlPort
				} else if databases[i].Type == "redis" && mapped.RedisPort > 0 {
					newPort = mapped.RedisPort
				}
			}
		}

		if newPort == oldPort {
			if databases[i].Type == "mysql" {
				actualPort := detectMySQLPort(databases[i].Host, databases[i].Username, databases[i].Password, oldPort)
				if actualPort > 0 && actualPort != oldPort {
					newPort = actualPort
				}
			} else if databases[i].Type == "redis" {
				actualPort := detectRedisPort(databases[i].Host, databases[i].Password, oldPort)
				if actualPort > 0 && actualPort != oldPort {
					newPort = actualPort
				}
			}
		}

		if newPort != oldPort {
			databases[i].Port = newPort
			changes = append(changes, map[string]interface{}{
				"id":      databases[i].ID,
				"name":    databases[i].Name,
				"type":    databases[i].Type,
				"oldPort": oldPort,
				"newPort": newPort,
			})
		}
	}
	mutex.Unlock()

	if len(changes) > 0 {
		saveData()
	}

	return changes
}

type dockerPortInfo struct {
	MysqlPort int
	RedisPort int
}

func getDockerPortMapping() map[string]dockerPortInfo {
	result := make(map[string]dockerPortInfo)
	if !isDockerAvailable() {
		return result
	}

	var list []struct {
		Names []string `json:"Names"`
		Ports []struct {
			PrivatePort int `json:"PrivatePort"`
			PublicPort  int `json:"PublicPort"`
		} `json:"Ports"`
	}
	if _, err := dockerUnixSocketGetJSON("/containers/json?all=0&size=0", &list); err != nil {
		return result
	}

	for _, c := range list {
		info := dockerPortInfo{}
		for _, p := range c.Ports {
			switch p.PrivatePort {
			case 3306, 3307, 3308, 3309, 3310, 3316:
				info.MysqlPort = p.PublicPort
			case 6379, 6380, 6381:
				info.RedisPort = p.PublicPort
			}
		}
		if info.MysqlPort > 0 || info.RedisPort > 0 {
			name := ""
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			result[name] = info
		}
	}

	return result
}

func detectMySQLPort(host, username, password string, knownPort int) int {
	portsToTry := []int{3306, 3307, 3308, 3309, 3310, 3316, 23366}
	for _, port := range portsToTry {
		if port == knownPort {
			continue
		}
		ver, _ := testMySQLConnection(host, port, username, password)
		if ver != "" {
			return port
		}
	}
	return 0
}

func detectRedisPort(host, password string, knownPort int) int {
	portsToTry := []int{6379, 6380, 6381, 6382, 26379}
	for _, port := range portsToTry {
		if port == knownPort {
			continue
		}
		ok, _ := testRedisConnection(host, port, "", password)
		if ok {
			return port
		}
	}
	return 0
}
