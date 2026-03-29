# MailBus 构建指南

本文档说明如何为不同平台构建 MailBus 二进制文件。

## 快速构建

### Linux/macOS

```bash
# 克隆仓库
git clone https://github.com/mailbus/mailbus.git
cd mailbus

# 构建 Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o mailbus-linux-amd64 ./cmd/mailbus

# 构建 Linux ARM64
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o mailbus-linux-arm64 ./cmd/mailbus

# 构建 macOS Intel
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o mailbus-darwin-amd64 ./cmd/mailbus

# 构建 macOS Apple Silicon
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o mailbus-darwin-arm64 ./cmd/mailbus
```

### Windows (PowerShell)

```powershell
# 克隆仓库
git clone https://github.com/mailbus/mailbus.git
cd mailbus

# 构建 Windows AMD64
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o mailbus-windows-amd64.exe ./cmd/mailbus
```

## 生成的二进制文件

构建完成后，会在当前目录生成以下文件：

- `mailbus-linux-amd64` - Linux 64位 (x86_64)
- `mailbus-linux-arm64` - Linux 64位 (ARM64)
- `mailbus-darwin-amd64` - macOS Intel
- `mailbus-darwin-arm64` - macOS Apple Silicon
- `mailbus-windows-amd64.exe` - Windows 64位

## 验证构建

```bash
# 检查二进制文件类型
file mailbus-*

# 查看文件大小
ls -lh mailbus-*

# 测试运行
./mailbus-linux-amd64 version
```

## 生成校验和

```bash
# 为所有二进制文件生成 SHA256 校验和
sha256sum mailbus-* > checksums.txt
```

## 交叉编译说明

Go 支持从任何平台交叉编译到其他平台，因为：

1. **纯 Go 代码**：设置 `CGO_ENABLED=0` 禁用 CGO
2. **静态链接**：生成的二进制文件不依赖系统库
3. **无依赖**：不需要目标平台的 SDK 或工具链

### 交叉编译矩阵

| 构建平台 | Linux | macOS | Windows |
|---------|-------|-------|---------|
| Linux   | ✅     | ✅     | ✅      |
| macOS   | ✅     | ✅     | ✅      |
| Windows | ✅     | ✅     | ✅      |

## 优化建议

### 减小二进制文件大小

```bash
# 使用 upx 压缩 (可选)
upx --best --lzma mailbus-linux-amd64
```

### 启用所有优化

```bash
go build \
  -ldflags="-s -w -buildid=" \
  -trimpath \
  -o mailbus-linux-amd64 \
  ./cmd/mailbus
```

## 故障排除

### 构建失败

```bash
# 确保 Go 版本正确
go version

# 清理缓存
go clean -cache

# 更新依赖
go mod tidy
go mod download
```

### CGO 相关错误

```bash
# 设置环境变量禁用 CGO
export CGO_ENABLED=0

# Windows PowerShell
$env:CGO_ENABLED = "0"
```

## 自动化构建脚本

使用提供的自动安装脚本：

```bash
curl -sSL https://raw.githubusercontent.com/mailbus/mailbus/main/scripts/install.sh | bash
```

或手动下载：

```bash
# Linux
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-linux-amd64

# macOS
curl -O https://github.com/mailbus/mailbus/releases/latest/download/mailbus-darwin-amd64

# Windows
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-windows-amd64.exe
```

## 下一步

- 安装和配置：见 [AGENT_INSTALLATION.md](AGENT_INSTALLATION.md)
- 使用指南：见 [README.md](README.md)
- 贡献指南：见 [CONTRIBUTING.md](CONTRIBUTING.md)
