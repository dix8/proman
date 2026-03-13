# Proman

Version, changelog and announcement management system with an admin dashboard and project-level read-only API.

- 后台管理项目、版本、更新日志和公告
- 对外通过项目级 `/v1` 只读接口提供版本信息、更新日志和公告查询
- 适合为业务项目提供"版本更新内容维护 + 稳定只读查询"的接入能力

## 功能特性

- 单管理员后台，JWT 鉴权
- 项目管理：创建、编辑、删除，独立 API Token
- 版本管理：语义化版本号，草稿/发布状态流转
- 更新日志：按版本维护，支持排序和 Markdown 预览
- 公告管理：发布/撤回、置顶
- 版本对比与日志导出
- 对外只读 API：项目级 Token 鉴权，Redis 限流
- Docker Compose 一键部署

## 快速开始

### Docker 部署（推荐）

```bash
git clone https://github.com/dix8/proman.git
cd proman
cp .env.example .env
```

编辑 `.env`，修改密码和密钥：

```
MYSQL_ROOT_PASSWORD=<强密码>
MYSQL_PASSWORD=<强密码>
JWT_SECRET=<随机长字符串>
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<管理员密码>
```

启动：

```bash
docker compose up -d --build
```

访问 `http://<服务器IP>:8080`，使用上面配置的管理员账号登录。

首次启动会自动执行数据库迁移并创建管理员账户。后续代码未变更时 `docker compose up -d` 即可。

### 服务架构

| 服务 | 说明 |
|---|---|
| proman | Go 后端 + React 前端，统一入口，端口 8080 |
| mysql | MySQL 8.0，仅内部网络，数据持久化到 Docker volume |
| redis | Redis 7，仅内部网络，数据持久化到 Docker volume |

三个服务均配置了健康检查，proman 会等待 MySQL 和 Redis 就绪后才启动。

## 本地开发

依赖：Go 1.22+、Node.js 20+、MySQL 8.0+、Redis 7+

### 后端

```bash
cd server
cp .env.example .env.local   # 按本机环境修改
go mod tidy
go run ./cmd/api
```

启动后自动连接 MySQL、执行 migration、连接 Redis、初始化管理员。

### 前端

```bash
cd web
npm install
npm run dev
```

访问 `http://127.0.0.1:5173`

## 环境变量

项目有三份环境变量样板，用途不同：

| 文件 | 用途 |
|---|---|
| `.env.example` | Docker Compose 正式部署 |
| `server/.env.example` | 后端本地开发 |
| `web/.env.example` | 前端本地开发 |

### 后端变量

| 变量 | 说明 |
|---|---|
| `APP_ENV` | 运行环境，`local` 时启用调试模式 |
| `HTTP_PORT` | 监听端口，默认 `8080` |
| `MYSQL_DSN` | MySQL 连接字符串 |
| `REDIS_ADDR` | Redis 地址 |
| `REDIS_PASSWORD` | Redis 密码 |
| `JWT_SECRET` | JWT 签名密钥 |
| `JWT_EXPIRE_HOURS` | JWT 过期时间，默认 `12` 小时 |
| `ADMIN_USERNAME` | 管理员用户名 |
| `ADMIN_PASSWORD` | 管理员密码 |
| `CORS_ALLOW_ORIGINS` | CORS 允许来源，逗号分隔 |

### 前端变量

| 变量 | 说明 |
|---|---|
| `VITE_API_BASE_URL` | 后端地址，默认 `http://localhost:8080` |
| `VITE_APP_TITLE` | 页面标题，默认 `Proman Admin` |

## API 概览

### 管理后台 `/api`（JWT 鉴权）

- `POST /api/auth/login` — 登录
- `GET/POST /api/projects` — 项目列表 / 创建
- `GET/PUT/DELETE /api/projects/:id` — 项目详情 / 更新 / 删除
- `POST /api/projects/:id/token/refresh` — 刷新项目 Token
- `GET/POST /api/projects/:id/versions` — 版本列表 / 创建
- `GET/PUT/DELETE /api/versions/:id` — 版本详情 / 更新 / 删除
- `PUT /api/versions/:id/publish` — 发布版本
- `GET/POST /api/versions/:id/changelogs` — 日志列表 / 创建
- `PUT/DELETE /api/changelogs/:id` — 更新 / 删除日志
- `PUT /api/versions/:id/changelogs/reorder` — 日志排序
- `GET /api/projects/:id/versions/compare` — 版本对比
- `GET /api/projects/:id/changelogs/export` — 日志导出
- `GET/POST /api/projects/:id/announcements` — 公告列表 / 创建
- `GET/PUT/DELETE /api/announcements/:id` — 公告详情 / 更新 / 删除
- `PUT /api/announcements/:id/publish` — 发布公告
- `PUT /api/announcements/:id/revoke` — 撤回公告
- `POST /api/markdown/preview` — Markdown 预览

### 对外只读 `/v1`（项目 Token 鉴权）

```bash
curl -H "Authorization: Bearer <project_token>" http://localhost:8080/v1/versions
```

| 接口 | 说明 |
|---|---|
| `GET /v1/project` | 项目信息 |
| `GET /v1/versions` | 已发布版本列表 |
| `GET /v1/versions/:version/changelogs` | 指定版本的更新日志 |
| `GET /v1/announcements` | 已发布公告列表 |

Token 通过 Header `Authorization: Bearer <project_token>` 传递，不支持 URL 拼接。Token 仅在创建项目或刷新时展示一次。

### 健康检查

```
GET /healthz
```

## 目录结构

```
proman/
├── .env.example                # Docker Compose 部署环境变量样板
├── docker-compose.yml          # 部署入口
├── server/                     # Go 后端（Gin + GORM）
│   ├── cmd/api/                # 启动入口
│   ├── internal/
│   │   ├── app/                # 应用初始化与路由
│   │   ├── config/             # 配置加载
│   │   ├── handler/            # HTTP 处理器
│   │   ├── middleware/         # 中间件（鉴权、CORS、限流）
│   │   ├── model/              # 数据模型
│   │   ├── repository/         # 数据访问层
│   │   ├── service/            # 业务逻辑层
│   │   ├── pkg/                # 内部工具包
│   │   ├── integration/        # 集成测试
│   │   └── testutil/           # 测试夹具
│   └── migrations/             # 数据库迁移
├── web/                        # React 前端（Vite + Ant Design）
│   ├── src/
│   │   ├── pages/              # 页面组件
│   │   ├── components/         # 可复用组件
│   │   ├── services/           # API 封装
│   │   └── hooks/              # 自定义 Hook
│   └── scripts/                # Smoke 测试脚本
└── README.md
```

## 测试

### 后端

```bash
cd server
go vet ./...
go test ./internal/service -v
go test ./internal/repository -v
go test ./internal/integration -v    # 需要 MySQL + Redis
```

### 前端

```bash
cd web
npm run build
npm run smoke:m6    # 需要前后端均已启动，以及本机浏览器
```

## 已知限制

- 前端构建存在 chunk 体积 warning（> 500 kB）
- 请求体大小限制中间件尚未实现
- 单管理员模式，暂不支持多用户

## License

[Apache-2.0](LICENSE)
