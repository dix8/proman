# 项目管理系统

单管理员后台 + 项目级只读开放接口的版本/公告管理系统。

这是一个用于维护具体项目版本、更新日志和公告的后台系统：
- 后台负责管理项目、版本、更新日志和公告
- 对外通过项目级 `/v1` 只读接口提供版本信息、更新日志和公告读取能力
- 适合为具体业务项目提供“版本更新内容维护 + 稳定只读查询”的接入能力

仓库主要包含：
- 后端服务：`server/`
- 前端后台：`web/`
- 正式部署编排：`deploy/`

## 当前进度

当前真实状态以仓库代码与最近验证结果为准：

- 后端 `M5` 已闭环
  - 认证、项目、版本、日志、公告、Markdown 预览、`/v1` 对外接口、导出接口均已完成
- 前端 `M6` 已闭环
  - `T601` 登录与鉴权基座
  - `T602` 后台布局与菜单
  - `T603` 项目模块
  - `T604` 版本与日志页面
  - `T605` 公告页面
  - `T606` 服务端 Markdown 预览接入
  - `T607` 版本对比与导出页面
- `M7` 基线任务 `T701-T704` 已完成
- 当前前端状态可描述为：桌面可用 + 手机端基本可用
- 当前版本已可作为 MVP 正式交付基线

这意味着当前仓库已经具备：
- 从本地环境启动前后端
- 使用后台完成项目、版本、日志、公告主流程
- 在侧边栏进入全局“接口接入”页查看 `/v1` 公开接口接入说明，并复制 URL / curl / fetch 示例
- 在项目详情页查看项目信息、进入版本页、刷新 Token，并跳转到“接口接入”页
- 运行后端测试与前端 smoke 验收
- 后台当前状态可描述为：桌面可用 + 手机端基本可用

## 公开接口调用

当前对外只读接口包括：
- `GET /v1/project`
- `GET /v1/versions`
- `GET /v1/versions/{version}/changelogs`
- `GET /v1/announcements`

这些接口用于：
- 读取当前项目公开基础信息
- 读取已发布版本列表
- 按版本号读取指定已发布版本的更新日志
- 读取已发布公告列表

## 鉴权方式

当前系统有两套鉴权方式，职责不同：
- 管理后台接口：使用管理员 JWT，通过 `Authorization: Bearer <jwt>` 调用 `/api/...`
- 对外公开接口：使用项目级 `project_token`，通过 `Authorization: Bearer <project_token>` 调用 `/v1/...`

注意：
- 对外接口当前只支持通过 Header 传递 `project_token`
- 不支持把 token 直接拼在 URL 上
- 真实 `project_token` 只会在“创建项目”或“刷新 Token”时展示一次

### 最小调用示例

获取已发布版本列表：

```bash
curl -X GET "http://localhost:8080/v1/versions" \
  -H "Authorization: Bearer <project_token>"
```

获取指定版本日志：

```bash
curl -X GET "http://localhost:8080/v1/versions/1.2.3/changelogs" \
  -H "Authorization: Bearer <project_token>"
```

## 接入指引

- 如果要查看完整的长期接入说明，可以在后台侧边栏进入“接口接入”页
- 如果要拿真实 token，需要到项目详情页刷新 Token，并在弹窗出现后立即复制保存

## 目录结构

```text
proman/
├── server/                     # Go + Gin + GORM 后端
│   ├── cmd/api/
│   ├── internal/
│   ├── migrations/
│   └── .env.example
├── web/                        # React + Vite + Ant Design 后台
│   ├── src/
│   ├── scripts/                # smoke / 联调脚本
│   ├── .env.example
│   └── package.json
├── deploy/                     # 正式部署编排
│   ├── docker-compose.yml
│   └── .env.prod.example
└── README.md                   # 本文档
```

### 目录职责边界

- `server/cmd/api/`
  - 后端启动入口。
- `server/internal/service/`
  - 核心业务规则、状态流转与事务边界。
- `server/internal/repository/`
  - 数据访问和持久化细节。
- `server/internal/handler/`
  - HTTP 入参绑定、响应封装和错误返回。
- `server/internal/testutil/`
  - 后端测试夹具，只服务测试，不参与生产逻辑。
- `server/internal/integration/`
  - HTTP 层接口集成测试。
- `server/migrations/`
  - 数据库 migration。
- `web/src/pages/`
  - 页面级 UI 与路由承载。
- `web/src/components/`
  - 页面可复用组件。
- `web/src/services/`
  - 前端接口封装与下载逻辑。
- `web/src/hooks/`
  - 页面间复用的轻量交互逻辑。
- `web/scripts/`
  - smoke / 联调验收脚本。
- `deploy/`
  - 正式部署编排文件。

## 环境准备

本地依赖：
- Go `1.22+`
- Node.js `20+`
- MySQL `8.0+`
- Redis `7+`

默认本地地址：
- MySQL：`127.0.0.1:3306`
- Redis：`127.0.0.1:6379`
- 后端：`http://localhost:8080`
- 前端：`http://127.0.0.1:5173`

本地开发需要自行准备可用的 MySQL / Redis 实例。

## 环境变量

后端样板：
- [server/.env.example](./server/.env.example)

前端样板：
- [web/.env.example](./web/.env.example)

### 后端关键变量

常用变量：
- `APP_ENV`
- `HTTP_PORT`
- `MYSQL_DSN`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`
- `CORS_ALLOW_ORIGINS`

推荐做法：
1. 从 [server/.env.example](./server/.env.example) 复制为 `server/.env.local`
2. 根据本机 MySQL / Redis 配置调整
3. `server/.env.local` 不纳入版本控制

### 前端关键变量

常用变量：
- `VITE_API_BASE_URL`
- `VITE_APP_TITLE`

默认前端并不强依赖 `web/.env.local`，因为：
- API 默认就是 `http://localhost:8080`
- 标题已有默认值

但如果你要换后端地址，建议自行创建 `web/.env.local`。

## 本地启动

### 1. 启动 MySQL / Redis

确认本地依赖已启动：
- MySQL 对应 `MYSQL_DSN`
- Redis 对应 `REDIS_ADDR`

### 2. 启动后端

进入后端目录：

```powershell
cd server
```

首次或依赖有变化时：

```powershell
go mod tidy
```

启动方式一，直接运行源码：

```powershell
go run ./cmd/api
```

启动方式二，先本地构建二进制后再运行：

```powershell
go build -o api.exe ./cmd/api
.\api.exe
```

后端启动后会自动：
- 连接 MySQL
- 执行 `server/migrations` 下 migration
- 连接 Redis
- 当 `users` 表为空时初始化默认管理员

### 3. 启动前端

进入前端目录：

```powershell
cd web
```

安装依赖：

```powershell
npm install
```

启动开发服务器：

```powershell
npm run dev
```

默认访问地址：
- `http://127.0.0.1:5173`

## 正式部署

当前仓库保留的 Compose 编排文件为：
- `deploy/docker-compose.yml`
  - 用于单台服务器上的正式部署基线

正式部署前，先准备生产环境变量文件：

```powershell
copy deploy\.env.prod.example deploy\.env.prod
```

至少需要按实际环境修改：
- MySQL 账号与密码
- `JWT_SECRET`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`

启动正式部署：

```powershell
docker compose --env-file deploy/.env.prod -f deploy/docker-compose.yml up -d --build
```

启动后：
- 正式部署服务收敛为：`proman + mysql + redis`
- `proman` 是唯一应用入口，同时提供前端页面和后端 API
- `proman` 同时处理前端页面访问、`/api`、`/v1` 和 `/healthz`
- 前端页面请求仍默认走相对路径代理；“接口接入”页和 Token 弹窗里的完整 URL 展示默认取当前访问域名
- `mysql`、`redis` 仅在 Compose 内部网络通信，不直接对宿主机开放端口
- 默认通过 `http://<server-host>:8080` 访问

## 开发命令约定

### 后端

进入目录：

```powershell
cd server
```

最小命令约定：

```powershell
gofmt -w ./cmd ./internal
go vet ./...
go test ./internal/service -v
go test ./internal/repository -v
go test ./internal/integration -v
go build ./cmd/api
```

说明：
- `gofmt -w ./cmd ./internal` 是当前 Go 代码格式化入口。
- `go vet ./...` 是当前后端最小静态检查入口。
- `go build ./cmd/api` 是后端构建入口。

### 前端

进入目录：

```powershell
cd web
```

最小命令约定：

```powershell
npm run dev
npm run build
npm run check
npm run format:check
npm run format:write
npm run smoke:m6
```

说明：
- `npm run check` 当前等价于 `npm run build`，作为最小静态检查入口。
- `npm run format:check` / `npm run format:write` 使用 Prettier 处理前端代码和 smoke 脚本格式。
- `npm run smoke:m6` 是当前前端联调验收入口。

## Migration 使用说明

当前 migration 目录：
- [server/migrations/001_init.sql](./server/migrations/001_init.sql)

迁移方式：
- 后端启动时自动执行 migration
- 正式入口在 `server/internal/pkg/migrate/migrate.go`

注意：
- 当前没有单独的 migration CLI
- 如果需要验证最新表结构，最直接方式是启动后端并观察启动日志

## 测试与验收

### 后端构建与测试

进入后端目录：

```powershell
cd server
```

构建：

```powershell
go build ./cmd/api
```

当前已可通过，但前置条件是：
- Go 依赖可从网络或本地模块缓存获取
- MySQL / Redis 对构建本身不是必须条件

单元测试 `T701`：

```powershell
gofmt -w ./cmd ./internal
go vet ./...
go test ./internal/service -v
go test ./internal/repository -v
```

接口集成测试 `T702`：

```powershell
go test ./internal/integration -v
```

后端测试前置条件：
- 需要 MySQL
- `T702` 需要 Redis
- 测试夹具会自动创建隔离测试库和清理 Redis 测试 DB

### 前端构建

进入前端目录：

```powershell
cd web
```

构建：

```powershell
npm run build
```

当前该命令已可通过。

### 前端 smoke / 联调脚本

当前脚本目录：
- [web/scripts](./web/scripts)

可用脚本：
- `smoke-auth.mjs`
- `smoke-projects.mjs`
- `smoke-versions.mjs`
- `smoke-announcements.mjs`
- `smoke-markdown-preview.mjs`
- `smoke-compare-export.mjs`
- `smoke-m6.mjs`
- `smoke-mobile.mjs`

进入前端目录后可直接运行：

```powershell
npm run check
npm run format:check
node scripts/smoke-auth.mjs
node scripts/smoke-projects.mjs
node scripts/smoke-versions.mjs
node scripts/smoke-announcements.mjs
node scripts/smoke-markdown-preview.mjs
node scripts/smoke-compare-export.mjs
node scripts/smoke-m6.mjs
node scripts/smoke-mobile.mjs
```

其中：
- `smoke-m6.mjs` 会顺序执行整套 M6 联调验收脚本，是当前最接近“一键验收”的入口
- `smoke-mobile.mjs` 用于验证后台在手机视口下的基本可用性，结论口径应为“桌面可用 + 手机端基本可用”

前端 smoke 前置条件：
- 后端已启动
- 前端开发服务器已启动
- 本机可用 Chromium / Edge 可执行文件

浏览器路径说明：
- 各脚本默认使用：
  - `C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe`
- 如果本机路径不同，可通过环境变量覆盖：

```powershell
$env:BROWSER_EXECUTABLE_PATH='C:/path/to/your/browser.exe'
node scripts/smoke-m6.mjs
```

## 验收执行建议

如果你是第一次运行这个仓库，建议按下面顺序做一轮完整验证：

1. 起 MySQL / Redis
2. 启动后端
3. 启动前端
4. 跑前端构建

```powershell
cd web
npm run build
```

5. 跑后端集成测试

```powershell
cd server
go test ./internal/integration -v
```

6. 跑前端汇总联调脚本

```powershell
cd web
npm run smoke:m6
```

7. 如需补充验证手机端基本可用性，再运行：

```powershell
cd web
node scripts/smoke-mobile.mjs
```

如果这些步骤都通过，基本可以认为：
- 后端核心接口链路可用
- 前端 M6 主流程可用
- 后台当前状态为：桌面可用 + 手机端基本可用
- 当前仓库已具备继续使用和维护的条件

## 已知限制 / 非阻塞技术债

- 前端构建仍有大包体积 warning，`npm run build` 仍会提示 `chunk > 500 kB`
- 前端当前直连 `http://localhost:8080`，后端 CORS 默认主要放行 `5173`；若前端开发端口改成 `5174+`，需要同步调整 `CORS_ALLOW_ORIGINS`
- 包括 `smoke-mobile.mjs` 在内的 smoke 脚本依赖浏览器可执行文件，路径不对时需要手动通过环境变量 `BROWSER_EXECUTABLE_PATH` 覆盖
- 请求体大小限制中间件尚未实现，规格中的 `1MB` 建议限制仍待补齐
- 多用户 / 反向越权测试覆盖仍可继续加强
- CI / pre-commit / lint 工具链仍可继续补强
