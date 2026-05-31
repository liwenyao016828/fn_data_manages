package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	svc := GetHealthService()

	switch r.Method {
	case "GET":
		uid := r.URL.Query().Get("uid")
		if uid != "" {
			status := svc.GetStatus(uid)
			needRecheck := status == nil || !status.Online || !svc.IsCacheValid(uid, int64(svc.GetConfig().IntervalSec)*2)
			if needRecheck {
				status = svc.ForceCheck(uid)
				if status == nil {
					writeJSON(w, map[string]interface{}{"code": 404, "msg": "实例未找到"})
					return
				}
			}
			writeJSON(w, map[string]interface{}{"code": 0, "data": status})
			return
		}

		svc.EnsureChecked()
		allStatus := svc.GetAllStatus()
		writeJSON(w, map[string]interface{}{"code": 0, "data": allStatus})

	case "POST":
		var req struct {
			UID string `json:"uid"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "无效请求"})
			return
		}
		if req.UID == "" {
			svc.checkAll()
			allStatus := svc.GetAllStatus()
			writeJSON(w, map[string]interface{}{"code": 0, "data": allStatus})
			return
		}
		status := svc.ForceCheck(req.UID)
		if status == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "实例未找到"})
			return
		}
		writeJSON(w, map[string]interface{}{"code": 0, "data": status})
	}
}

func healthCheckConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	svc := GetHealthService()

	switch r.Method {
	case "GET":
		cfg := svc.GetConfig()
		writeJSON(w, map[string]interface{}{"code": 0, "data": cfg})

	case "PUT":
		var cfg HealthConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "无效配置"})
			return
		}
		svc.UpdateConfig(cfg)
		updated := svc.GetConfig()
		sysLogInfo("HEALTH", fmt.Sprintf("测活配置已更新: 间隔=%ds, 超时=%ds, 告警=%v, 启用=%v",
			updated.IntervalSec, updated.TimeoutSec, updated.AlertEnabled, updated.Enabled))
		writeJSON(w, map[string]interface{}{"code": 0, "data": updated, "msg": "配置已保存"})
	}
}

func healthCheckDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := r.URL.Path
	prefix := "/api/health/check/"
	if !strings.HasPrefix(path, prefix) {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "无效路径"})
		return
	}
	uid := path[len(prefix):]
	if uid == "" {
		writeJSON(w, map[string]interface{}{"code": 400, "msg": "缺少实例标识"})
		return
	}

	svc := GetHealthService()

	if r.Method == "POST" {
		status := svc.ForceCheck(uid)
		if status == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "实例未找到"})
			return
		}
		writeJSON(w, map[string]interface{}{"code": 0, "data": status})
		return
	}

	status := svc.GetStatus(uid)
	needRecheck := status == nil || !status.Online || !svc.IsCacheValid(uid, int64(svc.GetConfig().IntervalSec)*2)
	if needRecheck {
		status = svc.ForceCheck(uid)
		if status == nil {
			writeJSON(w, map[string]interface{}{"code": 404, "msg": "实例未找到"})
			return
		}
	}
	writeJSON(w, map[string]interface{}{"code": 0, "data": status})
}
