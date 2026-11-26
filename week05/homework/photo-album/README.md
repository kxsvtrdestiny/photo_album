# 个人相册

学校、姓名、学号待本人填写。

## 第 1 阶段

- [x] Go + Gin 后端和 Vue 3 + Vite 前端可独立启动
- [x] 首次启动创建 SQLite 数据库、三张表和上传目录
- [x] 后端提供健康检查，Vite 开发服务器通过同源代理调用
- [x] 前端可构建到 `frontend/dist/`

后端使用 Gin 和 ncruces/go-sqlite3。SQLite 驱动不依赖 C 编译器。Go 版本要求见 `backend/go.mod`，Node.js 版本要求见 `frontend/package.json`。

在 `backend/` 设置下表中的 JWT 密钥，再运行 `go run .`，默认监听 `127.0.0.1:8080`。启动后访问 `/api/v1/health`，预期 HTTP 200 和 `{"code":0,"message":"ok","data":{"status":"ok"}}`。SQLite 连接检查失败时返回 HTTP 503。

| 后端环境变量 | 默认值 |
| --- | --- |
| `PHOTO_ALBUM_JWT_SECRET` | 必填，至少 32 字节，重启时复用同一密钥 |
| `PHOTO_ALBUM_COOKIE_SECURE` | 本地 HTTP 留空，HTTPS 部署设为 `true` |
| `PHOTO_ALBUM_ADDR` | `127.0.0.1:8080` |
| `PHOTO_ALBUM_DB_PATH` | `data/photo-album.db` |
| `PHOTO_ALBUM_UPLOAD_DIR` | `uploads` |

相对路径以启动后端时的工作目录为准。从 `frontend/` 运行 `npm ci` 安装锁定的依赖，再运行 `npm run dev`。打开 `http://127.0.0.1:5173/` 进入公开广场。联通状态可通过 `http://127.0.0.1:5173/api/v1/health` 验证。仍在 `frontend/` 运行 `npm run build` 可生成生产构建。

## 第 2 阶段

- [x] 注册时上传头像，数据库保存 bcrypt 密码哈希
- [x] 登录签发 24 小时 JWT，受保护的 API 使用 Bearer 鉴权
- [x] 创建公开或私有相册，分页查看本人相册
- [x] 匿名公开广场分页返回创建者用户名和头像

| 方法及路径 | 请求与响应 |
| --- | --- |
| `POST /api/v1/register` | multipart 字段 `username`、`password`、`avatar`。密码为 8–72 字节。头像支持 JPEG、PNG、WebP，大小上限 5 MiB，像素上限 1600 万。成功返回 201 和用户信息 |
| `POST /api/v1/login` | JSON `username`、`password`。返回 `token` 和 `user`，并设置作用域为 `/uploads/` 的 HttpOnly Cookie |
| `POST /api/v1/logout` | 清除图片 Cookie。调用端自行丢弃 Bearer Token |
| `POST /api/v1/albums` | 需要 Bearer Token。JSON 示例 `{"name":"旅行","description":"照片记录","is_public":true}`，省略 `is_public` 时为私有 |
| `GET /api/v1/albums` | 需要 Bearer Token。分页返回本人相册 |
| `GET /api/v1/albums/public` | 匿名可用。每条记录的 `creator` 含 `id`、`username`、`avatar_url` |
| `GET /uploads/avatars/<filename>` | 匿名读取已注册的头像 |

JSON 响应用 `code`、`message`、`data` 包装。成功时 `code` 为 0，例如 `{"code":0,"message":"ok","data":{"items":[],"page":1,"page_size":12,"total":0}}`。错误时 `code` 为 HTTP 状态码。列表从第 1 页开始，默认每页 12 条，最大 100 条，按相册 ID 降序。越过最后一页时返回空列表。

[Windows PowerShell 双账号验收](docs/phase2-windows.md)

## 第 3 阶段

| 方法及路径 | 请求与响应 |
| --- | --- |
| `POST /api/v1/albums/:id/photos` | 相册主人用 Bearer Token 提交 multipart 字段 `photo`。成功返回 201，照片含 `id`、`album_id`、`file_path`、`file_size`（字节）和 `created_at` |
| `GET /api/v1/albums/:id/photos` | 可选 Bearer Token。响应含 `album` 摘要、`can_upload`、`items`、`page`、`page_size`、`total`。空相册也有摘要 |
| `GET /uploads/photos/<filename>` | 公开照片可匿名读取；私有照片用登录时设置的同源 Cookie 鉴权。图片响应禁止缓存 |

公开相册允许匿名读列表和图片。私有相册的列表和图片对匿名访问返回 401，对非主人返回 403。上传公开或私有相册均限本人。上传只接受经解码验证的 JPEG、PNG、WebP，单张上限 5 MiB 和 1600 万像素，同名文件分别保存。

[Windows PowerShell 照片和权限验收](docs/phase3-windows.md)

## 第 4 阶段

- [x] Vue 3 和 Vue Router 接入注册头像、登录、退出及登录状态恢复
- [x] 我的相册支持创建、可见性标识和分页
- [x] 公开广场支持匿名浏览、创建者头像和分页
- [x] 相册详情支持真实图片、分页、本人上传以及权限和失败提示

[Windows PowerShell 启动和浏览器验收](docs/phase4-windows.md)

## 后续阶段与最终验收

下一阶段完成 Nginx 发布与交付。最终从 Nginx 的 `http://127.0.0.1:8088/` 完成验收。`/` 托管 `frontend/dist/`，`/api/` 和 `/uploads/` 均反代 Gin，私有图片直链由 Gin 按相册归属鉴权。

| 场景 | 后续验收标准 |
| --- | --- |
| 公开内容 | 匿名可浏览公开广场、头像与公开照片。广场显示创建者用户名和头像 |
| 私有相册及图片直链 | 主人可查看。匿名访问私有照片列表或直链返回 401，其他已登录用户返回 403 |
| 照片上传 | 仅相册主人可上传。私有图片由同源 HttpOnly Cookie 认证，业务写接口使用 Bearer。退出时清除图片 Cookie |
| 截图与视频 | 覆盖 Nginx 首页、注册上传头像、私有相册仅在本人列表可见、创建公开相册并上传、公开广场和照片渲染、跨账号 403、Network 同源证据。跨账号验收同时检查私有图片直链和上传权限 |

截图放入 README 或 `docs/screenshots/`。视频命名为 `姓名_学号_photo-album.mp4`，建议不超过 5 分钟，最长 6 分钟。开头用记事本展示学校、姓名、录制日期，并按课程通知的统一话术自我介绍。视频单独提交到老师指定入口，禁止提交到 Git。[视频提交入口](https://f.kdocs.cn/g/picg2EGQ/)
