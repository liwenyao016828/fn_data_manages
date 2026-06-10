package main

import "sync"

type Database struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Version     string   `json:"version"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	Database    string   `json:"database"`
	SSL         bool     `json:"ssl"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Disk        string   `json:"disk"`
	Permission  string   `json:"permission"`
	Container   string   `json:"container"`
	Databases   []string `json:"databases"`
	CreatedAt   string   `json:"createdAt"`
}

type PageInfo struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type DatabaseSearch struct {
	PageInfo
	Info string `json:"info"`
	Type string `json:"type"`
}

type DatabaseCreate struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Database    string `json:"database"`
	SSL         bool   `json:"ssl"`
	Description string `json:"description"`
	Permission  string `json:"permission"`
	Disk        string `json:"disk"`
	Container   string `json:"container"`
	TestOnly    bool   `json:"testOnly"`
	TestID      uint   `json:"testId"`
	TestSource  string `json:"testSource"`
}

type DatabaseUpdate struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	SSL         bool   `json:"ssl"`
	Description string `json:"description"`
	Permission  string `json:"permission"`
	Disk        string `json:"disk"`
	Container   string `json:"container"`
}

type DetectedInstance struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Source        string   `json:"source"`        // "Docker" or "宿主机"
	Image         string   `json:"image,omitempty"` // Docker image name
	Username      string   `json:"username"`
	Password      string   `json:"password"`
	Status        string   `json:"status"`
	Version       string   `json:"version"`
	Container     string   `json:"container"`
	ContainerID   string   `json:"containerId"`
	WeakPassword  bool     `json:"weakPassword"`
	ReachableFrom []string `json:"reachableFrom,omitempty"`
	Fingerprint   string   `json:"fingerprint"`
	Ignored       bool     `json:"ignored"`
}

type RemoteServer struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	SSL         bool   `json:"ssl"`
	Description string `json:"description"`
	Disk        string `json:"disk"`
	Container   string `json:"container"`
	CreatedAt   string `json:"createdAt"`
}

type Backup struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	ServerID    uint   `json:"serverId"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Database    string `json:"database"`
	FileName    string `json:"fileName"`
	FileSize    int64  `json:"fileSize"`
	Status      string `json:"status"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	BackupType  string `json:"backupType"`
	BackupLevel string `json:"backupLevel"`
	Source      string `json:"source"`
}

type ScheduledBackup struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	BackupLevel string `json:"backupLevel"`
	ServerID    uint   `json:"serverId"`
	Source      string `json:"source"`
	Database    string `json:"database"`
	Cron        string `json:"cron"`
	Label       string `json:"label"`
	Enabled     bool   `json:"enabled"`
	RetainCount int    `json:"retainCount"`
	LastRun     string `json:"lastRun"`
	CreatedAt   string `json:"createdAt"`
}

type PageResult struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
}

type MetricsSnapshot struct {
	ServerID       uint    `json:"serverId"`
	Timestamp      int64   `json:"timestamp"`
	Connections    int64   `json:"connections"`
	QPS            float64 `json:"qps"`
	TPS            float64 `json:"tps"`
	ThreadsRunning int64   `json:"threadsRunning"`
	BytesReceived  int64   `json:"bytesReceived"`
	BytesSent      int64   `json:"bytesSent"`
	ComSelect      int64   `json:"comSelect"`
	ComInsert      int64   `json:"comInsert"`
	ComUpdate      int64   `json:"comUpdate"`
	ComDelete      int64   `json:"comDelete"`
	SlowQueries    int64   `json:"slowQueries"`
	BPHitRate      float64 `json:"bpHitRate"`
	BPPagesDirty   int64   `json:"bpPagesDirty"`
	BPPagesFree    int64   `json:"bpPagesFree"`
	BPPagesTotal   int64   `json:"bpPagesTotal"`
}

type rawCounter struct {
	Questions int64
	Writes    int64
	Timestamp int64
}

var (
	databases        []Database
	remoteServers    []RemoteServer
	backups          []Backup
	scheduledBackups []ScheduledBackup
	metricsHistory   map[uint][]MetricsSnapshot
	mutex            sync.Mutex // #7 兼容用：保留旧全局锁，新代码逐步迁移到细分锁
	nextID           uint = 1
	nextRemoteID     uint = 1
	nextBackupID     uint = 1
	nextSchedID      uint = 1
	lastRawCounters  = make(map[uint]rawCounter)
	lastRawMutex     sync.Mutex

	// #7 细分锁：拆分全局 mutex，避免备份/调度与列表查询互斥
	// 命名约定：<域>Mu
	dbMu      sync.RWMutex // 保护 databases
	remoteMu  sync.RWMutex // 保护 remoteServers
	backupMu  sync.RWMutex // 保护 backups + scheduledBackups
	metricsMu sync.RWMutex // 保护 metricsHistory
	idMu      sync.Mutex   // 保护 nextID/nextRemoteID/nextBackupID/nextSchedID
)