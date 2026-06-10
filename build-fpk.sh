#!/bin/bash
# 构建飞牛 FPK 原生应用包

set -e

# 路径配置
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/frontend"
SERVER_DIR="$PROJECT_ROOT/server"
FPK_DIR="$PROJECT_ROOT/fpk"

echo "========================================"
echo "  飞牛 FPK 打包工具"
echo "========================================"
echo ""

# 1. 检查必要文件
echo "[1/6] 检查项目结构..."

REQUIRED_FILES=(
    "$FRONTEND_DIR/package.json"
    "$SERVER_DIR/main.go"
    "$FPK_DIR/manifest"
    "$FPK_DIR/fnpack"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "错误: 找不到必要文件 $file"
        exit 1
    fi
done
echo "  ✓ 项目结构检查通过"

# 2. 前端构建
echo ""
echo "[2/6] 构建前端..."

cd "$FRONTEND_DIR"
npm run build
echo "  ✓ 前端构建完成"

# 3. 复制前端到 FPK 目录
echo ""
echo "[3/6] 复制前端文件..."

FPK_UI_DIST="$FPK_DIR/app/ui/dist"
rm -rf "$FPK_UI_DIST"
mkdir -p "$FPK_UI_DIST"

cp -r "$FRONTEND_DIR/dist"/* "$FPK_UI_DIST/"
echo "  ✓ 前端文件已复制"

# 4. 后端交叉编译
echo ""
echo "[4/6] 交叉编译后端 (Linux amd64)..."

cd "$SERVER_DIR"
GOOS=linux GOARCH=amd64 go build -o "$FPK_DIR/app/server" .
echo "  ✓ 后端编译完成"

# 5. 复制图标
echo ""
echo "[5/6] 复制图标..."

IMAGES_DIR="$FPK_DIR/app/ui/images"
mkdir -p "$IMAGES_DIR"

cp "$FPK_DIR/ICON.PNG" "$IMAGES_DIR/icon_64.png"
cp "$FPK_DIR/ICON_256.PNG" "$IMAGES_DIR/icon_256.png"
echo "  ✓ 图标已复制"

# 6. 确保脚本行结束符
echo ""
echo "[5.5/6] 转换脚本为 Unix 格式..."

if command -v dos2unix &> /dev/null; then
    dos2unix "$FPK_DIR/cmd"/* 2>/dev/null || true
fi
echo "  ✓ 脚本格式已转换"

# 7. 构建 FPK
echo ""
echo "[6/6] 构建 FPK 包..."

cd "$FPK_DIR"
rm -f data_manages.fpk

./fnpack build

if [ ! -f data_manages.fpk ]; then
    echo "错误: FPK 构建失败"
    exit 1
fi

SIZE=$(du -h data_manages.fpk | cut -f1)
echo "  ✓ FPK 包构建完成 ($SIZE)"

# 完成
echo ""
echo "========================================"
echo "  FPK 打包成功!"
echo "========================================"
echo ""
echo "输出文件: $FPK_DIR/data_manages.fpk"
echo ""
