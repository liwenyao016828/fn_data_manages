package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// fileLocks 保证对同一文件的并发写入被串行化，避免产生孤儿 .tmp 文件
var fileLocks sync.Map

func lockForFile(path string) *sync.Mutex {
	v, _ := fileLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func saveData() {
	// #7 拆分锁：分别取四个域的快照，避开单一全局锁
	dbMu.RLock()
	dbCopy := make([]Database, len(databases))
	copy(dbCopy, databases)
	// 兼容旧数据：解密已加密的密码（内网明文存储）
	for i := range dbCopy {
		dbCopy[i].Password = decryptPassword(dbCopy[i].Password)
	}
	dbMu.RUnlock()

	remoteMu.RLock()
	rsCopy := make([]RemoteServer, len(remoteServers))
	copy(rsCopy, remoteServers)
	for i := range rsCopy {
		rsCopy[i].Password = decryptPassword(rsCopy[i].Password)
	}
	remoteMu.RUnlock()

	backupMu.RLock()
	bkCopy := make([]Backup, len(backups))
	copy(bkCopy, backups)
	schedCopy := make([]ScheduledBackup, len(scheduledBackups))
	copy(schedCopy, scheduledBackups)
	backupMu.RUnlock()

	dataDir := os.Getenv("TRIM_PKGVAR")
	if dataDir == "" {
		dataDir = "./data"
	}
	os.MkdirAll(dataDir, 0755)

	data, _ := json.Marshal(dbCopy)
	atomicWriteFile(filepath.Join(dataDir, "databases.json"), data, 0644)

	remoteData, _ := json.Marshal(rsCopy)
	atomicWriteFile(filepath.Join(dataDir, "remote_servers.json"), remoteData, 0644)

	backupData, _ := json.Marshal(bkCopy)
	atomicWriteFile(filepath.Join(dataDir, "backups.json"), backupData, 0644)

	schedData, _ := json.Marshal(schedCopy)
	atomicWriteFile(filepath.Join(dataDir, "scheduled_backups.json"), schedData, 0644)
}

// atomicWriteFile 原子写入文件，流程：
// 1. 写入 path.<pid>.<nanos>.tmp 临时文件
// 2. 关闭后 fsync 强制刷盘
// 3. 尝试 rename 覆盖目标（POSIX 原子，Windows 上目标存在时需先 remove）
// 4. 失败兜底：remove 目标后再 rename
// 5. 用 per-file 互斥锁防止并发写导致 .tmp 冲突
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	lock := lockForFile(path)
	lock.Lock()
	defer lock.Unlock()

	// 临时文件名带 pid+ns 避免与历史孤儿冲突
	tmpPath := fmt.Sprintf("%s.%d.%d.tmp", path, os.Getpid(), time.Now().UnixNano())

	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close tmp: %w", err)
	}

	// 优先尝试 rename（POSIX 原子）
	if err := os.Rename(tmpPath, path); err == nil {
		return nil
	}

	// 兜底：删除目标后 rename（Windows 等）
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		os.Remove(tmpPath)
		return fmt.Errorf("remove old: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename after remove: %w", err)
	}
	return nil
}

func loadData() {
	dataDir := os.Getenv("TRIM_PKGVAR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// 加载函数：单文件加载并保留原文件作为备份
	loadJSONFile := func(filename string, dst interface{}) bool {
		path := filepath.Join(dataDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			return false // 文件不存在时静默忽略
		}
		if err := json.Unmarshal(data, dst); err != nil {
			// 解析失败：保留原文件为 .bak，启动时仅警告
			bakPath := path + ".bak"
			if _, cpErr := os.Stat(bakPath); cpErr == nil {
				// 已存在备份，覆盖它
				os.WriteFile(bakPath, data, 0644)
			} else {
				os.Rename(path, bakPath)
			}
			sysLogError("STORE", fmt.Sprintf("%s 解析失败: %v (已备份为 %s)", filename, err, bakPath))
			return false
		}
		return true
	}

	if loadJSONFile("databases.json", &databases) {
		// 兼容旧数据：解密已加密的密码
		for i := range databases {
			databases[i].Password = decryptPassword(databases[i].Password)
		}
		if len(databases) > 0 {
			nextID = databases[len(databases)-1].ID + 1
		}
		saveData() // 回写为明文
	}

	if loadJSONFile("remote_servers.json", &remoteServers) {
		for i := range remoteServers {
			remoteServers[i].Password = decryptPassword(remoteServers[i].Password)
		}
		if len(remoteServers) > 0 {
			nextRemoteID = remoteServers[len(remoteServers)-1].ID + 1
		}
		saveData() // 回写为明文
	}

	if loadJSONFile("backups.json", &backups) {
		if len(backups) > 0 {
			nextBackupID = backups[len(backups)-1].ID + 1
		}
	}

	if loadJSONFile("scheduled_backups.json", &scheduledBackups) {
		if len(scheduledBackups) > 0 {
			nextSchedID = scheduledBackups[len(scheduledBackups)-1].ID + 1
		}
	}

	needsMigrate := false
	for i := range backups {
		oldLevel := backups[i].BackupLevel
		oldType := backups[i].Type
		normalizeBackup(&backups[i])
		if backups[i].BackupLevel != oldLevel || backups[i].Type != oldType {
			needsMigrate = true
		}
	}
	if needsMigrate {
		saveData()
	}

	metricsHistory = make(map[uint][]MetricsSnapshot)
	loadJSONFile("metrics_history.json", &metricsHistory)

	migrateMetricsHistory()
}

func migrateMetricsHistory() {
	if metricsHistory == nil {
		return
	}
	localIDs := make(map[uint]bool)
	for _, db := range databases {
		localIDs[db.ID] = true
	}
	remoteIDs := make(map[uint]bool)
	for _, rs := range remoteServers {
		remoteIDs[rs.ID] = true
	}

	migrated := false
	for id := range metricsHistory {
		if localIDs[id] && remoteIDs[id] {
			delete(metricsHistory, id)
			migrated = true
		} else if remoteIDs[id] && !localIDs[id] {
			offsetID := id + 100000
			if _, exists := metricsHistory[offsetID]; !exists {
				metricsHistory[offsetID] = metricsHistory[id]
				delete(metricsHistory, id)
				migrated = true
			}
		}
	}

	if migrated {
		saveMetricsHistory()
	}
}

func saveMetricsHistory() {
	dataDir := os.Getenv("TRIM_PKGVAR")
	if dataDir == "" {
		dataDir = "./data"
	}
	os.MkdirAll(dataDir, 0755)

	// #7 使用 metricsMu 而非全局 mutex
	metricsMu.RLock()
	data, _ := json.Marshal(metricsHistory)
	metricsMu.RUnlock()

	atomicWriteFile(filepath.Join(dataDir, "metrics_history.json"), data, 0644)
}

func addMetricsSnapshot(s MetricsSnapshot) {
	metricsMu.Lock()
	defer metricsMu.Unlock()

	if metricsHistory == nil {
		metricsHistory = make(map[uint][]MetricsSnapshot)
	}

	history := metricsHistory[s.ServerID]
	history = append(history, s)

	maxSnapshots := 10080
	if len(history) > maxSnapshots {
		history = history[len(history)-maxSnapshots:]
	}

	metricsHistory[s.ServerID] = history
}

func getMetricsHistory(serverID uint, timeRangeSec int64) []MetricsSnapshot {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	history, ok := metricsHistory[serverID]
	if !ok {
		return []MetricsSnapshot{}
	}

	now := time.Now().Unix()
	cutoff := now - timeRangeSec

	var result []MetricsSnapshot
	for _, s := range history {
		if s.Timestamp >= cutoff {
			result = append(result, s)
		}
	}

	return result
}
