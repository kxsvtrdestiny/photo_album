# 个人相册

学校、姓名、学号待本人填写。

## 第 1 阶段

- [x] Go + Gin 后端和 Vue 3 + Vite 前端可独立启动
- [x] 首次启动创建 SQLite 数据库、三张表和上传目录
- [x] 后端提供健康检查，Vite 开发服务器通过同源代理调用
- [x] 前端可构建到 `frontend/dist/`

后端使用 Gin 和 ncruces/go-sqlite3。SQLite 驱动不依赖 C 编译器。Go 版本要求见 `backend/go.mod`，Node.js 版本要求见 `frontend/package.json`。

从 `backend/` 运行 `go run .`，默认监听 `127.0.0.1:8080`。启动后访问 `/api/v1/health`，预期 HTTP 200 和 `{"code":0,"message":"ok","data":{"status":"ok"}}`。SQLite 连接检查失败时返回 HTTP 503。

| 后端环境变量 | 默认值 |
| --- | --- |
| `PHOTO_ALBUM_ADDR` | `127.0.0.1:8080` |
| `PHOTO_ALBUM_DB_PATH` | `data/photo-album.db` |
| `PHOTO_ALBUM_UPLOAD_DIR` | `uploads` |

相对路径以启动后端时的工作目录为准。从 `frontend/` 运行 `npm ci` 安装锁定的依赖，再运行 `npm run dev`。打开 `http://127.0.0.1:5173/`，页面会请求健康检查并显示连接状态。仍在 `frontend/` 运行 `npm run build` 可生成生产构建。

## 后续阶段与最终验收

按阶段继续完成用户与相册 API、照片 API、前端联调、Nginx 发布与交付。最终从 Nginx 的 `http://127.0.0.1:8088/` 完成验收。`/` 托管 `frontend/dist/`，`/api/` 和 `/uploads/` 均反代 Gin，私有图片直链由 Gin 按相册归属鉴权。

| 场景 | 后续验收标准 |
| --- | --- |
| 公开内容 | 匿名可浏览公开广场、头像与公开照片。广场显示创建者用户名和头像 |
| 私有相册及图片直链 | 主人可查看。匿名访问私有照片列表或直链返回 401，其他已登录用户返回 403 |
| 照片上传 | 仅相册主人可上传。私有图片由同源 HttpOnly Cookie 认证，业务写接口使用 Bearer。退出时清除图片 Cookie |
| 截图与视频 | 覆盖 Nginx 首页、注册上传头像、私有相册仅在本人列表可见、创建公开相册并上传、公开广场和照片渲染、跨账号 403、Network 同源证据。跨账号验收同时检查私有图片直链和上传权限 |

截图放入 README 或 `docs/screenshots/`。视频命名为 `姓名_学号_photo-album.mp4`，建议不超过 5 分钟，最长 6 分钟。开头用记事本展示学校、姓名、录制日期，并按课程通知的统一话术自我介绍。视频单独提交到老师指定入口，禁止提交到 Git。[视频提交入口](https://f.kdocs.cn/g/picg2EGQ/)
