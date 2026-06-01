package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HealthStatus struct {
	UID        string `json:"uid"`
	ServerID   uint   `json:"serverId"`
	Source     string `json:"source"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Online     bool   `json:"online"`
	LatencyMs  int64  `json:"latencyMs"`
	CheckedAt  int64  `json:"checkedAt"`
	Error      string `json:"error,omitempty"`
	PrevOnline *bool  `json:"prevOnline,omitempty"`
}

type HealthConfig struct {
	IntervalSec   int  `json:"intervalSec"`
	TimeoutSec    int  `json:"timeoutSec"`
	AlertEnabled  bool `json:"alertEnabled"`
	Enabled       bool `json:"enabled"`
}

type HealthCheckService struct {
	mu       sync.RWMutex
	cache    map[string]*HealthStatus
	config   HealthConfig
	stopCh   chan struct{}
	running  bool
	alertCh  chan HealthStatus
}

var (
	healthService *HealthCheckService
	healthOnce    sync.Once
)

func newHealthCheckService() *HealthCheckService {
	cfg := defaultHealthConfig()
	svc := &HealthCheckService{
		cache:   make(map[string]*HealthStatus),
		config:  cfg,
		stopCh:  make(chan struct{}),
		alertCh: make(chan HealthStatus, 64),
	}
	svc.loadConfig()
	return svc
}

func GetHealthService() *HealthCheckService {
	healthOnce.Do(func() {
		healthService = newHealthCheckService()
	})
	return healthService
}

func defaultHealthConfig() HealthConfig {
	return HealthConfig{
		IntervalSec:   30,
		TimeoutSec:    5,
		AlertEnabled:  true,
		Enabled:       true,
	}
}

func (s *HealthCheckService) loadConfig() {
	dataDir := getDataDir()
	configPath := filepath.Join(dataDir, "health_config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	var cfg HealthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	if cfg.IntervalSec < 10 {
		cfg.IntervalSec = 10
	}
	if cfg.IntervalSec > 300 {
		cfg.IntervalSec = 300
	}
	if cfg.TimeoutSec < 1 {
		cfg.TimeoutSec = 1
	}
	if cfg.TimeoutSec > 30 {
		cfg.TimeoutSec = 30
	}
	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()
}

func (s *HealthCheckService) saveConfig() {
	dataDir := getDataDir()
	configPath := filepath.Join(dataDir, "health_config.json")
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()
	data, _ := json.Marshal(cfg)
	atomicWriteFile(configPath, data, 0644)
}

func (s *HealthCheckService) GetConfig() HealthConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *HealthCheckService) UpdateConfig(cfg HealthConfig) {
	if cfg.IntervalSec < 10 {
		cfg.IntervalSec = 10
	}
	if cfg.IntervalSec > 300 {
		cfg.IntervalSec = 300
	}
	if cfg.TimeoutSec < 1 {
		cfg.TimeoutSec = 1
	}
	if cfg.TimeoutSec > 30 {
		cfg.TimeoutSec = 30
	}
	s.mu.Lock()
	wasRunning := s.running
	s.config = cfg
	s.mu.Unlock()
	s.saveConfig()
	if wasRunning {
		s.Stop()
		s.Start()
	}
}

func (s *HealthCheckService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	go s.runLoop()
	go s.processAlerts()

	fmt.Println("[health] 测活服务已启动")
	sysLogInfo("HEALTH", "数据库连接测活服务已启动")
}

func (s *HealthCheckService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	close(s.stopCh)
	s.running = false
	fmt.Println("[health] 测活服务已停止")
}

func (s *HealthCheckService) runLoop() {
	s.checkAll()
	for {
		s.mu.RLock()
		interval := time.Duration(s.config.IntervalSec) * time.Second
		enabled := s.config.Enabled
		s.mu.RUnlock()

		if !enabled {
			time.Sleep(5 * time.Second)
			continue
		}

		select {
		case <-s.stopCh:
			return
		case <-time.After(interval):
			s.checkAll()
		}
	}
}

func (s *HealthCheckService) checkAll() {
	var servers []struct {
		uid    string
		id     uint
		source string
		server RemoteServer
	}

	mutex.Lock()
	for _, db := range databases {
		servers = append(servers, struct {
			uid    string
			id     uint
			source string
			server RemoteServer
		}{
			uid:    fmt.Sprintf("l:%d", db.ID),
			id:     db.ID,
			source: "local",
			server: RemoteServer{
				ID:          db.ID,
				Name:        db.Name,
				Type:        db.Type,
				Version:     db.Version,
				Host:        db.Host,
				Port:        db.Port,
				Username:    db.Username,
				Password:    db.Password,
				SSL:         db.SSL,
				Description: db.Description,
				Disk:        db.Disk,
			},
		})
	}
	for _, rs := range remoteServers {
		servers = append(servers, struct {
			uid    string
			id     uint
			source string
			server RemoteServer
		}{
			uid:    fmt.Sprintf("r:%d", rs.ID),
			id:     rs.ID,
			source: "remote",
			server: rs,
		})
	}
	mutex.Unlock()

	s.mu.RLock()
	timeout := time.Duration(s.config.TimeoutSec) * time.Second
	s.mu.RUnlock()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, srv := range servers {
		wg.Add(1)
		sem <- struct{}{}
		go func(uid string, id uint, source string, server RemoteServer) {
			defer wg.Done()
			defer func() { <-sem }()
			status := s.checkOne(uid, id, source, server, timeout)
			s.setStatus(uid, status)
		}(srv.uid, srv.id, srv.source, srv.server)
	}
	wg.Wait()

	// 清理已删除实例的缓存状态
	validUids := make(map[string]bool, len(servers))
	for _, srv := range servers {
		validUids[srv.uid] = true
	}
	s.mu.Lock()
	for uid := range s.cache {
		if !validUids[uid] {
			delete(s.cache, uid)
		}
	}
	s.mu.Unlock()
}

func (s *HealthCheckService) checkOne(uid string, id uint, source string, server RemoteServer, timeout time.Duration) *HealthStatus {
	status := &HealthStatus{
		UID:       uid,
		ServerID:  id,
		Source:    source,
		Type:      server.Type,
		Name:      server.Name,
		Host:      server.Host,
		Port:      server.Port,
		CheckedAt: time.Now().Unix(),
	}

	prev := s.GetStatus(uid)
	if prev != nil {
		prevCopy := prev.Online
		status.PrevOnline = &prevCopy
	}

	start := time.Now()

	if strings.ToLower(server.Type) == "redis" {
		s.checkRedis(server, timeout, status)
	} else {
		s.checkMySQL(server, timeout, status)
	}

	status.LatencyMs = time.Since(start).Milliseconds()

	if prev != nil && prev.Online != status.Online {
		if status.Online {
			status.PrevOnline = nil
		}
		select {
		case s.alertCh <- *status:
		default:
		}
	}

	return status
}

func (s *HealthCheckService) checkMySQL(server RemoteServer, timeout time.Duration, status *HealthStatus) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[health] checkMySQL panic: %v\n", r)
			status.Online = false
			status.Error = fmt.Sprintf("panic: %v", r)
		}
	}()

	timeoutStr := fmt.Sprintf("%ds", int(timeout.Seconds()))
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=%s&readTimeout=%s&allowNativePasswords=true",
		server.Username, server.Password, server.Host, server.Port, timeoutStr, timeoutStr)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		status.Online = false
		status.Error = err.Error()
		return
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		status.Online = false
		status.Error = err.Error()
		return
	}
	status.Online = true
}

func (s *HealthCheckService) checkRedis(server RemoteServer, timeout time.Duration, status *HealthStatus) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[health] checkRedis panic: %v\n", r)
			status.Online = false
			status.Error = fmt.Sprintf("panic: %v", r)
		}
	}()

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		status.Online = false
		status.Error = err.Error()
		return
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	if server.Username != "" && server.Password != "" {
		resp, err := redisDo(conn, "AUTH", server.Username, server.Password)
		if err != nil {
			status.Online = false
			status.Error = fmt.Sprintf("认证失败: %v", err)
			return
		}
		if errStr, ok := resp.(string); ok && errStr != "OK" {
			status.Online = false
			status.Error = fmt.Sprintf("认证失败: %s", errStr)
			return
		}
	} else if server.Password != "" {
		resp, err := redisDo(conn, "AUTH", server.Password)
		if err != nil {
			status.Online = false
			status.Error = fmt.Sprintf("认证失败: %v", err)
			return
		}
		if errStr, ok := resp.(string); ok && errStr != "OK" {
			status.Online = false
			status.Error = fmt.Sprintf("认证失败: %s", errStr)
			return
		}
	}

	resp, err := redisDo(conn, "PING")
	if err != nil {
		status.Online = false
		status.Error = err.Error()
		return
	}
	if pong, ok := resp.(string); ok && pong == "PONG" {
		status.Online = true
		return
	}
	status.Online = false
	status.Error = "unexpected PING response"
}

func (s *HealthCheckService) setStatus(uid string, status *HealthStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[uid] = status
}

func (s *HealthCheckService) GetStatus(uid string) *HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.cache[uid]; ok {
		cp := *st
		return &cp
	}
	return nil
}

func (s *HealthCheckService) GetAllStatus() []HealthStatus {
	s.mu.RLock()
	result := make([]HealthStatus, 0, len(s.cache))
	for _, st := range s.cache {
		result = append(result, *st)
	}
	s.mu.RUnlock()
	return result
}

func (s *HealthCheckService) EnsureChecked() {
	s.mu.RLock()
	hasData := len(s.cache) > 0
	s.mu.RUnlock()
	if !hasData {
		s.checkAll()
	}
}

func (s *HealthCheckService) ForceCheck(uid string) *HealthStatus {
	var server RemoteServer
	var id uint
	var source string
	var found bool

	mutex.Lock()
	if strings.HasPrefix(uid, "r:") {
		sid, err := strconv.ParseUint(uid[2:], 10, 32)
		if err == nil {
			for _, rs := range remoteServers {
				if rs.ID == uint(sid) {
					server = rs
					id = rs.ID
					source = "remote"
					found = true
					break
				}
			}
		}
	} else if strings.HasPrefix(uid, "l:") {
		sid, err := strconv.ParseUint(uid[2:], 10, 32)
		if err == nil {
			for _, db := range databases {
				if db.ID == uint(sid) {
					server = RemoteServer{
						ID:       db.ID,
						Name:     db.Name,
						Type:     db.Type,
						Version:  db.Version,
						Host:     db.Host,
						Port:     db.Port,
						Username: db.Username,
						Password: db.Password,
						SSL:      db.SSL,
						Disk:     db.Disk,
					}
					id = db.ID
					source = "local"
					found = true
					break
				}
			}
		}
	}
	mutex.Unlock()

	if !found {
		return nil
	}

	s.mu.RLock()
	timeout := time.Duration(s.config.TimeoutSec) * time.Second
	s.mu.RUnlock()

	status := s.checkOne(uid, id, source, server, timeout)
	s.setStatus(uid, status)
	return status
}

func (s *HealthCheckService) processAlerts() {
	for {
		select {
		case <-s.stopCh:
			return
		case status := <-s.alertCh:
			s.mu.RLock()
			alertEnabled := s.config.AlertEnabled
			s.mu.RUnlock()
			if !alertEnabled {
				continue
			}

			if status.Online {
				sysLogInfo("HEALTH", fmt.Sprintf("连接恢复: %s (%s:%d) [%s]", status.Name, status.Host, status.Port, status.Type))
			} else {
				errMsg := ""
				if status.Error != "" {
					errMsg = fmt.Sprintf(" - %s", status.Error)
				}
				sysLogWarn("HEALTH", fmt.Sprintf("连接异常: %s (%s:%d) [%s]%s", status.Name, status.Host, status.Port, status.Type, errMsg))
			}
		}
	}
}

func (s *HealthCheckService) RemoveStatus(uid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, uid)
}

func (s *HealthCheckService) IsCacheValid(uid string, maxAgeSec int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.cache[uid]
	if !ok {
		return false
	}
	return time.Now().Unix()-st.CheckedAt <= maxAgeSec
}
