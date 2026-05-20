BINARY_NAME := roost
MODULE_NAME := github.com/phpgao/roost
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMODTIDY := $(GOCMD) mod tidy
GOLINT := golangci-lint

# 版本信息
VERSION := $(shell git describe --always --tags --dirty=-dirty 2>/dev/null || echo "v0.0.0")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -ldflags "-X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'" -trimpath

# 输出目录
OUTPUT_DIR := bin

.PHONY: help build run install test lint clean fmt vet all

help: ## 显示帮助信息
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 编译二进制
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

run: build ## 编译并运行
	@echo "Running $(BINARY_NAME)..."
	@$(OUTPUT_DIR)/$(BINARY_NAME)

install: ## 安装到 GOBIN/GOPATH/bin，输出安装路径
	@echo "Installing $(BINARY_NAME)..."
	@$(GOCMD) install $(LDFLAGS) .
	@INSTALL_DIR=$$(go env GOBIN 2>/dev/null); \
	if [ -z "$$INSTALL_DIR" ]; then \
		INSTALL_DIR=$$(go env GOPATH)/bin; \
	fi; \
	echo "Installed to $$INSTALL_DIR/$(BINARY_NAME)"

test: ## 运行测试
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete."

lint: ## 运行 golangci-lint
	@echo "Running linter..."
	$(GOLINT) run ./...
	@echo "Lint complete."

clean: ## 清理构建产物
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f coverage.out
	$(GOCLEAN)
	@echo "Clean complete."

fmt: ## 格式化代码
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "Format complete."

vet: ## 运行 go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...
	@echo "Vet complete."

tidy: ## 整理 go.mod 依赖
	@echo "Tidying modules..."
	$(GOMODTIDY)
	@echo "Tidy complete."

all: fmt vet test build ## 执行格式化、检查、测试、编译
