package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lib/pq"
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

// buildShowGrantsFor SHOW GRANTS FOR 'user'@'host'
func buildShowGrantsFor(username, host string) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("SHOW GRANTS FOR %s@%s", quoteString(username), quoteString(host)), nil
}

// buildCreateUserWithPassword CREATE USER 'user'@'host' IDENTIFIED BY ?
// 使用占位符 ? 传密码，避免密码内容进入 SQL 文本
func buildCreateUserWithPassword(username, host string) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("CREATE USER %s@%s IDENTIFIED BY ?", quoteString(username), quoteString(host)), nil
}

// buildCreateUserIfNotExists CREATE USER IF NOT EXISTS 'user'@'host' IDENTIFIED BY ?
func buildCreateUserIfNotExists(username, host string) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("CREATE USER IF NOT EXISTS %s@%s IDENTIFIED BY ?",
		quoteString(username), quoteString(host)), nil
}

// buildAlterUserPassword ALTER USER 'user'@'host' IDENTIFIED BY ?
func buildAlterUserPassword(username, host string) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("ALTER USER %s@%s IDENTIFIED BY ?",
		quoteString(username), quoteString(host)), nil
}

// buildAlterUserAccount ALTER USER 'user'@'host' ACCOUNT LOCK|UNLOCK
func buildAlterUserAccount(username, host string, locked bool) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	action := "UNLOCK"
	if locked {
		action = "LOCK"
	}
	return fmt.Sprintf("ALTER USER %s@%s ACCOUNT %s",
		quoteString(username), quoteString(host), action), nil
}

// buildRevokeAllDBPrivileges REVOKE ALL PRIVILEGES ON `db`.* FROM 'user'@'host'
func buildRevokeAllDBPrivileges(dbName, username, host string) (string, error) {
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
	}
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("REVOKE ALL PRIVILEGES ON %s.* FROM %s@%s",
		quoteIdentifier(dbName), quoteString(username), quoteString(host)), nil
}

// buildRevokeAllGlobalPrivileges REVOKE ALL PRIVILEGES ON *.* FROM 'user'@'host'
func buildRevokeAllGlobalPrivileges(username, host string) (string, error) {
	if err := validateMySQLUser(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	if err := validateMySQLHost(host); err != nil {
		return "", fmt.Errorf("主机名验证失败: %w", err)
	}
	return fmt.Sprintf("REVOKE ALL PRIVILEGES ON *.* FROM %s@%s",
		quoteString(username), quoteString(host)), nil
}

// buildGrantDBPrivilegesCustom GRANT <privs> ON `db`.* TO 'user'@'host'
func buildGrantDBPrivilegesCustom(privs, dbName, username, host string) (string, error) {
	if err := validatePrivileges(privs); err != nil {
		return "", fmt.Errorf("权限验证失败: %w", err)
	}
	if err := validateIdentifier(dbName); err != nil {
		return "", fmt.Errorf("数据库名验证失败: %w", err)
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
	return fmt.Sprintf("GRANT %s ON %s.* TO %s@%s",
		upper, quoteIdentifier(dbName), quoteString(username), quoteString(host)), nil
}

// buildPostgresCreateUser CREATE USER "name" WITH PASSWORD ?
// 使用 ? 占位符传密码，密码不会出现在 SQL 文本中
func buildPostgresCreateUser(username string) (string, error) {
	if err := validateIdentifier(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	return fmt.Sprintf("CREATE USER %s WITH PASSWORD ?", pq.QuoteIdentifier(username)), nil
}

// buildPostgresAlterUser ALTER USER "name" WITH PASSWORD ?
func buildPostgresAlterUser(username string) (string, error) {
	if err := validateIdentifier(username); err != nil {
		return "", fmt.Errorf("用户名验证失败: %w", err)
	}
	return fmt.Sprintf("ALTER USER %s WITH PASSWORD ?", pq.QuoteIdentifier(username)), nil
}
// 仅允许从 GRANTS 中提取的标准权限子句，避免拼接非授权内容
func validateGrantSQL(rawGrant, newUser, newHost string) (string, bool) {
	rawGrant = strings.TrimSpace(rawGrant)
	if rawGrant == "" {
		return "", false
	}
	upper := strings.ToUpper(rawGrant)
	if !strings.HasPrefix(upper, "GRANT ") {
		return "", false
	}
	// 拒绝 PROXY 等特殊授权
	if strings.Contains(upper, "GRANT PROXY ON") {
		return "", false
	}
	// 提取 PRIVILEGES 部分（GRANT 后到 ON 之前）
	// 形式：GRANT <privs> ON <obj> TO ...
	// 使用正则提取 privs 和 obj
	re := regexp.MustCompile(`(?i)^GRANT\s+(.+?)\s+ON\s+(.+?)\s+TO\s+`)
	matches := re.FindStringSubmatch(rawGrant)
	if len(matches) < 3 {
		return "", false
	}
	privs := strings.TrimSpace(matches[1])
	onPart := strings.TrimSpace(matches[2])
	// 验证 privs 是否完全由允许的权限组合（白名单 token）
	if !isValidPrivilegesList(privs) {
		return "", false
	}
	// 校验 user/host
	if err := validateMySQLUser(newUser); err != nil {
		return "", false
	}
	if err := validateMySQLHost(newHost); err != nil {
		return "", false
	}
	return fmt.Sprintf("GRANT %s ON %s TO %s@%s",
		privs, onPart, quoteString(newUser), quoteString(newHost)), true
}

// isValidPrivilegesList 权限列表是否由白名单 token 组成
func isValidPrivilegesList(privs string) bool {
	upper := strings.ToUpper(strings.TrimSpace(privs))
	if upper == "ALL PRIVILEGES" || upper == "ALL" {
		return true
	}
	// 处理可选的 GRANT OPTION 后缀
	grantOpt := false
	if strings.HasSuffix(upper, " WITH GRANT OPTION") {
		grantOpt = true
		upper = strings.TrimSuffix(upper, " WITH GRANT OPTION")
	}
	parts := strings.Split(upper, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !allowedMySQLPrivileges[p] {
			return false
		}
	}
	_ = grantOpt
	return true
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

// allowedExecuteSQLKeywords SQL 控制台允许的首关键字白名单
// 配合 parseFirstKeyword 使用，跳过注释和字符串字面量
var allowedExecuteSQLKeywords = map[string]bool{
	"SELECT":       true,
	"SHOW":         true,
	"DESCRIBE":     true,
	"DESC":         true,
	"EXPLAIN":      true,
	"INSERT":       true,
	"UPDATE":       true,
	"DELETE":       true,
	"REPLACE":      true,
	"CREATE":       true, // 受限于具体对象：TABLE/INDEX/VIEW/TRIGGER/FUNCTION/PROCEDURE
	"ALTER":        true, // 受限于具体对象：TABLE/INDEX/VIEW
	"DROP":         true, // 受限于具体对象：TABLE/INDEX/VIEW/TRIGGER/FUNCTION/PROCEDURE
	"RENAME":       true, // RENAME TABLE 允许，RENAME USER 拒绝
	"SET":          true, // 仅允许非 GLOBAL 持久变量
	"ANALYZE":      true,
	"OPTIMIZE":     true,
	"REPAIR":       true,
	"CHECK":        true,
	"TRUNCATE":     true,
	"START":        true,
	"COMMIT":       true,
	"ROLLBACK":     true,
	"USE":          true,
	"WITH":         true,
}

// allowedExecuteSqlRestrictedKeywords 需要进一步校验后缀的允许关键字
// map[首关键字] -> 允许的"后缀白名单"集合
var allowedExecuteSqlRestrictedKeywords = map[string]map[string]bool{
	"CREATE": {
		"TABLE": true, "INDEX": true, "VIEW": true, "TRIGGER": true,
		"FUNCTION": true, "PROCEDURE": true, "TEMPORARY": true,
	},
	"ALTER": {
		"TABLE": true, "INDEX": true, "VIEW": true,
	},
	"DROP": {
		"TABLE": true, "INDEX": true, "VIEW": true, "TRIGGER": true,
		"FUNCTION": true, "PROCEDURE": true,
	},
	"RENAME": {
		"TABLE": true,
	},
}

// stripSQLCommentsAndStrings 去掉 SQL 中的行内/块注释与字符串字面量，避免误判
// 注意：仅用于关键字提取，不是 SQL 解析器，足够做白名单判断
func stripSQLCommentsAndStrings(sql string) string {
	var b strings.Builder
	i := 0
	for i < len(sql) {
		// 行注释
		if i+1 < len(sql) && sql[i] == '-' && sql[i+1] == '-' {
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			continue
		}
		// 块注释
		if i+1 < len(sql) && sql[i] == '/' && sql[i+1] == '*' {
			i += 2
			for i+1 < len(sql) && !(sql[i] == '*' && sql[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		// 单引号字符串
		if sql[i] == '\'' {
			i++
			for i < len(sql) {
				if sql[i] == '\\' && i+1 < len(sql) {
					i += 2
					continue
				}
				if sql[i] == '\'' {
					i++
					break
				}
				i++
			}
			b.WriteByte(' ')
			continue
		}
		// 双引号字符串
		if sql[i] == '"' {
			i++
			for i < len(sql) {
				if sql[i] == '\\' && i+1 < len(sql) {
					i += 2
					continue
				}
				if sql[i] == '"' {
					i++
					break
				}
				i++
			}
			b.WriteByte(' ')
			continue
		}
		// 反引号标识符
		if sql[i] == '`' {
			i++
			for i < len(sql) {
				if sql[i] == '`' && i+1 < len(sql) && sql[i+1] == '`' {
					i += 2
					continue
				}
				if sql[i] == '`' {
					i++
					break
				}
				i++
			}
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(sql[i])
		i++
	}
	return b.String()
}

// firstTwoKeywords 提取 SQL 中前两个非空关键字
// 用于受限关键字的"后缀白名单"判断
func firstTwoKeywords(sql string) (first, second string) {
	stripped := stripSQLCommentsAndStrings(sql)
	fields := strings.Fields(stripped)
	if len(fields) == 0 {
		return "", ""
	}
	first = strings.ToUpper(fields[0])
	if len(fields) >= 2 {
		second = strings.ToUpper(fields[1])
	}
	return first, second
}

// validateExecuteSQL 检查 SQL 首关键字是否在白名单内
// 黑名单模式容易被绕过（大小写、注释、字符串），白名单更安全
func validateExecuteSQL(sql string) error {
	first, second := firstTwoKeywords(sql)
	if first == "" {
		return fmt.Errorf("SQL 为空")
	}
	if !allowedExecuteSQLKeywords[first] {
		return fmt.Errorf("禁止执行 %s 语句，仅允许 DML/受限 DDL/查询", first)
	}
	if restricted, ok := allowedExecuteSqlRestrictedKeywords[first]; ok {
		// 受限关键字必须以允许的对象类型开头
		if !restricted[second] {
			return fmt.Errorf("不允许的 %s 对象类型: %s", first, second)
		}
	}
	if first == "SET" {
		// 拒绝 SET GLOBAL/PERSIST/SESSION 改全局变量
		upper := strings.ToUpper(sql)
		// 仅检查语句级关键字（不一定在第二位）
		for _, kw := range []string{"GLOBAL", "PERSIST", "PERSIST_ONLY"} {
			// 必须以 SET <kw> ... 形式出现，且不在字符串/注释中
			// 使用单词边界匹配（粗糙但够用）
			if matched, _ := regexp.MatchString(`(?i)\bSET\s+`+kw+`\b`, upper); matched {
				return fmt.Errorf("不允许使用 SET %s 修改全局变量", kw)
			}
		}
	}
	return nil
}
