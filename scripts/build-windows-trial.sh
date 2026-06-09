#!/usr/bin/env bash
# Windows 便携试用包：在 macOS/Linux 上交叉编译 openocta.exe 并打 zip
# 用法: ./scripts/build-windows-trial.sh
# 产物: dist/windows-trial/OpenOcta-Trial-Windows-v<version>.zip

set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-}"
if [[ -z "$VERSION" ]]; then
  VERSION=$(git describe --tags --always 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
fi
VERSION="${VERSION#v}"

PKG_NAME="OpenOcta-Trial-Windows-v${VERSION}"
OUT_ROOT="dist/windows-trial"
OUT_DIR="${OUT_ROOT}/${PKG_NAME}"
ZIP_PATH="${OUT_ROOT}/${PKG_NAME}.zip"

# Windows CMD 需要 CRLF；bat 文件仅用 ASCII，避免 GBK/UTF-8 乱码
to_crlf() {
  local f="$1"
  sed 's/\r$//' "$f" | awk '{ sub(/\r$/,""); printf "%s\r\n", $0 }' > "${f}.tmp"
  mv "${f}.tmp" "$f"
}

echo "==> 版本: ${VERSION}"
echo "==> 嵌入前端与配置资源..."
make embed

echo "==> 交叉编译 Windows amd64..."
rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"
(
  cd src
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -ldflags "-s -w" \
    -trimpath \
    -o "../${OUT_DIR}/openocta.bin" ./cmd/openocta
)
rm -f "${OUT_DIR}/openocta.exe"

echo "==> 复制启动脚本与说明（ASCII 文件名 + CRLF）..."
# 使用 .bin 分发，避免邮件/网盘剥离 .exe；start.bat 首次运行时会复制为 openocta.exe
cp deploy/windows/start.bat deploy/windows/stop.bat "${OUT_DIR}/"
to_crlf "${OUT_DIR}/start.bat"
to_crlf "${OUT_DIR}/stop.bat"
# README 使用 UTF-8 BOM，方便 Windows 记事本正确显示中文
cp deploy/windows/00-EXTRACT-FIRST.txt "${OUT_DIR}/"
printf '\xEF\xBB\xBF' > "${OUT_DIR}/README.txt"
cat deploy/windows/README.txt >> "${OUT_DIR}/README.txt"

echo "==> 打包 zip..."
rm -f "${ZIP_PATH}"
(
  cd "${OUT_ROOT}"
  zip -rq "${PKG_NAME}.zip" "${PKG_NAME}"
)

BIN_SIZE=$(du -h "${OUT_DIR}/openocta.bin" | cut -f1)
ZIP_SIZE=$(du -h "${ZIP_PATH}" | cut -f1)

echo ""
echo "==> 完成"
echo "    目录: ${OUT_DIR}/"
echo "    压缩包: ${ZIP_PATH} (${ZIP_SIZE})"
echo "    程序文件: openocta.bin (${BIN_SIZE})，start.bat 首次运行复制为 openocta.exe"
echo "    启动: start.bat / 停止: stop.bat"
echo ""
echo "交付给客户: 发送 ${ZIP_PATH}"
