# MailBus 架构设计文档

## 项目概述

**MailBus** 是一个基于标准邮件协议（SMTP/IMAP）的智能体消息总线，让任何脚本、程序或自动化组件都能像发邮件一样简单地进行异步通信。

### 核心价值
- **零集成成本**：任何能发送邮件的程序都能接入
- **语言无关**：不依赖特定编程语言或运行时
- **天然异步**：利用邮件协议的异步特性
- **简单可靠**：基于成熟的邮件基础设施

---

## 技术架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                         MailBus CLI                         │
├──────────────┬──────────────┬──────────────┬───────────────┤
│   send 命令   │   poll 命令  │   list 命令  │   mark 命令   │
└──────┬───────┴──────┬───────┴──────┬───────┴───────┬───────┘
       │              │              │               │
┌──────▼──────────────▼──────────────▼───────────────▼───────┐
│                      Core Layer                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ Protocol │  │ Handler  │  │  Filter  │  │  Config  │  │
│  │  Engine  │  │  Engine  │  │  Engine  │  │ Manager  │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
└──────────────────────────┬────────────────────────────────┘
                           │
┌──────────────────────────▼────────────────────────────────┐
│                   Transport Layer                          │
│              ┌─────────┴─────────┐                          │
│              │  Adapter Interface │                          │
│              └─────────┬─────────┘                          │
│       ┌──────────────┼──────────────┐                      │
│       ▼              ▼              ▼                      │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐                  │
│  │  SMTP   │   │  IMAP   │   │ Webhook │ (future)          │
│  │ Adapter │   │ Adapter │   │ Adapter │                    │
│  └─────────┘   └─────────┘   └─────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

### 技术栈

| 层级 | 技术选择 | 说明 |
|------|---------|------|
| 语言 | Go 1.21+ | 单文件二进制，跨平台，高性能 |
| CLI框架 | cobra | 成熟的命令行框架 |
| 配置管理 | viper | 支持多种格式，环境变量 |
| 邮件协议 | go-smtp + go-imap | 原生Go实现，无CGO依赖 |
| 测试 | testify + gomock | 标准测试工具 |
| 构建 | goreleaser | 自动化多平台构建 |

---

## 模块设计

### 1. 核心模块 (pkg/core)

#### 1.1 Message（消息模型）
```go
type Message struct {
    ID          string            // 消息唯一ID
    From        string            // 发件人
    To          []string          // 收件人列表
    Subject     string            // 主题（支持标签过滤）
    Body        string            // 正文（JSON格式）
    ContentType string            // 内容类型
    Headers     map[string]string // 扩展头
    Attachments []Attachment      // 附件列表
    Timestamp   time.Time         // 时间戳
    Flags       []string          // 标志（已读、已处理等）
}

type Attachment struct {
    Filename    string
    ContentType string
    Size        int64
    ContentID   string
}
```

#### 1.2 Filter（过滤器引擎）
```go
type Filter struct {
    SubjectPattern string         // 主题正则表达式
    FromPattern    string         // 发件人模式
    MinDate        time.Time      // 最早时间
    MaxDate        time.Time      // 最晚时间
    Flags          []string       // 标志过滤
    Custom         func(*Message) bool // 自定义过滤函数
}

type FilterEngine interface {
    Match(msg *Message) bool
    ParseSubjectTags(subject string) []string  // 解析 [task.research] 这样的标签
}
```

#### 1.3 Handler（处理器引擎）
```go
type Handler interface {
    CanHandle(msg *Message) bool
    Handle(ctx context.Context, msg *Message) (*HandlerResult, error)
}

type HandlerResult struct {
    Success   bool
    Message   string
    ReplyTo   string   // 可选：回复地址
    ReplyBody string   // 可选：回复内容
    Action    string   // "mark_read", "delete", "move"
}

// 内置处理器
type ExecHandler struct {
    Command string
    Args    []string
}

type ScriptHandler struct {
    ScriptPath string
    Timeout    time.Duration
}
```

### 2. 协议适配层 (pkg/adapter)

#### 2.1 Adapter接口
```go
type Adapter interface {
    Connect(ctx context.Context, cfg *ConnectionConfig) error
    Close() error
    Send(ctx context.Context, msg *Message) error
    Receive(ctx context.Context, filter *Filter) ([]*Message, error)
    Mark(ctx context.Context, msgID string, action string) error
}

type ConnectionConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    UseTLS   bool
}
```

#### 2.2 SMTP适配器
```go
type SMTPAdapter struct {
    client *smtp.Client
    config *ConnectionConfig
}
```

#### 2.3 IMAP适配器
```go
type IMAPAdapter struct {
    client *imap.Client
    config *ConnectionConfig
}
```

### 3. 配置管理 (pkg/config)

```go
type Config struct {
    DefaultAccount string                    `yaml:"default_account"`
    Accounts       map[string]*AccountConfig `yaml:"accounts"`
    Global         *GlobalConfig             `yaml:"global"`
}

type AccountConfig struct {
    IMAP struct {
        Host   string `yaml:"host"`
        Port   int    `yaml:"port"`
        UseTLS bool   `yaml:"use_tls"`
    } `yaml:"imap"`
    SMTP struct {
        Host   string `yaml:"host"`
        Port   int    `yaml:"port"`
        UseTLS bool   `yaml:"use_tls"`
    } `yaml:"smtp"`
    Username      string `yaml:"username"`
    PasswordEnv   string `yaml:"password_env"`  // 环境变量名
    From          string `yaml:"from"`          // 默认发件人
}

type GlobalConfig struct {
    PollInterval   time.Duration `yaml:"poll_interval"`   // 轮询间隔
    BatchSize      int            `yaml:"batch_size"`      // 批量大小
    Timeout        time.Duration  `yaml:"timeout"`         // 超时时间
    MaxRetries     int            `yaml:"max_retries"`     // 最大重试
    Verbose        bool           `yaml:"verbose"`         // 详细输出
}
```

---

## 命令设计

### send - 发送消息
```bash
mailbus send [flags]

Flags:
  -t, --to string[]           收件人地址（必需）
  -s, --subject string        主题（必需）
  -b, --body string           正文内容
  -f, --file string           从文件读取正文
  -a, --attach string[]       附件路径
  -H, --header stringToString 自定义头（key=value）
  -A, --account string        使用指定账户
      --format string         正文格式（text/json, 默认json）
      --priority string       优先级（high/normal/low）
      --ttl int               消息存活时间（秒）

示例：
# 发送简单消息
mailbus send --to agent@example.com --subject "[task] Hello" --body '{"msg":"world"}'

# 发送带附件的消息
mailbus send --to agent@example.com --subject "[data] Report" \
  --body '{"type":"monthly"}' --attach report.pdf

# 发送带自定义头的消息
mailbus send --to agent@example.com --subject "[alert] Error" \
  --body '{"error":"connection failed"}' \
  --header "X-Priority=1" --header "X-Timeout=300"
```

### poll - 轮询并处理消息
```bash
mailbus poll [flags]

Flags:
  -s, --subject string       主题过滤器（支持正则）
  -f, --from string          发件人过滤器
  -u, --unread               只处理未读消息
  -n, --once                 处理一条后退出
  -c, --continuous           持续轮询模式
  -i, --interval int         轮询间隔（秒）
  -H, --handler string       处理器命令
      --handler-timeout int  处理器超时（秒）
      --on-error string      错误处理策略（continue/stop/retry）
      --reply-with-result    将处理结果作为回复发送
      --mark-after string    处理后的标记动作（read/delete/none）
  -A, --account string       使用指定账户
  -F, --folder string        IMAP文件夹（默认INBOX）

示例：
# 列出未读消息
mailbus poll --unread --subject "\[task\]"

# 执行处理器
mailbus poll --subject "\[task\]" --handler "./process.sh" --once

# 持续监控模式
mailbus poll --subject "\[alert\]" --handler "./alert.sh" --continuous --interval 30

# 带结果回复
mailbus poll --subject "\[query\]" --handler "python query.py" \
  --reply-with-result --mark-after read
```

### list - 列出消息
```bash
mailbus list [flags]

Flags:
  -u, --unread             只显示未读
  -s, --subject string     主题过滤器
  -f, --from string        发件人过滤器
  -n, --limit int          限制数量（默认20）
  -o, --offset int         偏移量
  -F, --format string      输出格式（table/json/compact）
  -A, --account string     使用指定账户

示例：
mailbus list --unread --subject "\[task\]" --format json
```

### mark - 标记消息
```bash
mailbus mark [flags]

Flags:
  -i, --id string          消息ID（必需）
  -a, --action string      动作：read/unread/delete/move（必需）
  -F, --folder string      移动目标文件夹（move时必需）
  -A, --account string     使用指定账户

示例：
mailbus mark --id "123@localhost" --action read
mailbus mark --id "123@localhost" --action move --folder "Processed"
```

### config - 配置管理
```bash
mailbus config [subcommand]

Subcommands:
  init      初始化配置文件
  validate  验证配置
  list      列出所有账户
  add       添加新账户
  remove    删除账户
  set       设置全局选项

示例：
mailbus config init
mailbus config add --name work --username agent@company.com
mailbus config set --poll-interval 60
```

---

## 目录结构

```
mailbus/
├── cmd/
│   ├── mailbus/           # 主命令入口
│   │   └── main.go
│   ├── send/              # send命令
│   │   └── send.go
│   ├── poll/              # poll命令
│   │   └── poll.go
│   ├── list/              # list命令
│   │   └── list.go
│   ├── mark/              # mark命令
│   │   └── mark.go
│   └── config/            # config命令
│       └── config.go
├── pkg/
│   ├── core/              # 核心逻辑
│   │   ├── message.go
│   │   ├── filter.go
│   │   ├── handler.go
│   │   └── engine.go
│   ├── adapter/           # 协议适配器
│   │   ├── adapter.go
│   │   ├── smtp.go
│   │   └── imap.go
│   ├── config/            # 配置管理
│   │   ├── config.go
│   │   └── validator.go
│   └── util/              # 工具函数
│       ├── retry.go
│       └── logger.go
├── internal/              # 内部工具
│   ├── version/
│   └── testutil/
├── docs/                  # 文档
│   ├── getting-started.md
│   ├── commands.md
│   ├── examples.md
│   └── architecture.md
├── examples/              # 示例
│   ├── handlers/
│   │   ├── simple.sh
│   │   └── advanced.py
│   └── workflows/
│       └── github-integration.yml
├── test/                  # 测试
│   ├── integration/
│   └── e2e/
├── .goreleaser.yml        # 构建配置
├── LICENSE                # Apache 2.0
├── README.md
├── go.mod
└── go.sum
```

---

## 消息协议规范

### 标准消息头扩展

```email
Subject: [tag.category] 人类可读标题
X-MailBus-Version: 1.0
X-MailBus-Message-Type: request
X-MailBus-Request-ID: req_1234567890
X-MailBus-Timestamp: 2024-03-29T10:30:00Z
X-MailBus-TTL: 3600
X-MailBus-Priority: normal
X-MailBus-Require-Ack: false
```

### 消息正文格式

```json
{
  "metadata": {
    "sender_id": "agent://example/researcher/v1",
    "receiver_id": "agent://example/processor/v1",
    "correlation_id": "corr_abc123",
    "timestamp": "2024-03-29T10:30:00Z",
    "tags": ["task", "research"]
  },
  "payload": {
    "type": "function_call",
    "data": {
      "query": "AI trends 2024",
      "options": {
        "max_results": 10
      }
    }
  },
  "expectation": {
    "response_format": "json",
    "timeout": 300
  }
}
```

---

## 安全考虑

### 1. 凭证管理
- 密码通过环境变量传递，不在配置文件明文存储
- 支持 OAuth2.0（后期增强）
- 支持密钥环集成（系统keychain）

### 2. 传输安全
- 默认强制 TLS/STARTTLS
- 支持证书指纹验证
- 禁用不安全的密码套件

### 3. 处理器安全
- 沙箱执行（可选）
- 超时限制
- 资源限制（CPU、内存）
- 输入验证和清理

---

## 测试策略

### 单元测试
- 每个模块 >80% 覆盖率
- 表格驱动测试
- Mock外部依赖

### 集成测试
- 使用测试邮件服务器（如 inbucket）
- 测试真实SMTP/IMAP交互

### E2E测试
- 完整工作流测试
- 跨平台测试（Linux、macOS、Windows）

---

## 性能指标

| 指标 | 目标值 |
|------|--------|
| 启动时间 | <100ms |
| send 命令延迟 | <500ms |
| poll 命令延迟 | <1s |
| 内存占用 | <50MB |
| 二进制大小 | <15MB |

---

## 扩展点（为未来版本预留）

1. **Webhook适配器**：实时推送支持
2. **插件系统**：自定义处理器类型
3. **消息存储**：本地缓存和索引
4. **规则引擎**：复杂路由规则
5. **监控指标**：Prometheus导出器
6. **配置中心**：远程配置管理
7. **多协议支持**：AMQP、MQTT等

---

## 开发计划

### Phase 1: 核心功能 (Week 1-2)
- [x] 项目初始化
- [ ] 核心数据结构
- [ ] SMTP适配器
- [ ] IMAP适配器
- [ ] 基础配置管理

### Phase 2: 命令实现 (Week 3-4)
- [ ] send命令
- [ ] poll命令
- [ ] list命令
- [ ] mark命令
- [ ] config命令

### Phase 3: 处理器系统 (Week 5)
- [ ] Exec处理器
- [ ] Script处理器
- [ ] 过滤器引擎
- [ ] 错误处理策略

### Phase 4: 文档与示例 (Week 6)
- [ ] README
- [ ] 快速开始指南
- [ ] 命令参考文档
- [ ] 示例脚本

### Phase 5: 测试与优化 (Week 7-8)
- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能优化
- [ ] 跨平台测试

---

## License

Apache License 2.0

保留权利说明：
- 保留"MailBus"商标使用权
- 保留云服务商业化权利
- 保留企业级功能（如Webhook、规则引擎）的商业化权利
