package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func dashboardMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}

	timeRangeStr := r.URL.Query().Get("time_range")
	timeRangeSec, err := strconv.ParseInt(timeRangeStr, 10, 64)
	if err != nil || timeRangeSec <= 0 {
		timeRangeSec = 3600
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	source := r.URL.Query().Get("source")

	server := findAnyServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	if strings.ToLower(server.Type) == "redis" {
		writeRedisMetrics(w, server, uint(id), timeRangeSec)
		return
	}

	online := checkMySQLOnline(server)
	writeMySQLMetrics(w, server, uint(id), timeRangeSec, online)
}

func checkMySQLOnline(server *RemoteServer) bool {
	uid := fmt.Sprintf("l:%d", server.ID)
	svc := GetHealthService()
	if svc.IsCacheValid(uid, 60) {
		if st := svc.GetStatus(uid); st != nil {
			return st.Online
		}
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[recover] checkMySQLOnline panic: %v\n", r)
		}
	}()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=5s&readTimeout=5s&allowNativePasswords=true",
		server.Username, server.Password, server.Host, server.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false
	}
	defer db.Close()
	err = db.Ping()
	return err == nil
}

func buildRangeStats(history []MetricsSnapshot) map[string]interface{} {
	if len(history) < 2 {
		return map[string]interface{}{"available": false}
	}

	// 计算增量时处理计数器重置（MySQL重启后计数器归零）
	// 逐段累加 delta，遇到负值时使用当前值（说明计数器被重置了）
	calcDelta := func(getVal func(s MetricsSnapshot) int64) int64 {
		var total int64
		for i := 1; i < len(history); i++ {
			diff := getVal(history[i]) - getVal(history[i-1])
			if diff < 0 {
				// 计数器重置，使用当前值作为增量
				total += getVal(history[i])
			} else {
				total += diff
			}
		}
		return total
	}

	var sumThreadsRunning int64
	var sumBPHitRate float64
	var sumBPPagesDirty int64
	var countBP int64
	for _, s := range history {
		sumThreadsRunning += s.ThreadsRunning
		if s.BPHitRate > 0 || s.BPPagesTotal > 0 {
			sumBPHitRate += s.BPHitRate
			sumBPPagesDirty += s.BPPagesDirty
			countBP++
		}
	}
	n := int64(len(history))

	result := map[string]interface{}{
		"available":          true,
		"deltaBytesReceived": calcDelta(func(s MetricsSnapshot) int64 { return s.BytesReceived }),
		"deltaBytesSent":     calcDelta(func(s MetricsSnapshot) int64 { return s.BytesSent }),
		"deltaComSelect":     calcDelta(func(s MetricsSnapshot) int64 { return s.ComSelect }),
		"deltaComInsert":     calcDelta(func(s MetricsSnapshot) int64 { return s.ComInsert }),
		"deltaComUpdate":     calcDelta(func(s MetricsSnapshot) int64 { return s.ComUpdate }),
		"deltaComDelete":     calcDelta(func(s MetricsSnapshot) int64 { return s.ComDelete }),
		"deltaSlowQueries":   calcDelta(func(s MetricsSnapshot) int64 { return s.SlowQueries }),
		"avgThreadsRunning":  fmt.Sprintf("%.1f", float64(sumThreadsRunning)/float64(n)),
		"avgBPHitRate":       fmt.Sprintf("%.2f", float64(0)),
		"avgBPPagesDirty":    fmt.Sprintf("%.0f", float64(0)),
	}

	if countBP > 0 {
		result["avgBPHitRate"] = fmt.Sprintf("%.2f", sumBPHitRate/float64(countBP))
		result["avgBPPagesDirty"] = fmt.Sprintf("%.0f", float64(sumBPPagesDirty)/float64(countBP))
	}

	return result
}

func buildEChartsConfig(history []MetricsSnapshot, isRedis bool) map[string]interface{} {
	var connSeriesData [][]interface{}
	var qpsSeriesData [][]interface{}
	var tpsSeriesData [][]interface{}
	var netInSeriesData [][]interface{}
	var netOutSeriesData [][]interface{}

	maxPoints := 200
	step := 1
	if len(history) > maxPoints {
		step = len(history) / maxPoints
		if step < 1 {
			step = 1
		}
	}

	for i, snapshot := range history {
		if step > 1 && i%step != 0 && i != len(history)-1 {
			continue
		}
		timestamp := snapshot.Timestamp * 1000
		connSeriesData = append(connSeriesData, []interface{}{timestamp, snapshot.Connections})
		qpsSeriesData = append(qpsSeriesData, []interface{}{timestamp, snapshot.QPS})
		if !isRedis {
			tpsSeriesData = append(tpsSeriesData, []interface{}{timestamp, snapshot.TPS})

			var netInRate float64
			var netOutRate float64
			if i > 0 {
				prev := history[i-1]
				elapsed := snapshot.Timestamp - prev.Timestamp
				if elapsed > 0 && elapsed <= 120 && prev.BytesReceived > 0 && prev.BytesSent > 0 {
					deltaIn := snapshot.BytesReceived - prev.BytesReceived
					deltaOut := snapshot.BytesSent - prev.BytesSent
					if deltaIn >= 0 {
						netInRate = float64(deltaIn) / float64(elapsed)
					}
					if deltaOut >= 0 {
						netOutRate = float64(deltaOut) / float64(elapsed)
					}
				}
				maxNetRate := 10737418240.0
				if netInRate > maxNetRate {
					netInRate = 0
				}
				if netOutRate > maxNetRate {
					netOutRate = 0
				}
				netInSeriesData = append(netInSeriesData, []interface{}{timestamp, netInRate})
				netOutSeriesData = append(netOutSeriesData, []interface{}{timestamp, netOutRate})
			}
		}
	}

	var connName string
	var qpsName string
	if isRedis {
		connName = "连接数"
		qpsName = "操作数/秒"
	} else {
		connName = "连接数"
		qpsName = "QPS"
	}

	series := []map[string]interface{}{
		{
			"name": connName,
			"type": "line",
			"data": connSeriesData,
		},
		{
			"name": qpsName,
			"type": "line",
			"data": qpsSeriesData,
		},
	}

	if !isRedis {
		series = append(series, map[string]interface{}{
			"name": "TPS",
			"type": "line",
			"data": tpsSeriesData,
		})
	}

	result := map[string]interface{}{
		"xAxis": map[string]interface{}{
			"type": "time",
		},
		"yAxis": []map[string]interface{}{
			{
				"type": "value",
				"name": connName,
			},
			{
				"type": "value",
				"name": qpsName,
			},
		},
		"series": series,
	}

	if !isRedis {
		result["network"] = map[string]interface{}{
			"series": []map[string]interface{}{
				{
					"name": "接收",
					"type": "line",
					"data": netInSeriesData,
				},
				{
					"name": "发送",
					"type": "line",
					"data": netOutSeriesData,
				},
			},
		}
	}

	return result
}

func writeMySQLMetrics(w http.ResponseWriter, server *RemoteServer, serverID uint, timeRangeSec int64, online bool) {
	history := getMetricsHistory(serverID, timeRangeSec)
	echarts := buildEChartsConfig(history, false)
	rangeStats := buildRangeStats(history)

	if !online {
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"type":       "mysql",
				"online":     false,
				"traffic":    map[string]interface{}{"echarts": echarts},
				"rangeStats": rangeStats,
			},
		})
		return
	}

	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"type":       "mysql",
				"online":     false,
				"traffic":    map[string]interface{}{"echarts": echarts},
				"rangeStats": rangeStats,
			},
		})
		return
	}
	defer db.Close()

	statusMap := make(map[string]string)
	rows, err := db.Query("SHOW GLOBAL STATUS")
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"type":       "mysql",
				"online":     false,
				"traffic":    map[string]interface{}{"echarts": echarts},
				"rangeStats": rangeStats,
			},
		})
		return
	}
	for rows.Next() {
		var name, val string
		rows.Scan(&name, &val)
		statusMap[name] = val
	}
	rows.Close()

	varMap := make(map[string]string)
	varRows, err := db.Query("SHOW VARIABLES WHERE Variable_name IN ('version', 'max_connections', 'innodb_buffer_pool_size', 'datadir', 'basedir')")
	if err == nil {
		for varRows.Next() {
			var name, val string
			varRows.Scan(&name, &val)
			varMap[name] = val
		}
		varRows.Close()
	}

	uptime, _ := strconv.ParseInt(statusMap["Uptime"], 10, 64)
	questions, _ := strconv.ParseInt(statusMap["Questions"], 10, 64)
	comSelect, _ := strconv.ParseInt(statusMap["Com_select"], 10, 64)
	comInsert, _ := strconv.ParseInt(statusMap["Com_insert"], 10, 64)
	comUpdate, _ := strconv.ParseInt(statusMap["Com_update"], 10, 64)
	comDelete, _ := strconv.ParseInt(statusMap["Com_delete"], 10, 64)

	running := statusMap["Threads_running"]
	connected := statusMap["Threads_connected"]
	slowQueries := statusMap["Slow_queries"]
	bytesReceived := statusMap["Bytes_received"]
	bytesSent := statusMap["Bytes_sent"]

	bpReadReq, _ := strconv.ParseInt(statusMap["Innodb_buffer_pool_read_requests"], 10, 64)
	bpReads, _ := strconv.ParseInt(statusMap["Innodb_buffer_pool_reads"], 10, 64)
	bpPagesFree := statusMap["Innodb_buffer_pool_pages_free"]
	bpPagesTotal := statusMap["Innodb_buffer_pool_pages_total"]
	bpPagesDirty := statusMap["Innodb_buffer_pool_pages_dirty"]

	tableLocksWaited := statusMap["Table_locks_waited"]
	createdTmpDisk := statusMap["Created_tmp_disk_tables"]
	createdTmpTables := statusMap["Created_tmp_tables"]
	abortedConnects := statusMap["Aborted_connects"]
	maxUsedConn := statusMap["Max_used_connections"]
	deadlocks := statusMap["Innodb_deadlocks"]

	var bpHitRate float64
	if bpReadReq > 0 {
		bpHitRate = float64(bpReadReq-bpReads) / float64(bpReadReq) * 100
	}

	maxConns, _ := strconv.ParseInt(varMap["max_connections"], 10, 64)
	connCurrent, _ := strconv.ParseInt(connected, 10, 64)
	var connUsage float64
	if maxConns > 0 {
		connUsage = float64(connCurrent) / float64(maxConns) * 100
	}

	conns, _ := strconv.ParseInt(statusMap["Connections"], 10, 64)

	var qps float64
	if uptime > 0 {
		qps = float64(questions) / float64(uptime)
	}

	bpSize := varMap["innodb_buffer_pool_size"]
	if bpSize == "" {
		bpSize = statusMap["Innodb_buffer_pool_bytes_data"]
	}

	var processList []map[string]interface{}
	pRows, pErr := db.Query("SELECT Id, User, Host, db, Command, Time, State, IFNULL(Info,'') FROM information_schema.PROCESSLIST ORDER BY Time DESC")
	if pErr == nil {
		defer pRows.Close()
		for pRows.Next() {
			var id, user, host, dbName, command, state, info string
			var time int64
			pRows.Scan(&id, &user, &host, &dbName, &command, &time, &state, &info)
			processList = append(processList, map[string]interface{}{
				"id":      id,
				"user":    user,
				"host":    host,
				"db":      dbName,
				"command": command,
				"time":    time,
				"state":   state,
				"info":    info,
			})
		}
	}

	var dataTotalSize string = "-"
	var diskRemaining string = "-"

	var totalSize float64
	row := db.QueryRow("SELECT COALESCE(SUM(data_length + index_length), 0) / 1024 / 1024 FROM information_schema.TABLES WHERE table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')")
	if err := row.Scan(&totalSize); err == nil && totalSize > 0 {
		if totalSize > 1024 {
			dataTotalSize = fmt.Sprintf("%.2f GB", totalSize/1024)
		} else {
			dataTotalSize = fmt.Sprintf("%.2f MB", totalSize)
		}
	}

	var freeSpace float64
	row2 := db.QueryRow("SELECT SUM(DATA_FREE) / 1024 / 1024 / 1024 FROM information_schema.FILES WHERE FILE_TYPE = 'TABLESPACE'")
	if err := row2.Scan(&freeSpace); err == nil && freeSpace > 0 && freeSpace < 1000 {
		diskRemaining = fmt.Sprintf("%.2f GB", freeSpace)
	}

	now := map[string]interface{}{
		"online": connCurrent,
	}

	totals := map[string]interface{}{
		"inBytes":  bytesReceived,
		"outBytes": bytesSent,
	}

	writeJSON(w, map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"type":     "mysql",
			"online":   true,
			"host":     server.Host,
			"port":     server.Port,
			"version":  varMap["version"],
			"rangeSec": timeRangeSec,

			"uptime":         uptime,
			"uptime_display": formatUptime(uptime),

			"questions":  questions,
			"qps":        fmt.Sprintf("%.2f", qps),
			"com_select": comSelect,
			"com_insert": comInsert,
			"com_update": comUpdate,
			"com_delete": comDelete,

			"threads_running":     running,
			"threads_connected":   connected,
			"threads_cached":      statusMap["Threads_cached"],
			"max_connections":     fmt.Sprintf("%d", maxConns),
			"connection_usage":    fmt.Sprintf("%.1f", connUsage),
			"max_used_connection": maxUsedConn,
			"connections_total":   conns,

			"bytes_received": bytesReceived,
			"bytes_sent":     bytesSent,
			"network_in":     formatBytes(bytesReceived),
			"network_out":    formatBytes(bytesSent),

			"slow_queries":       slowQueries,
			"table_locks_waited": tableLocksWaited,
			"tmp_table_disk":     createdTmpDisk,
			"tmp_table_total":    createdTmpTables,
			"aborted_connects":   abortedConnects,
			"deadlocks":          deadlocks,

			"innodb_buffer_pool_size":          bpSize,
			"innodb_buffer_pool_hit_rate":      fmt.Sprintf("%.2f", bpHitRate),
			"innodb_buffer_pool_pages_free":    bpPagesFree,
			"innodb_buffer_pool_pages_total":   bpPagesTotal,
			"innodb_buffer_pool_pages_dirty":   bpPagesDirty,
			"innodb_buffer_pool_read_requests": fmt.Sprintf("%d", bpReadReq),
			"innodb_buffer_pool_reads":         fmt.Sprintf("%d", bpReads),
			"processlist":                      processList,
			"data_total_size":                  dataTotalSize,
			"disk_remaining":                   diskRemaining,

			"traffic": map[string]interface{}{
				"echarts": echarts,
			},
			"rangeStats": rangeStats,
			"now":        now,
			"totals":     totals,
		},
	})
}

func writeRedisMetrics(w http.ResponseWriter, server *RemoteServer, serverID uint, timeRangeSec int64) {
	history := getMetricsHistory(serverID, timeRangeSec)
	echarts := buildEChartsConfig(history, true)

	conn, err := openRedis(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"type":    "redis",
				"online":  false,
				"host":    server.Host,
				"port":    server.Port,
				"traffic": map[string]interface{}{"echarts": echarts},
			},
		})
		return
	}
	defer conn.Close()

	resp, err := redisDo(conn, "INFO")
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"type":    "redis",
				"online":  false,
				"host":    server.Host,
				"port":    server.Port,
				"traffic": map[string]interface{}{"echarts": echarts},
			},
		})
		return
	}

	infoStr, ok := resp.(string)
	if !ok {
		writeJSON(w, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"type":    "redis",
				"online":  false,
				"host":    server.Host,
				"port":    server.Port,
				"traffic": map[string]interface{}{"echarts": echarts},
			},
		})
		return
	}

	infoMap := make(map[string]string)
	for _, line := range strings.Split(infoStr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			infoMap[parts[0]] = parts[1]
		}
	}

	uptime, _ := strconv.ParseInt(infoMap["uptime_in_seconds"], 10, 64)
	opsPerSec := infoMap["instantaneous_ops_per_sec"]
	usedMemory := infoMap["used_memory_human"]
	memPeak := infoMap["used_memory_peak_human"]
	keyspaceHits, _ := strconv.ParseInt(infoMap["keyspace_hits"], 10, 64)
	keyspaceMisses, _ := strconv.ParseInt(infoMap["keyspace_misses"], 10, 64)

	var hitRate float64
	if keyspaceHits+keyspaceMisses > 0 {
		hitRate = float64(keyspaceHits) / float64(keyspaceHits+keyspaceMisses) * 100
	}

	history = getMetricsHistory(serverID, timeRangeSec)

	online, _ := strconv.ParseInt(infoMap["connected_clients"], 10, 64)
	now := map[string]interface{}{
		"online": online,
	}

	totals := map[string]interface{}{
		"inBytes":  0,
		"outBytes": 0,
	}

	writeJSON(w, map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"type":     "redis",
			"host":     server.Host,
			"port":     server.Port,
			"version":  infoMap["redis_version"],
			"rangeSec": timeRangeSec,

			"uptime":         uptime,
			"uptime_display": formatUptime(uptime),

			"connected_clients": infoMap["connected_clients"],
			"blocked_clients":   infoMap["blocked_clients"],
			"maxclients":        infoMap["maxclients"],

			"used_memory":       usedMemory,
			"used_memory_peak":  memPeak,
			"mem_fragmentation": infoMap["mem_fragmentation_ratio"],

			"ops_per_sec":       opsPerSec,
			"total_commands":    infoMap["total_commands_processed"],
			"total_connections": infoMap["total_connections_received"],

			"keyspace_hits":   fmt.Sprintf("%d", keyspaceHits),
			"keyspace_misses": fmt.Sprintf("%d", keyspaceMisses),
			"hit_rate":        fmt.Sprintf("%.2f", hitRate),

			"rdb_last_save": infoMap["rdb_last_save_time"],
			"rdb_changes":   infoMap["rdb_changes_since_last_save"],
			"aof_enabled":   infoMap["aof_enabled"],

			"traffic": map[string]interface{}{
				"echarts": buildEChartsConfig(history, true),
			},
			"now":    now,
			"totals": totals,
		},
	})
}

func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%d天 %d小时 %d分", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小时 %d分", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

func formatBytes(val string) string {
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return val
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	var i int
	f := float64(n)
	for i = 0; i < len(units)-1 && f >= 1024; i++ {
		f /= 1024
	}
	return fmt.Sprintf("%.2f %s", f, units[i])
}

func dashboardHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	serverID := r.URL.Query().Get("server_id")
	if serverID == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "server_id required"})
		return
	}

	id, err := strconv.ParseUint(serverID, 10, 32)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "invalid server_id"})
		return
	}

	source := r.URL.Query().Get("source")

	server := findAnyServer(uint(id), source)
	if server == nil {
		writeJSON(w, map[string]interface{}{"code": 404, "msg": "server not found"})
		return
	}

	if strings.ToLower(server.Type) == "redis" {
		writeRedisSnapshot(w, server)
		return
	}

	writeMySQLSnapshot(w, server)
}

func writeMySQLSnapshot(w http.ResponseWriter, server *RemoteServer) {
	db, err := openMySQL(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false}})
		return
	}
	defer db.Close()

	statusMap := make(map[string]string)
	rows, err := db.Query("SHOW GLOBAL STATUS")
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false}})
		return
	}
	for rows.Next() {
		var name, val string
		rows.Scan(&name, &val)
		statusMap[name] = val
	}
	rows.Close()

	connected, _ := strconv.ParseInt(statusMap["Threads_connected"], 10, 64)
	questions, _ := strconv.ParseInt(statusMap["Questions"], 10, 64)
	bpReadReq, _ := strconv.ParseInt(statusMap["Innodb_buffer_pool_read_requests"], 10, 64)
	bpReads, _ := strconv.ParseInt(statusMap["Innodb_buffer_pool_reads"], 10, 64)
	bytesRecv, _ := strconv.ParseInt(statusMap["Bytes_received"], 10, 64)
	bytesSent, _ := strconv.ParseInt(statusMap["Bytes_sent"], 10, 64)

	var bpHit float64
	if bpReadReq > 0 {
		bpHit = float64(bpReadReq-bpReads) / float64(bpReadReq) * 100
	}

	writeJSON(w, map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"connections":  connected,
			"questions":    questions,
			"bp_hit_rate":  fmt.Sprintf("%.1f", bpHit),
			"bytes_recv":   bytesRecv,
			"bytes_sent":   bytesSent,
			"threads_run":  statusMap["Threads_running"],
			"slow_queries": statusMap["Slow_queries"],
			"tmp_tables":   statusMap["Created_tmp_disk_tables"],
		},
	})
}

func writeRedisSnapshot(w http.ResponseWriter, server *RemoteServer) {
	conn, err := openRedis(server)
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false}})
		return
	}
	defer conn.Close()

	resp, err := redisDo(conn, "INFO", "stats")
	if err != nil {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false}})
		return
	}

	infoStr, ok := resp.(string)
	if !ok {
		writeJSON(w, map[string]interface{}{"code": 0, "data": map[string]interface{}{"online": false}})
		return
	}

	infoMap := make(map[string]string)
	for _, line := range strings.Split(infoStr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			infoMap[parts[0]] = parts[1]
		}
	}

	connected, _ := strconv.ParseInt(infoMap["connected_clients"], 10, 64)
	ops, _ := strconv.ParseFloat(infoMap["instantaneous_ops_per_sec"], 64)
	hits, _ := strconv.ParseInt(infoMap["keyspace_hits"], 10, 64)
	misses, _ := strconv.ParseInt(infoMap["keyspace_misses"], 10, 64)

	var hitRate float64
	if hits+misses > 0 {
		hitRate = float64(hits) / float64(hits+misses) * 100
	}

	writeJSON(w, map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"connections": connected,
			"ops_per_sec": ops,
			"hit_rate":    fmt.Sprintf("%.1f", hitRate),
		},
	})
}

type dashboardRequest struct {
	ServerID uint `json:"server_id"`
}

func jsonDecode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func collectAllMetrics() {
	mutex.Lock()
	localCopy := make([]RemoteServer, len(databases))
	for i, db := range databases {
		localCopy[i] = RemoteServer{
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
		}
	}
	remoteCopy := make([]RemoteServer, len(remoteServers))
	copy(remoteCopy, remoteServers)
	mutex.Unlock()

	collectServerMetrics := func(server RemoteServer, historyID uint) {
		snapshot := MetricsSnapshot{
			ServerID:  historyID,
			Timestamp: time.Now().Unix(),
		}

		if strings.ToLower(server.Type) == "redis" {
			conn, err := openRedis(&server)
			if err == nil {
				resp, _ := redisDo(conn, "INFO", "stats")
				conn.Close()
				infoStr, ok := resp.(string)
				if ok {
					infoMap := make(map[string]string)
					for _, line := range strings.Split(infoStr, "\n") {
						line = strings.TrimSpace(line)
						if line == "" || strings.HasPrefix(line, "#") {
							continue
						}
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							infoMap[parts[0]] = parts[1]
						}
					}
					snapshot.Connections, _ = strconv.ParseInt(infoMap["connected_clients"], 10, 64)
					snapshot.QPS, _ = strconv.ParseFloat(infoMap["instantaneous_ops_per_sec"], 64)
				}
			}
			addMetricsSnapshot(snapshot)
		} else {
			db, err := openMySQL(&server)
			if err == nil {
				statusMap := make(map[string]string)
				rows, qErr := db.Query("SHOW GLOBAL STATUS")
				if qErr == nil {
					for rows.Next() {
						var name, val string
						rows.Scan(&name, &val)
						statusMap[name] = val
					}
					rows.Close()
				}
				db.Close()

				snapshot.Connections, _ = strconv.ParseInt(statusMap["Threads_connected"], 10, 64)
				snapshot.ThreadsRunning, _ = strconv.ParseInt(statusMap["Threads_running"], 10, 64)
				snapshot.BytesReceived, _ = strconv.ParseInt(statusMap["Bytes_received"], 10, 64)
				snapshot.BytesSent, _ = strconv.ParseInt(statusMap["Bytes_sent"], 10, 64)
				snapshot.ComSelect, _ = strconv.ParseInt(statusMap["Com_select"], 10, 64)
				snapshot.ComInsert, _ = strconv.ParseInt(statusMap["Com_insert"], 10, 64)
				snapshot.ComUpdate, _ = strconv.ParseInt(statusMap["Com_update"], 10, 64)
				snapshot.ComDelete, _ = strconv.ParseInt(statusMap["Com_delete"], 10, 64)
				snapshot.SlowQueries, _ = strconv.ParseInt(statusMap["Slow_queries"], 10, 64)
				snapshot.BPPagesTotal, _ = strconv.ParseInt(statusMap["Innodb_buffer_pool_pages_total"], 10, 64)
				snapshot.BPPagesFree, _ = strconv.ParseInt(statusMap["Innodb_buffer_pool_pages_free"], 10, 64)
				snapshot.BPPagesDirty, _ = strconv.ParseInt(statusMap["Innodb_buffer_pool_pages_dirty"], 10, 64)

				bpReadReq, _ := strconv.ParseInt(statusMap["Innodb_buffer_pool_read_requests"], 10, 64)
				bpReads, _ := strconv.ParseInt(statusMap["Innodb_buffer_pool_reads"], 10, 64)
				if bpReadReq > 0 {
					snapshot.BPHitRate = float64(bpReadReq-bpReads) / float64(bpReadReq) * 100
				}

				questions, _ := strconv.ParseInt(statusMap["Questions"], 10, 64)
				comInsert, _ := strconv.ParseInt(statusMap["Com_insert"], 10, 64)
				comUpdate, _ := strconv.ParseInt(statusMap["Com_update"], 10, 64)
				comDelete, _ := strconv.ParseInt(statusMap["Com_delete"], 10, 64)
				totalWrites := comInsert + comUpdate + comDelete

				nowTs := time.Now().Unix()
				lastRawMutex.Lock()
				prev, hasPrev := lastRawCounters[historyID]
				if hasPrev && nowTs > prev.Timestamp && nowTs-prev.Timestamp <= 300 {
					elapsed := nowTs - prev.Timestamp
					deltaQ := questions - prev.Questions
					deltaW := totalWrites - prev.Writes
					if deltaQ >= 0 {
						snapshot.QPS = float64(deltaQ) / float64(elapsed)
					} else {
						snapshot.QPS = float64(questions) / float64(nowTs)
					}
					if deltaW >= 0 {
						snapshot.TPS = float64(deltaW) / float64(elapsed)
					}
				}
				lastRawCounters[historyID] = rawCounter{Questions: questions, Writes: totalWrites, Timestamp: nowTs}
				lastRawMutex.Unlock()

				addMetricsSnapshot(snapshot)
			}
		}
	}

	for _, s := range localCopy {
		collectServerMetrics(s, s.ID)
	}
	for _, s := range remoteCopy {
		collectServerMetrics(s, s.ID+100000)
	}
}
