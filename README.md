# AR-MusicAPI

多音源音乐解析 API + 可视化管理后台。

对外提供统一的点歌接口，支持 QQ 音乐、网易云音乐、酷狗音乐、汽水音乐；内置管理后台，可管理 Cookie 池、查看调用日志与运行数据，并直接在后台播放解析结果（含逐行滚动歌词）。

> 仓库：<https://github.com/iyiyus/AR-MusicAPI>
>
> 部署：[宝塔部署文档（DEPLOY.md）](./DEPLOY.md) ｜ [下载部署包（Releases）](https://github.com/iyiyus/AR-MusicAPI/releases/latest)

---

## 功能特性

- **多音源解析**：QQ 音乐、网易云音乐、酷狗音乐、汽水音乐（按歌曲 ID / 分享链接）
- **精准匹配**：支持传入歌手名，优先命中原唱，自动避开翻唱与同名 DJ 版
- **字段完整**：播放直链、封面、纯文本歌词、带时间轴的 LRC 歌词、歌曲时长
- **可视化管理后台**：运行概览、Cookie 池、调用日志、系统设置
- **在线试听**：日志列表可直接播放，封面可点击放大，歌词随播放进度逐行滚动并居中
- **鉴权与安全**：JWT 登录、密码加密存储、公开接口可开关、Cookie 加密保存
- **暗色模式 / 响应式布局**：基于 Art Design Pro 模板，桌面与移动端均可使用

## 技术栈

| 端 | 技术 |
| --- | --- |
| 后端 | Go（标准库 `net/http`）、MySQL 8、JWT、bcrypt |
| 前端 | Vue 3、TypeScript、Vite、Element Plus、Tailwind CSS、Pinia |

## 目录结构

```
music/
├── BackEnd/
│   ├── db/schema.sql          # 数据库结构
│   ├── internal/
│   │   ├── api/               # HTTP 接口（认证、仪表盘、Cookie、日志、设置、点歌）
│   │   ├── config/            # 配置加载
│   │   ├── httpx/             # 统一响应与请求参数
│   │   ├── jwtx/              # JWT 签发与校验
│   │   ├── music/             # 各音源解析实现
│   │   └── store/             # MySQL 数据访问
│   ├── tools/migrate/         # 数据库升级脚本（一次性执行）
│   ├── config.json            # 运行配置（不入库）
│   └── main.go
└── FrontEnd/
    ├── src/views/home/        # 点歌首页
    ├── src/views/music/       # 管理后台（概览、Cookie、日志、设置）
    ├── src/api/               # 接口封装
    └── src/utils/             # 工具函数（歌词解析等）
```

## 生产部署

服务器不用装 Go / Node，直接用已编译好的部署包：

1. 下载 [`ar-musicapi-linux-amd64.zip`](https://github.com/iyiyus/AR-MusicAPI/releases/latest/download/ar-musicapi-linux-amd64.zip)
2. 上传到 `/www/wwwroot/<你的域名>/` 解压，得到 `api/`、`db/`、`web/` 三个目录
3. 按 **[宝塔部署文档 DEPLOY.md](./DEPLOY.md)** 操作：建库导表 → 改 `config.json` → 加守护进程 → 建站点 → 整段替换 Nginx 配置 → 登录后台

单域名即可跑通，`/api` 与 `/uploads` 由 Nginx 反代到 `127.0.0.1:8080`；站点根目录指向 `web/`，`config.json` 不在网站根目录内。

## 快速开始

以下为源码开发/自行构建的方式。

### 环境要求

- Go 1.26+
- MySQL 8.0+
- Node.js 20.19+、pnpm 8+

### 1. 初始化数据库

```sql
CREATE DATABASE IF NOT EXISTS `music_admin`
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;
```

```bash
mysql -u root -p music_admin < BackEnd/db/schema.sql
```

> 从旧版本升级（补齐日志表字段）：`cd BackEnd && go run ./tools/migrate`

### 2. 启动后端

```bash
cd BackEnd
cp config.example.json config.json   # 按需修改数据库账号密码
go mod download
go run .
```

启动后监听 `http://127.0.0.1:8080`，可用 `GET /api/health` 检查状态。

首次启动会使用配置中的 `admin` 账号初始化管理员，登录后请及时修改密码。

### 3. 启动前端

```bash
cd FrontEnd
pnpm install
pnpm dev
```

- 开发环境通过 Vite 代理把 `/api` 转发到后端（见 `.env.development` 中的 `VITE_API_PROXY_URL`）
- 生产构建：`pnpm build`，产物在 `dist/`，将 `/api` 反向代理到后端即可

### 前端页面

| 路径 | 说明 |
| --- | --- |
| `/` | 点歌首页（在线搜索试听） |
| `/dashboard` | 运行概览（调用量、成功率、趋势、最近调用） |
| `/cookie` | Cookie 池管理（按音源维护登录 Cookie） |
| `/log` | 调用日志（筛选、详情、播放） |
| `/settings` | 系统设置（站点信息、默认音源、账号密码） |

## 配置说明

`BackEnd/config.json`：

| 字段 | 说明 |
| --- | --- |
| `server.port` | 后端监听端口，默认 `8080` |
| `server.mode` | 运行模式，`debug` / `release` |
| `database.host/port/user/password/name` | MySQL 连接信息 |
| `database.charset` | 字符集，建议 `utf8mb4` |
| `jwt.secret` | JWT 签名密钥，**部署时务必更换** |
| `jwt.expire_hours` | 访问令牌有效期（小时） |
| `jwt.refresh_hours` | 刷新令牌有效期（小时） |
| `admin.default_username` | 初始管理员账号 |
| `admin.default_password` | 初始管理员密码 |

## 接口文档

### 公开接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` | 健康检查 |
| GET | `/api/site/info` | 站点名称、图标等公开信息 |
| GET | `/api/music/sources` | 已支持的音乐源列表 |
| GET | `/api/music/search` | 统一点歌接口 |

`GET /api/music/search` 参数：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `name` | 是（汽水音乐除外） | 歌曲名，兼容 `keyword` / `s` |
| `singer` | 否 | 歌手名，用于优先命中原唱，兼容 `artist` |
| `source` | 否 | `qqmusic` / `wyy` / `kugou` / `qishui`，缺省取系统设置中的默认音源 |
| `id` | 否 | 歌曲 ID 或分享链接（汽水音乐必填） |

示例：

```bash
curl "http://127.0.0.1:8080/api/music/search?name=晴天&singer=周杰伦&source=wyy"
```

```json
{
  "code": 200,
  "source": "wyy",
  "song_id": "186016",
  "title": "晴天",
  "singer": "周杰伦",
  "cover": "http://p1.music.126.net/...jpg",
  "link": "https://music.163.com/song?id=186016",
  "music_url": "http://m801.music.126.net/....mp3",
  "lyric": "晴天 - 周杰伦\n词：周杰伦...",
  "lyric_lrc": "[00:00.00]晴天 - 周杰伦\n[00:29.26]故事的小黄花...",
  "duration_ms": 620,
  "song_duration_ms": 269000
}
```

> `code` 为 `200` 表示成功；`msg` 非空时通常为提示信息（例如音源未返回直链）。

### 管理接口（需 `Authorization: <token>`）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/auth/login` | 登录，返回访问/刷新令牌 |
| POST | `/api/auth/refresh` | 刷新令牌 |
| POST | `/api/auth/logout` | 退出登录 |
| GET | `/api/user/info` | 当前登录用户信息 |
| PUT | `/api/user/profile` | 修改个人信息 |
| GET | `/api/dashboard/stats` | 概览统计 |
| GET | `/api/dashboard/trend` | 调用趋势 |
| GET | `/api/dashboard/recent-logs` | 最近调用 |
| GET | `/api/cookie/list` | Cookie 列表 |
| POST | `/api/cookie/add` | 新增 Cookie |
| PUT | `/api/cookie/update` | 修改 Cookie |
| DELETE | `/api/cookie/delete` | 删除 Cookie |
| POST | `/api/cookie/toggle` | 启用 / 停用 Cookie |
| POST | `/api/cookie/reset` | 重置失败计数 |
| GET | `/api/log/list` | 调用日志分页查询 |
| DELETE | `/api/log/clear` | 清理日志（按天或全部） |
| GET | `/api/settings` | 读取系统设置 |
| PUT | `/api/settings` | 保存系统设置 |
| PUT | `/api/settings/password` | 修改管理员密码 |
| POST | `/api/settings/icon` | 上传站点图标 |

## 音源支持

| 音源 | 标识 | 按歌名搜索 | 按 ID / 链接解析 |
| --- | --- | --- | --- |
| QQ 音乐 | `qqmusic` | 支持 | 支持 |
| 网易云音乐 | `wyy` | 支持 | 支持 |
| 酷狗音乐 | `kugou` | 支持 | 支持 |
| 汽水音乐 | `qishui` | 不支持 | 支持 |

## 使用提示

- Cookie 需要填写完整的登录 Cookie（含 `uin`、`skey` 等），否则可能只能拿到封面与歌词
- QQ 音乐播放直链需要账号开通绿钻，否则 `music_url` 为空，但封面、歌词、详情页仍可正常返回
- 各平台直链均有时效，解析后请尽快使用
- 公开接口可在系统设置中关闭，关闭后仅登录用户可调用

## 免责声明

本项目仅供个人学习与技术研究使用，所有音乐版权归各音乐平台及版权方所有，请勿用于商业用途。解析结果来自第三方平台，接口若发生变动可能导致不可用。

## 开发者

- 开发者：逸雨网络
- QQ：204325164
