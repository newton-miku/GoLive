.PHONY: all build build-all frontend dist-all clean install run docker-build docker-push

# ========== 配置 ==========
APP      := golive
VERSION  ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD    := $(shell date -u +%Y%m%dT%H%M%SZ 2>/dev/null || date +"%Y%m%dT%H%M%SZ")
LDFLAGS  := -ldflags "-X main.version=$(VERSION) -X main.build=$(BUILD)"
GO       := go
GOBUILD  := $(GO) build $(LDFLAGS)
GOFLAGS  := -trimpath

# 前端工作目录
FRONTEND_DIR := frontend
DIST_DIR    := dist

# 目标平台
PLATFORMS := \
	darwin/amd64   \
	darwin/arm64   \
	linux/amd64    \
	linux/arm64    \
	linux/armv7    \
	windows/amd64  \
	windows/arm64

# ========== 入口 ==========
all: build frontend

# ========== Go 服务端 ==========
build:
	$(GOBUILD) $(GOFLAGS) -o $(APP) ./cmd/server

# 构建单个平台
build-platform:
	@mkdir -p $(DIST_DIR)/$(PLATFORM)
	GOOS=$(word 1,$(subst /, ,$(PLATFORM))) \
	GOARCH=$(word 2,$(subst /, ,$(PLATFORM))) \
	$(GOBUILD) $(GOFLAGS) -o $(DIST_DIR)/$(PLATFORM)/$(APP)$(if $(findstring windows,$(PLATFORM)),.exe) \
		./cmd/server

# 构建所有平台（顺序执行，避免并行竞争）
build-all:
	@echo "==> 构建所有平台..."
	$(foreach PLATFORM,$(PLATFORMS),$(MAKE) build-platform PLATFORM=$(PLATFORM);)

# ========== 前端 ==========
frontend:
	@echo "==> 构建前端..."
	cd $(FRONTEND_DIR) && npm install --silent 2>/dev/null || npm install
	cd $(FRONTEND_DIR) && npm run build

# ========== 打包发行版 ==========
dist-%:
	$(MAKE) build-platform PLATFORM=$*
	@echo "==> 打包 $*..."
	@PLATFORM=$*; \
	TARGET="$(DIST_DIR)/$$PLATFORM"; \
	BIN="$$TARGET/$(APP)$$(if $(findstring windows,$$PLATFORM),.exe)"; \
	mkdir -p "$$TARGET/dist"; \
	cp -r $(FRONTEND_DIR)/dist/. "$$TARGET/dist/"; \
	cp README.md "$$TARGET/" 2>/dev/null || true; \
	ARCHIVE="$(APP)-$$PLATFORM.tar.gz"; \
	cd $(DIST_DIR) && tar -czf "$$ARCHIVE" -C "$$PLATFORM" . && rm -rf "$$PLATFORM"; \
	echo "    打包完成: $$ARCHIVE"

dist-all: frontend
	@echo "==> 打包所有平台发行版..."
	$(foreach PLATFORM,$(PLATFORMS),$(MAKE) dist PLATFORM=$(PLATFORM);)
	@echo ""
	@echo "==> 发行版列表:"
	@ls -lh $(DIST_DIR)/*.tar.gz 2>/dev/null

# ========== Docker ==========
# 构建 amd64 镜像（默认）
docker-build:
	docker build --platform linux/amd64 --tag $(APP):latest --tag $(APP):$(VERSION) .

# 推送到仓库（需先登录）
docker-push:
	docker push $(APP):$(VERSION)
	docker push $(APP):latest

# ========== 开发 ==========
install: build frontend
	cp -r $(FRONTEND_DIR)/dist ./dist-ready

run:
	$(GO) run ./cmd/server

dev: frontend
	$(GO) run ./cmd/server

# ========== 清理 ==========
clean:
	rm -rf $(DIST_DIR)
	rm -f $(APP)
	rm -f $(APP).exe
	rm -rf dist-ready
