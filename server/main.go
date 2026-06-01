package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[panic] handler %s %s: %v", r.Method, r.URL.Path, err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{"code": 500, "msg": "服务器内部错误"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	initCrypto()
	loadData()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			checkScheduledBackups()
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[recover] collectAllMetrics panic: %v\n", r)
			}
		}()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			collectAllMetrics()
			saveMetricsHistory()
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			syncAllPorts()
		}
	}()

	GetHealthService().Start()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/system/info", systemInfoHandler)
	http.HandleFunc("/api/dashboard/metrics", dashboardMetricsHandler)
	http.HandleFunc("/api/dashboard/snapshot", dashboardHistoryHandler)
	http.HandleFunc("/api/databases/refresh", refreshHandler)
	http.HandleFunc("/api/databases/detect", detectHandler)
	http.HandleFunc("/api/sync/ports", syncPortsHandler)
	http.HandleFunc("/api/databases/db", dbHandler)
	http.HandleFunc("/api/databases/db/search", searchHandler)
	http.HandleFunc("/api/databases/db/", getHandler)
	http.HandleFunc("/api/databases/db/list/", listHandler)
	http.HandleFunc("/api/databases/db/check", checkHandler)

	http.HandleFunc("/api/remote-servers", remoteServerHandler)
	http.HandleFunc("/api/remote-servers/", remoteServerDetailHandler)
	http.HandleFunc("/api/remote-servers/test", remoteServerTestHandler)

	http.HandleFunc("/api/backups", backupHandler)
	http.HandleFunc("/api/backups/import", backupImportHandler)
	http.HandleFunc("/api/backups/restore", backupRestoreHandler)
	http.HandleFunc("/api/backups/scheduled", scheduledBackupHandler)
	http.HandleFunc("/api/backups/scheduled/run", runScheduledBackupHandler)
	http.HandleFunc("/api/backups/scheduled/", scheduledBackupDetailHandler)
	http.HandleFunc("/api/backups/retention", backupRetentionHandler)
	http.HandleFunc("/api/backups/", backupDetailHandler)

	http.HandleFunc("/api/mysql/databases", mysqlDatabasesHandler)
	http.HandleFunc("/api/mysql/databases/create", mysqlCreateDatabaseHandler)
	http.HandleFunc("/api/mysql/databases/delete", mysqlDeleteDatabaseHandler)
	http.HandleFunc("/api/mysql/tables", mysqlTablesHandler)
	http.HandleFunc("/api/mysql/columns", mysqlColumnsHandler)
	http.HandleFunc("/api/mysql/data", mysqlDataHandler)
	http.HandleFunc("/api/mysql/execute", mysqlExecuteHandler)
	http.HandleFunc("/api/mysql/config", mysqlConfigHandler)
	http.HandleFunc("/api/mysql/logs", mysqlLogsHandler)
	http.HandleFunc("/api/mysql/logs/clear", mysqlLogsClearHandler)
	http.HandleFunc("/api/mysql/port", mysqlPortHandler)
	http.HandleFunc("/api/mysql/users", mysqlUsersHandler)
	http.HandleFunc("/api/mysql/users/rename", mysqlRenameUserHandler)
	http.HandleFunc("/api/mysql/users/grant", mysqlGrantHandler)
	http.HandleFunc("/api/mysql/users/db-grant", mysqlDbGrantHandler)
	http.HandleFunc("/api/mysql/ping", mysqlPingHandler)
	http.HandleFunc("/api/mysql/restart", mysqlRestartHandler)
	http.HandleFunc("/api/mysql/stop", mysqlStopHandler)

	http.HandleFunc("/api/redis/info", redisInfoHandler)
	http.HandleFunc("/api/redis/keys", redisKeysHandler)
	http.HandleFunc("/api/redis/key", redisKeyHandler)
	http.HandleFunc("/api/redis/execute", redisExecuteHandler)
	http.HandleFunc("/api/redis/config", redisConfigHandler)
	http.HandleFunc("/api/redis/logs", redisLogsHandler)
	http.HandleFunc("/api/redis/backup", redisBackupHandler)
	http.HandleFunc("/api/redis/restore", redisRestoreHandler)
	http.HandleFunc("/api/redis/restart", redisRestartHandler)
	http.HandleFunc("/api/redis/stop", redisStopHandler)
	http.HandleFunc("/api/log-config", logConfigHandler)
	http.HandleFunc("/api/system/logs", systemLogsHandler)
	http.HandleFunc("/api/system/logs/clear", systemLogsClearHandler)
	http.HandleFunc("/api/system/logs/write", systemLogWriteHandler)

	http.HandleFunc("/api/health/check", healthCheckHandler)
	http.HandleFunc("/api/health/check/", healthCheckDetailHandler)
	http.HandleFunc("/api/health/config", healthCheckConfigHandler)

	http.HandleFunc("/api/fs/browse", fsBrowseHandler)

	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	
	distPath := filepath.Join(execDir, "frontend", "dist")
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		distPath = filepath.Join(execDir, "ui", "dist")
	}
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		distPath = filepath.Join(execDir, "dist")
	}
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		distPath = filepath.Join(wd, "frontend", "dist")
	}
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		distPath = filepath.Join(wd, "dist")
	}
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		distPath = filepath.Join(wd, "..", "frontend", "dist")
	}
	
	fmt.Printf("Serving frontend from: %s\n", distPath)
	
	staticHandler := http.FileServer(http.Dir(distPath))
	
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		distFile := filepath.Join(distPath, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(distFile); err == nil {
			staticHandler.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(distPath, "index.html"))
	})

	port := os.Getenv("TRIM_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on :%s\n", port)
	sysLogInfo("SYSTEM", "系统服务启动")
	srv := &http.Server{Addr: ":" + port, Handler: recoveryMiddleware(http.DefaultServeMux)}
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("Shutting down server...")
		GetHealthService().Stop()
		saveData()
		saveMetricsHistory()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
	}
	fmt.Println("Server stopped")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

const AppVersion = "1.0.0"

func systemInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	username := "unknown"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}
	hostname := "unknown"
	if h, err := os.Hostname(); err == nil {
		hostname = h
	}
	writeJSON(w, map[string]interface{}{
		"username": username,
		"hostname": hostname,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"version":  AppVersion,
	})
}