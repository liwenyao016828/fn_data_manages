package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
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
		return nil, errors.New(line[1:])
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

func openPostgreSQL(s *RemoteServer) (*sql.DB, error) {
	return openPostgreSQLWithDB(s, "")
}

func openPostgreSQLWithDB(s *RemoteServer, dbName string) (*sql.DB, error) {
	dbname := dbName
	if dbname == "" {
		dbname = "postgres"
	}
	sslmode := "disable"
	if s.SSL {
		sslmode = "require"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=10",
		s.Username, s.Password, s.Host, s.Port, dbname, sslmode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}
	return db, nil
}

func openSQLite(s *RemoteServer) (*sql.DB, error) {
	filePath := s.Host // SQLite uses Host field to store file path
	if filePath == "" {
		return nil, fmt.Errorf("SQLite文件路径不能为空")
	}
	// #8 SQLite 路径校验：拒绝包含危险字符的 Host 字段
	// DSN 中这些字符可能被解析为查询参数、URL 编码片段或路径分隔符
	if err := validateSQLitePath(filePath); err != nil {
		return nil, fmt.Errorf("SQLite路径校验失败: %w", err)
	}
	// 使用 url.Values 拼查询参数，避免 ? & 拆分错误
	dsn := filePath + "?mode=rwc&_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开SQLite文件失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接SQLite失败: %w", err)
	}
	// 优化SQLite性能设置
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA foreign_keys=ON")
	return db, nil
}

// validateSQLitePath 检查 SQLite 文件路径是否合法
// - 不能包含 URL/DSN 元字符: ? # \ : * ? " < > |
// - 必须是绝对路径
// - 不允许 ".." 路径组件
func validateSQLitePath(p string) error {
	if p == "" {
		return fmt.Errorf("路径不能为空")
	}
	// 元字符检查（在拼接到 DSN 前必须排除）
	forbidden := []string{"?", "#", "\\", "*", "\"", "<", ">", "|"}
	for _, ch := range forbidden {
		if strings.Contains(p, ch) {
			return fmt.Errorf("路径包含非法字符: %q", ch)
		}
	}
	// 必须为绝对路径（Unix: 以 / 开头；Windows: 含盘符）
	isAbs := strings.HasPrefix(p, "/")
	if runtime.GOOS == "windows" && len(p) >= 3 && p[1] == ':' {
		isAbs = true
	}
	if !isAbs {
		return fmt.Errorf("SQLite 路径必须为绝对路径: %s", p)
	}
	// 拒绝 ".."
	if strings.Contains(p, "..") {
		return fmt.Errorf("SQLite 路径不允许包含 ..: %s", p)
	}
	return nil
}
