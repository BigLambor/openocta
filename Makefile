# OpenOcta 构建
# 构建顺序：前端 -> 复制 embed 资源 -> 后端

.PHONY: ui embed go launcher build clean release snapshot docker run run-ui prepare-wails-icons wails wails-nsis wails-dmg wails-dmg-signed wails-dmg-arm64 wails-dmg-arm64-signed wails-dmg-amd64 wails-dmg-amd64-signed wails-dmg-all wails-dmg-all-signed wails-dev windows-trial

# 构建前端（输出到 src/embed/frontend）
ui:
	cd ui && npm install && npm run build

# 从 git tag 设置版本，复制 config-schema、openocta.json.example、.env 到 embed 目录
embed: ui
	./scripts/set-version.sh
	@test -f src/config-schema.json && cp src/config-schema.json src/embed/ || true
	@test -f src/openocta.json.example && cp src/openocta.json.example src/embed/ || true
	@test -f src/.env && cp src/.env src/embed/ || true

# 构建 Go 二进制（需先执行 embed）
go: embed
	cd src && go build -ldflags "-s -w" -o ../openocta ./cmd/openocta

# 构建桌面启动器（Windows/macOS）
launcher: embed
	cd src && go build -ldflags "-s -w" -o ../openocta-launcher ./cmd/openocta-launcher

# 完整构建（默认）
build: go

# Windows 便携试用包（macOS/Linux 交叉编译，产物在 dist/windows-trial/）
windows-trial:
	./scripts/build-windows-trial.sh

# 清理
clean:
	rm -rf dist dist-trial dist-mac src/embed/frontend src/embed/config-schema.json src/embed/openocta.json.example openocta openocta.exe openocta-launcher openocta-launcher.exe src/build/bin

# GoReleaser 快照构建（不发布）
snapshot:
	@sed -i 's/\r$$//' deploy/scripts/*.sh
	goreleaser release --snapshot --clean --skip=publish

# GoReleaser 正式发布
release:
	@sed -i 's/\r$$//' deploy/scripts/*.sh
	goreleaser release --clean

# 本地 Docker 构建（使用 deploy/Dockerfile 多阶段构建）
docker:
	docker build -f deploy/Dockerfile -t openocta:local .

# 开发：构建并启动 Gateway（端口 18900）
run: build
	./openocta gateway run

# 开发：仅启动前端开发服务器（端口 5173，需另行启动 Gateway）
run-ui:
	cd ui && npm run dev

# 将横版 logo 缩放到最长边 ≤256（ICO 单字节尺寸限制）。有 macOS sips 时自动更新；否则依赖已提交的 png。
imgs/openocta_logo_wails.png: imgs/openocta_logo.png
	@if command -v sips >/dev/null 2>&1; then \
		sips -Z 256 "$(CURDIR)/imgs/openocta_logo.png" --out "$(CURDIR)/imgs/openocta_logo_wails.png"; \
	else \
		test -f "$(CURDIR)/imgs/openocta_logo_wails.png" || (echo "ERROR: 缺少 imgs/openocta_logo_wails.png。请在 macOS 执行: sips -Z 256 imgs/openocta_logo.png --out imgs/openocta_logo_wails.png"; exit 1); \
	fi

# OpenOcta 品牌：同步 PNG 到 Wails 默认路径（wails build 会据 appicon.png 生成 winres/syso），再生成 ICO。
prepare-wails-icons: imgs/openocta_logo_wails.png
	@mkdir -p src/build src/build/windows
	@cp "$(CURDIR)/imgs/openocta_logo_wails.png" "$(CURDIR)/src/build/appicon.png"
	node "$(CURDIR)/scripts/png-to-ico.mjs" "$(CURDIR)/imgs/openocta_logo_wails.png" "$(CURDIR)/src/build/appicon.ico"
	@cp "$(CURDIR)/src/build/appicon.ico" "$(CURDIR)/src/build/windows/icon.ico"

# Wails 桌面应用（单二进制，内嵌 Gateway，端口 18900）
wails: embed prepare-wails-icons
	cd src && wails build -skipbindings

# Windows NSIS 安装器（需在 Windows + NSIS 环境；逻辑见 ./build.sh wails-nsis）
wails-nsis:
	./build.sh wails-nsis

# Wails + 打包 .dmg（macOS），产物在 dist-mac/（避免占用 GoReleaser 的 dist/）
wails-dmg: wails
	./deploy/macos/build-app.sh

# Wails + gon 签名/公证 + 打包 .dmg（macOS）
# 需要环境变量：AC_USERNAME / AC_PASSWORD / AC_TEAM_ID（见 gon-sign.json）
wails-dmg-signed: wails
	OPENOCTA_GON=1 ./deploy/macos/build-app.sh

# 分别打包 Apple Silicon / Intel 的 .dmg（需在 macOS 上执行；需已安装 Wails、Xcode CLI、对应交叉编译依赖）。
# 不能从 Linux 交叉产出 Wails macOS 应用；CI 请使用 macos-latest。
wails-dmg-arm64: embed prepare-wails-icons
	cd src && wails build -skipbindings -platform darwin/arm64
	cd "$(CURDIR)" && SKIP_MAKE_WAILS=1 ARCH=arm64 ./deploy/macos/build-app.sh

wails-dmg-arm64-signed: embed prepare-wails-icons
	cd src && wails build -skipbindings -platform darwin/arm64
	cd "$(CURDIR)" && SKIP_MAKE_WAILS=1 ARCH=arm64 OPENOCTA_GON=1 ./deploy/macos/build-app.sh

wails-dmg-amd64: embed prepare-wails-icons
	cd src && wails build -skipbindings -platform darwin/amd64
	cd "$(CURDIR)" && SKIP_MAKE_WAILS=1 ARCH=amd64 ./deploy/macos/build-app.sh

wails-dmg-amd64-signed: embed prepare-wails-icons
	cd src && wails build -skipbindings -platform darwin/amd64
	cd "$(CURDIR)" && SKIP_MAKE_WAILS=1 ARCH=amd64 OPENOCTA_GON=1 ./deploy/macos/build-app.sh

wails-dmg-all: wails-dmg-arm64 wails-dmg-amd64

wails-dmg-all-signed: wails-dmg-arm64-signed wails-dmg-amd64-signed

# Wails 开发模式（热重载）
wails-dev: embed prepare-wails-icons
	cd src && wails dev
