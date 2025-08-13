# JobTracker

智能求职邮件分析工具 - 使用 MCP 协议简化邮箱登录，通过 LLM 自动分析求职进度。

## 功能特点

- 🚀 **简化登录**：基于 MCP 协议，支持一键浏览器登录多种邮箱
- 🧠 **智能分析**：使用 LLM 自动识别求职相关邮件并提取关键信息
- 📊 **状态跟踪**：自动解析申请状态（已申请、OA、面试、Offer、拒绝等）
- 📋 **数据导出**：生成详细的 CSV 报告和统计信息
- 🔒 **安全可靠**：不存储密码，基于标准协议

## 项目结构

```
JobTracker/
├── cmd/                     # 应用程序入口
│   └── jobtracker/
│       └── main.go         # 主程序
├── internal/                # 内部包（不对外暴露）
│   ├── analyzer/           # LLM 邮件分析器
│   │   └── analyzer.go
│   ├── client/             # MCP 邮件客户端
│   │   └── mcp_client.go
│   ├── config/             # 配置管理
│   │   └── config.go
│   ├── exporter/           # CSV 导出器
│   │   └── csv_exporter.go
│   └── types/              # 数据类型定义
│       └── types.go
├── configs/                 # 配置文件
│   └── config.yaml
├── scripts/                 # 脚本文件
│   └── build.sh           # 构建脚本
├── docs/                   # 文档目录（待扩展）
├── .env                    # 环境变量（本地配置）
├── .gitignore              # Git 忽略文件
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖版本锁定
└── README.md               # 项目说明
```

## 快速开始

### 1. 环境准备

**系统要求**：
- Go 1.22+
- MCP 邮件服务（或使用默认端点）
- LLM API（支持 OpenAI 格式的 API）

### 2. 安装与构建

```bash
# 克隆项目
git clone <your-repo-url>
cd JobTracker

# 使用构建脚本（推荐）
./scripts/build.sh

# 或手动构建
go mod tidy
go build -o bin/jobtracker ./cmd/jobtracker
```

### 3. 配置

**环境变量配置**：
```bash
# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件
vim .env
```

**.env 文件内容**：
```bash
# 用户邮箱地址（必需）
USER_EMAIL=your.email@gmail.com

# LLM API 配置（必需）
DEEPSEEK_API_KEY=sk-your-deepseek-api-key

# 其他 LLM 选项
# OPENAI_API_KEY=sk-your-openai-api-key
# ANTHROPIC_API_KEY=sk-ant-your-anthropic-api-key

# MCP 服务配置（可选）
# MCP_ENDPOINT=http://localhost:8080/mcp
# MCP_API_KEY=your-mcp-api-key
```

**应用配置**：
编辑 `configs/config.yaml`：
```yaml
imap:
  email: ${USER_EMAIL}
  folders:
    - "INBOX"
    - "[Gmail]/Sent Mail"

mcp:
  endpoint: "http://localhost:8080/mcp"
  api_key: ""

fetch:
  start: "2025-01-01"
  end: "2025-12-31"
  max_emails: 200

llm:
  api_base: "https://api.deepseek.com/v1"
  api_key: ${DEEPSEEK_API_KEY}
  model: "deepseek-chat"
  temperature: 0.2
  max_tokens: 2000

export:
  file: "job_applications.csv"
```

### 4. 运行程序

```bash
# 确保环境变量已设置
source .env

# 运行程序
./bin/jobtracker

# 或直接运行（开发模式）
go run ./cmd/jobtracker
```

## 工作流程

```mermaid
graph LR
    A[用户配置] --> B[邮箱登录]
    B --> C[获取邮件]
    C --> D[LLM分析]
    D --> E[状态识别]
    E --> F[CSV导出]
```

### 详细步骤

1. **配置与登录**：读取配置，通过 MCP 启动邮箱登录
2. **邮件获取**：按时间范围和文件夹获取邮件
3. **智能过滤**：使用 LLM 判断邮件是否与求职相关
4. **信息提取**：分析求职邮件，提取关键信息
5. **数据导出**：生成 CSV 报告和统计信息

## 开发说明

### 架构设计

采用分层架构设计：

- **cmd/**: 应用程序入口层
- **internal/**: 业务逻辑层
  - `analyzer`: 负责 LLM 分析逻辑
  - `client`: 负责 MCP 通信
  - `config`: 负责配置管理
  - `exporter`: 负责数据导出
  - `types`: 定义数据结构

### 添加新功能

**添加新的邮件提供商**：
1. 在 `internal/client/` 中扩展 `EmailProvider` 枚举
2. 实现对应的 MCP 接口

**添加新的导出格式**：
1. 在 `internal/exporter/` 中创建新的导出器
2. 在主程序中集成

**自定义 LLM 提示**：
1. 修改 `internal/analyzer/analyzer.go` 中的提示模板
2. 调整状态识别逻辑

### 构建与部署

```bash
# 开发构建
go build ./cmd/jobtracker

# 生产构建（压缩）
go build -ldflags="-s -w" -o jobtracker ./cmd/jobtracker

# 交叉编译
GOOS=linux GOARCH=amd64 go build ./cmd/jobtracker
GOOS=windows GOARCH=amd64 go build ./cmd/jobtracker
```

## 支持的状态

- `APPLIED` - 已申请/简历已投递
- `OA` - 在线测试/笔试邀请  
- `INTERVIEW` - 面试邀请/面试安排
- `OFFER` - 收到录用通知/offer
- `REJECTED` - 被拒绝/未通过
- `WITHDRAWN` - 撤回申请
- `OTHER` - 其他状态

## 常见问题

### Q: 什么是 MCP 协议？
MCP (Model Context Protocol) 是一个标准化协议，简化应用与外部服务的通信。在邮件场景中，它统一了不同邮件提供商的认证和数据获取流程。

### Q: 如何获取 MCP 服务？
目前可以：
1. 使用默认的本地端点（开发中）
2. 自建 MCP 服务
3. 使用第三方 MCP 服务

### Q: 支持哪些 LLM？
支持所有兼容 OpenAI API 格式的 LLM：
- DeepSeek
- OpenAI GPT
- Claude (通过代理)
- 本地模型

### Q: 如何保护隐私？
- 使用标准 OAuth 流程，不存储密码
- 敏感信息通过环境变量管理
- 邮件内容仅用于本地分析
- 可配置本地 LLM 避免数据外传

### Q: 项目依赖有哪些？
核心依赖：
- `gopkg.in/yaml.v3` - YAML 配置解析
- Go 标准库 - HTTP 客户端、CSV 处理等

无外部重依赖，保持项目轻量化。

## 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件 