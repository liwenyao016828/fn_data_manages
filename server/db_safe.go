package main

import (
	"fmt"
	"regexp"
	"strings"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

var allowedCharsets = map[string]bool{
	"utf8":    true,
	"utf8mb4": true,
	"latin1":  true,
	"gbk":     true,
	"gb2312":  true,
	"gb18030": true,
	"big5":    true,
	"ascii":   true,
	"binary":  true,
	"ucs2":    true,
	"utf16":   true,
	"utf32":   true,
}

var allowedMySQLPrivileges = map[string]bool{
	"ALL PRIVILEGES": true,
	"SELECT":         true,
	"INSERT":         true,
	"UPDATE":         true,
	"DELETE":         true,
	"CREATE":         true,
	"DROP":           true,
	"RELOAD":         true,
	"SHUTDOWN":       true,
	"PROCESS":        true,
	"FILE":           true,
	"GRANT OPTION":   true,
	"REFERENCES":     true,
	"INDEX":          true,
	"ALTER":          true,
	"SHOW DATABASES": true,
	"SUPER":          true,
	"CREATE TEMPORARY TABLES": true,
	"LOCK TABLES":    true,
	"EXECUTE":        true,
	"REPLICATION SLAVE":    true,
	"REPLICATION CLIENT":   true,
	"CREATE VIEW":   true,
	"SHOW VIEW":     true,
	"CREATE ROUTINE": true,
	"ALTER ROUTINE": true,
	"CREATE USER":   true,
	"EVENT":         true,
	"TRIGGER":       true,
	"CREATE TABLESPACE": true,
}

func validateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("标识符不能为空")
	}
	if len(name) > 64 {
		return fmt.Errorf("标识符长度不能超过64个字符")
	}
	if !identifierRegex.MatchString(name) {
		return fmt.Errorf("标识符包含非法字符，只允许字母、数字和下划线")
	}
	return nil
}

func validateDatabaseName(name string) error {
	if name == "" {
		return fmt.Errorf("数据库名不能为空")
	}
	if len(name) > 64 {
		return fmt.Errorf("数据库名长度不能超过64个字符")
	}
	return nil
}

func validateCharset(charset string) error {
	if charset == "" {
		return fmt.Errorf("字符集不能为空")
	}
	if !allowedCharsets[strings.ToLower(charset)] {
		return fmt.Errorf("不支持的字符集: %s", charset)
	}
	return nil
}

func validateMySQLUser(username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(username) > 32 {
		return fmt.Errorf("用户名长度不能超过32个字符")
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+$`, username)
	if !matched {
		return fmt.Errorf("用户名包含非法字符")
	}
	return nil
}

func validateMySQLHost(host string) error {
	if host == "" {
		return fmt.Errorf("主机名不能为空")
	}
	if host == "%" {
		return nil
	}
	if host == "localhost" {
		return nil
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.%]+$`, host)
	if !matched {
		return fmt.Errorf("主机名包含非法字符")
	}
	return nil
}

func validatePrivileges(privs string) error {
	if privs == "" {
		return fmt.Errorf("权限不能为空")
	}
	upper := strings.ToUpper(strings.TrimSpace(privs))
	if upper == "ALL" || upper == "ALL PRIVILEGES" {
		return nil
	}
	parts := strings.Split(upper, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !allowedMySQLPrivileges[p] {
			return fmt.Errorf("不支持的权限类型: %s", p)
		}
	}
	return nil
}

func validateVariableName(name string) error {
	if name == "" {
		return fmt.Errorf("变量名不能为空")
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\.]+$`, name)
	if !matched {
		return fmt.Errorf("变量名包含非法字符")
	}
	return nil
}

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quoteString(s string) string {
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\x00", "\\0")
	return "'" + escaped + "'"
}

func buildSelectFromDBTable(dbName, tableName string) (string, error) {
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	if err := validateIdentifier(tableName); err != nil {
		return "", fmt.Errorf("表名验证失败: %w", err)
	}
	return fmt.Sprintf("SELECT * FROM %s.%s", quoteIdentifier(dbName), quoteIdentifier(tableName)), nil
}

func buildCountFromDBTable(dbName, tableName string) (string, error) {
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	if err := validateIdentifier(tableName); err != nil {
		return "", fmt.Errorf("表名验证失败: %w", err)
	}
	return fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", quoteIdentifier(dbName), quoteIdentifier(tableName)), nil
}

func buildShowTables(dbName string) (string, error) {
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	return fmt.Sprintf("SHOW TABLES FROM %s", quoteIdentifier(dbName)), nil
}

func buildShowColumns(dbName, tableName string) (string, error) {
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	if err := validateIdentifier(tableName); err != nil {
		return "", fmt.Errorf("表名验证失败: %w", err)
	}
	return fmt.Sprintf("SHOW FULL COLUMNS FROM %s.%s", quoteIdentifier(dbName), quoteIdentifier(tableName)), nil
}

func buildCreateDatabase(name, charset string) (string, error) {
	if err := validateDatabaseName(name); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	if err := validateCharset(charset); err != nil {
		return "", fmt.Errorf("字符集验证失败: %w", err)
	}
	return fmt.Sprintf("CREATE DATABASE %s CHARACTER SET %s COLLATE %s_general_ci", quoteIdentifier(name), charset, charset), nil
}

func buildDropDatabase(name string) (string, error) {
	if err := validateDatabaseName(name); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	return fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdentifier(name)), nil
}

func buildCreateUser(username, host string) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("CREATE USER IF NOT EXISTS %s@%s IDENTIFIED BY ?", quoteString(username), quoteString(host)), nil
}

func buildGrantDBPrivileges(dbName, username, host string) (string, error) {
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s@%s", quoteIdentifier(dbName), quoteString(username), quoteString(host)), nil
}

func buildGrantGlobalPrivileges(privs, username, host string) (string, error) {
	if err := validatePrivileges(privs); err != nil {
		return "", fmt.Errorf("权限验证失败: %w", err)
	}
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	upper := strings.ToUpper(strings.TrimSpace(privs))
	if upper == "ALL" {
		upper = "ALL PRIVILEGES"
	}
	return fmt.Sprintf("GRANT %s ON *.* TO %s@%s", upper, quoteString(username), quoteString(host)), nil
}

func buildDropUser(username, host string) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("DROP USER %s@%s", quoteString(username), quoteString(host)), nil
}

func buildUseDatabase(dbName string) (string, error) {
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	return fmt.Sprintf("USE %s", quoteIdentifier(dbName)), nil
}

func buildSetGlobalVariable(name string, numericValue bool) (string, error) {
	if err := validateVariableName(name); err != nil {
		return "", fmt.Errorf("变量名验证失败: %w", err)
	}
	if numericValue {
		return fmt.Sprintf("SET GLOBAL %s = ?", quoteIdentifier(name)), nil
	}
	return fmt.Sprintf("SET GLOBAL %s = ?", quoteIdentifier(name)), nil
}

func buildShowVariablesLike(varName string) (string, error) {
	if err := validateVariableName(varName); err != nil {
		return "", fmt.Errorf("变量名验证失败: %w", err)
	}
	return "SHOW VARIABLES LIKE ?", nil
}
