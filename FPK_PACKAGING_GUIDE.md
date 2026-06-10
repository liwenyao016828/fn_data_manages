# 飞牛 FPK 打包流程文档

> 适用于飞牛 fnOS 原生应用打包

## 项目结构

```
项目根目录/
├── frontend/          # Vue 前端
├── server/            # Go 后端
└── fpk/               # FPK 打包目录
    ├── manifest
    ├── ICON.PNG       # 64x64
    ├── ICON_256.PNG   # 256x256
    ├── LICENSE
    ├── config/
    │   ├── privilege
    │   └── resource
    ├── cmd/
    │   ├── main
    │   ├── install_init
    │   ├── install_callback
    │   ├── upgrade_init
    │   ├── upgrade_callback
    │   ├── uninstall_init
    │   └── uninstall_callback
    ├── wizard/
    │   ├── install
    │   └── uninstall
    └── app/
        ├── server     # Linux amd64 二进制
        └── ui/
            ├── config
            ├── images/
            │   ├── icon_64.png
            │   └── icon_256.png
            └── dist/  # 前端构建产物
```

## 关键配置要点

### manifest
```
appname=data_manages
version=1.0.0
display_name=数据库管理工具
desc="""数据库管理工具是一款功能强大的MySQL和Redis数据库管理应用，支持数据浏览、SQL执行、备份恢复、实时监控等功能。"""
platform=x86
source=thirdparty
maintainer=DataManages Team
maintainer_url=https://github.com
distributor=DataManages
distributor_url=https://github.com
desktop_uidir=ui
desktop_applaunchname=data_manages.APPLICATION
service_port=8080
checkport=true
```

### config/privilege
```json
{
    "defaults":
    {
        "run-as": "package"
    },
    "username": "data_manages",
    "groupname": "data_manages"
}
```

### config/resource
```json
{
    "data-share":
    {
        "shares":
        [
            {
                "name": "data_manages",
                "permission":
                {
                    "rw":
                    [
                        "data_manages"
                    ]
                }
            }
        ]
    }
}
```

### app/ui/config
```json
{
    ".url": {
        "data_manages.APPLICATION": {
            "title": "数据库管理工具",
            "icon": "images/icon_{0}.png",
            "type": "iframe",
            "protocol": "http",
            "port": "8080",
            "url": "/",
            "allUsers": true
        }
    }
}
```

### cmd/main (Unix 行结束符)
```bash
#!/bin/sh

APP_DIR="$1"
PKG_DIR="$2"
CMD="$3"

PID_FILE="${PKG_DIR}/run/service.pid"
LOG_FILE="${PKG_DIR}/log/service.log"

case "$CMD" in
start)
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE" 2>/dev/null)
        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            echo "service is already running with pid $PID"
            exit 0
        fi
        rm -f "$PID_FILE"
    fi

    mkdir -p "${PKG_DIR}/log"
    mkdir -p "${PKG_DIR}/run"

    export TRIM_PKGVAR="$PKG_DIR"
    export TRIM_PKGSHARE="${APP_DIR}"

    "$APP_DIR/app/server" > "$LOG_FILE" 2>&1 &

    PID=$!
    echo "$PID" > "$PID_FILE"
    echo "service started with pid $PID"
    ;;
stop)
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE" 2>/dev/null)
        if [ -n "$PID" ]; then
            kill -TERM "$PID" 2>/dev/null
            sleep 2
            if kill -0 "$PID" 2>/dev/null; then
                kill -9 "$PID" 2>/dev/null
            fi
        fi
        rm -f "$PID_FILE"
        echo "service stopped"
    fi
    ;;
status)
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE" 2>/dev/null)
        if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
            echo "running"
            exit 0
        fi
        rm -f "$PID_FILE"
    fi
    echo "stopped"
    exit 1
    ;;
*)
    echo "Usage: $0 app_dir pkg_dir {start|stop|status}"
    exit 1
    ;;
esac

exit 0
```

## 完整打包步骤

### 1. 前端构建
```bash
cd frontend
npm run build
```

### 2. 复制前端到 FPK 目录
```bash
# Windows PowerShell
Copy-Item dist\* ..\fpk\app\ui\dist\ -Recurse -Force

# Linux/Mac
cp -r dist/* ../fpk/app/ui/dist/
```

### 3. 后端交叉编译 (Linux amd64)
```bash
cd server
GOOS=linux GOARCH=amd64 go build -o ../fpk/app/server .
```

### 4. 复制图标
```bash
# Windows PowerShell
Copy-Item ../fpk/ICON.PNG ../fpk/app/ui/images/icon_64.png -Force
Copy-Item ../fpk/ICON_256.PNG ../fpk/app/ui/images/icon_256.png -Force

# Linux/Mac
cp ../fpk/ICON.PNG ../fpk/app/ui/images/icon_64.png
cp ../fpk/ICON_256.PNG ../fpk/app/ui/images/icon_256.png
```

### 5. 确保 cmd 脚本为 Unix 行结束符
```bash
# Linux/Mac
dos2unix fpk/cmd/*
```

### 6. 构建 FPK
```bash
cd fpk
fnpack build
```

## 输出文件

`data_manages.fpk` - 完整的飞牛原生应用包
