package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func findRemoteServer(id uint) *RemoteServer {
	mutex.Lock()
	defer mutex.Unlock()
	for i := range remoteServers {
		if remoteServers[i].ID == id {
			s := remoteServers[i]
			return &s
		}
	}
	return nil
}

func findAnyServer(id uint, source string) *RemoteServer {
	mutex.Lock()
	defer mutex.Unlock()

	if source == "remote" {
		for i := range remoteServers {
			if remoteServers[i].ID == id {
				s := remoteServers[i]
				return &s
			}
		}
		return nil
	}

	for i := range databases {
		if databases[i].ID == id {
			return &RemoteServer{
				ID:       databases[i].ID,
				Name:     databases[i].Name,
				Host:     databases[i].Host,
				Port:     databases[i].Port,
				Username: databases[i].Username,
				Password: databases[i].Password,
				Type:     databases[i].Type,
				Version:  databases[i].Version,
			}
		}
	}
	return nil
}

func openMySQL(s *RemoteServer) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=10s&readTimeout=30s&writeTimeout=30s&allowNativePasswords=true",
		s.Username, s.Password, s.Host, s.Port)
	return sql.Open("mysql", dsn)
}

func openMySQLWithTimeout(s *RemoteServer, timeout time.Duration) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=%s&readTimeout=%s&writeTimeout=%s&allowNativePasswords=true",
		s.Username, s.Password, s.Host, s.Port,
		timeout.String(), timeout.String(), timeout.String())
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	return db, nil
}

func openRedis(s *RemoteServer) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	if s.Username != "" && s.Password != "" {
		resp, err := redisDo(conn, "AUTH", s.Username, s.Password)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("认证失败: %w", err)
		}
		if errStr, ok := resp.(string); ok && errStr != "OK" {
			conn.Close()
			return nil, fmt.Errorf("认证失败: %s", errStr)
		}
	} else if s.Password != "" {
		resp, err := redisDo(conn, "AUTH", s.Password)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("认证失败: %w", err)
		}
		if errStr, ok := resp.(string); ok && errStr != "OK" {
			conn.Close()
			return nil, fmt.Errorf("认证失败: %s", errStr)
		}
	}
	return conn, nil
}

func redisWriteCmd(conn net.Conn, args ...string) error {
	cmd := fmt.Sprintf("*%d\r\n", len(args))
	for _, arg := range args {
		cmd += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
	}
	_, err := conn.Write([]byte(cmd))
	return err
}

func redisReadResp(conn net.Conn) (interface{}, error) {
	reader := bufio.NewReader(conn)
	return redisParseResp(reader)
}

func redisParseResp(reader *bufio.Reader) (interface{}, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	switch line[0] {
	case '+':
		return line[1:], nil
	case '-':
		return nil, fmt.Errorf(line[1:])
	case ':':
		return strconv.ParseInt(line[1:], 10, 64)
	case '$':
		count, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if count == -1 {
			return nil, nil
		}
		data := make([]byte, count)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return nil, err
		}
		reader.Discard(2)
		return string(data), nil
	case '*':
		count, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if count == -1 {
			return nil, nil
		}
		arr := make([]interface{}, count)
		for i := 0; i < count; i++ {
			arr[i], err = redisParseResp(reader)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	default:
		return line, nil
	}
}

func redisDo(conn net.Conn, args ...string) (interface{}, error) {
	if err := redisWriteCmd(conn, args...); err != nil {
		return nil, err
	}
	return redisReadResp(conn)
}

func findRedisServer(id uint, source string) *RemoteServer {
	return findAnyServer(id, source)
}

func findContainerName(id uint, source string) string {
	mutex.Lock()
	defer mutex.Unlock()

	if source == "remote" {
		for i := range remoteServers {
			if remoteServers[i].ID == id {
				return remoteServers[i].Container
			}
		}
		return ""
	}

	for i := range databases {
		if databases[i].ID == id {
			return databases[i].Container
		}
	}
	return ""
}