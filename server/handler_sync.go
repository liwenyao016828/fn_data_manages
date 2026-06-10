package main

import (
	"fmt"
	"net/http"
	"os/exec"
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

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return result
	}

	cmd := exec.Command(dockerPath, "ps", "--format", "{{.Names}}\t{{.Ports}}")
	out, err := cmd.Output()
	if err != nil {
		return result
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		ports := parts[1]

		info := dockerPortInfo{}
		mysqlPort := extractHostPort(ports, "mysql")
		redisPort := extractHostPort(ports, "redis")
		if mysqlPort > 0 {
			info.MysqlPort = mysqlPort
		}
		if redisPort > 0 {
			info.RedisPort = redisPort
		}
		if info.MysqlPort > 0 || info.RedisPort > 0 {
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
		if testMySQLConnection(host, port, username, password) != "" {
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
		if testRedisConnection(host, port, "", password) {
			return port
		}
	}
	return 0
}
