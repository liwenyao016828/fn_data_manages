package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func saveData() {
	mutex.Lock()
	dbCopy := make([]Database, len(databases))
	copy(dbCopy, databases)
	rsCopy := make([]RemoteServer, len(remoteServers))
	copy(rsCopy, remoteServers)
	bkCopy := make([]Backup, len(backups))
	copy(bkCopy, backups)
	schedCopy := make([]ScheduledBackup, len(scheduledBackups))
	copy(schedCopy, scheduledBackups)
	mutex.Unlock()

	dataDir := os.Getenv("TRIM_PKGVAR")
	if dataDir == "" {
		dataDir = "./data"
	}
	os.MkdirAll(dataDir, 0755)

	for i := range dbCopy {
		if !isPasswordEncrypted(dbCopy[i].Password) {
			dbCopy[i].Password = encryptPassword(dbCopy[i].Password)
		}
	}
	data, _ := json.Marshal(dbCopy)
	atomicWriteFile(filepath.Join(dataDir, "databases.json"), data, 0644)

	for i := range rsCopy {
		if !isPasswordEncrypted(rsCopy[i].Password) {
			rsCopy[i].Password = encryptPassword(rsCopy[i].Password)
		}
	}
	remoteData, _ := json.Marshal(rsCopy)
	atomicWriteFile(filepath.Join(dataDir, "remote_servers.json"), remoteData, 0644)

	backupData, _ := json.Marshal(bkCopy)
	atomicWriteFile(filepath.Join(dataDir, "backups.json"), backupData, 0644)

	schedData, _ := json.Marshal(schedCopy)
	atomicWriteFile(filepath.Join(dataDir, "scheduled_backups.json"), schedData, 0644)
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpPath, path)
}

func loadData() {
	dataDir := os.Getenv("TRIM_PKGVAR")
	if dataDir == "" {
		dataDir = "./data"
	}

	data, err := os.ReadFile(filepath.Join(dataDir, "databases.json"))
	if err == nil {
		json.Unmarshal(data, &databases)
		migrated := false
		for i := range databases {
			if databases[i].Password != "" && !isPasswordEncrypted(databases[i].Password) {
				migrated = true
			} else {
				databases[i].Password = decryptPassword(databases[i].Password)
			}
		}
		if len(databases) > 0 {
			nextID = databases[len(databases)-1].ID + 1
		}
		if migrated {
			saveData()
		}
	}

	remoteData, err := os.ReadFile(filepath.Join(dataDir, "remote_servers.json"))
	if err == nil {
		json.Unmarshal(remoteData, &remoteServers)
		migrated := false
		for i := range remoteServers {
			if remoteServers[i].Password != "" && !isPasswordEncrypted(remoteServers[i].Password) {
				migrated = true
			} else {
				remoteServers[i].Password = decryptPassword(remoteServers[i].Password)
			}
		}
		if len(remoteServers) > 0 {
			nextRemoteID = remoteServers[len(remoteServers)-1].ID + 1
		}
		if migrated {
			saveData()
		}
	}

	backupData, err := os.ReadFile(filepath.Join(dataDir, "backups.json"))
	if err == nil {
		json.Unmarshal(backupData, &backups)
		if len(backups) > 0 {
			nextBackupID = backups[len(backups)-1].ID + 1
		}
	}

	schedData, err := os.ReadFile(filepath.Join(dataDir, "scheduled_backups.json"))
	if err == nil {
		json.Unmarshal(schedData, &scheduledBackups)
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
	metricsData, err := os.ReadFile(filepath.Join(dataDir, "metrics_history.json"))
	if err == nil {
		json.Unmarshal(metricsData, &metricsHistory)
	}

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

	mutex.Lock()
	data, _ := json.Marshal(metricsHistory)
	mutex.Unlock()

	atomicWriteFile(filepath.Join(dataDir, "metrics_history.json"), data, 0644)
}

func addMetricsSnapshot(s MetricsSnapshot) {
	mutex.Lock()
	defer mutex.Unlock()

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
	mutex.Lock()
	defer mutex.Unlock()

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