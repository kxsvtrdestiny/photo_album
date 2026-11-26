# Windows 本地部署与验收

最终从 `http://127.0.0.1:8088/` 打开应用。使用 Go 1.25+、Node.js 22.12+ 和适用于 Windows 的 Nginx 稳定版。已在 Debian 的 Nginx 1.26.3 上验证本配置。

把第 5 阶段 ZIP 解压到 `D:\`，使 `D:\photo_album-phase5` 直接包含 `week05`。如果解压工具产生了额外目录，请将下面的 `$project` 改为实际项目路径。下载官方 Windows ZIP，解压到 `D:\nginx`，使该目录直接包含 `nginx.exe`、`conf\mime.types` 及 `logs`。如果有一层版本号目录，将步骤 3 的 `$nginxHome` 改为实际路径。[Nginx 官方下载](https://nginx.org/en/download.html)

如第 4 阶段的 Gin 还占用 `8080`，先在原窗口按 `Ctrl+C`。本流程使用 `D:\photo-album-phase5-data` 持久化资料，便于重启后继续验收。想沿用第 4 阶段数据时，先备份原数据库及其伴随文件和上传目录，然后将步骤 2 的数据库和上传路径指向原位置。密钥变化后请重新登录。

## 1. 构建前端

在窗口 A 从任意目录执行前两行进入 `frontend/`，然后安装和构建。预期生成 `frontend\dist\index.html` 和 `assets`。

```powershell
$project = 'D:\photo_album-phase5\week05\homework\photo-album'
Set-Location (Join-Path $project 'frontend')
npm.cmd ci
npm.cmd run build
Test-Path -LiteralPath .\dist\index.html
```

最后一行预期输出 `True`。

## 2. 启动后端

在窗口 B 从任意目录进入 `backend/`。密钥在本机首次生成，保存于 ZIP 外的本地数据目录，后续重启复用。请保留此文件，并限制访问权限。

```powershell
$project = 'D:\photo_album-phase5\week05\homework\photo-album'
Set-Location (Join-Path $project 'backend')
$data = 'D:\photo-album-phase5-data'
New-Item -ItemType Directory -Force -Path $data | Out-Null
$keyFile = Join-Path $data 'jwt-secret.txt'
if (-not (Test-Path -LiteralPath $keyFile)) {
  [IO.File]::WriteAllText($keyFile, [guid]::NewGuid().ToString('N') + [guid]::NewGuid().ToString('N'))
}
$env:PHOTO_ALBUM_JWT_SECRET = [IO.File]::ReadAllText($keyFile).Trim()
$env:PHOTO_ALBUM_DB_PATH = Join-Path $data 'album.db'
$env:PHOTO_ALBUM_UPLOAD_DIR = Join-Path $data 'uploads'
$env:PHOTO_ALBUM_ADDR = '127.0.0.1:8080'
$env:PHOTO_ALBUM_COOKIE_SECURE = 'false'
go test -v ./...
go run .
```

预期 Go 测试显示 `PASS`，Gin 监听 `127.0.0.1:8080`。保持窗口运行，结束时按 `Ctrl+C`。若首次下载依赖访问 `proxy.golang.org` 超时，在同一窗口执行 `$env:GOPROXY = 'https://goproxy.cn'`，再执行 `go mod download`、`go test -v ./...`、`go run .`。Go 依赖校验继续启用。

## 3. 检查并启动 Nginx

在窗口 C 从任意目录设置路径，然后切到 Nginx 解压目录。命令把仓库中的模板渲染到 Nginx 的 `conf\photo-album.conf`。渲染使用 `dist` 的绝对路径并将反斜杠替换成正斜杠，保留配置中的路径引号。

```powershell
$project = 'D:\photo_album-phase5\week05\homework\photo-album'
$nginxHome = 'D:\nginx'
$dist = (Resolve-Path -LiteralPath (Join-Path $project 'frontend\dist') -ErrorAction Stop).Path.Replace('\','/')
Set-Location -LiteralPath $nginxHome
New-Item -ItemType Directory -Force -Path .\logs | Out-Null
$source = Get-Content -Raw -LiteralPath (Join-Path $project 'deploy\nginx.conf')
$config = Join-Path $nginxHome 'conf\photo-album.conf'
[IO.File]::WriteAllText($config, $source.Replace('__PHOTO_ALBUM_DIST__', $dist), [Text.UTF8Encoding]::new($false))
.\nginx.exe -v
.\nginx.exe -p . -c conf/photo-album.conf -t
```

预期打印真实安装版本，并显示配置语法正确和测试成功。通过后，仍在 Nginx 解压目录执行：

```powershell
Start-Process -FilePath (Join-Path $nginxHome 'nginx.exe') -WorkingDirectory $nginxHome -ArgumentList '-p', '.', '-c', 'conf/photo-album.conf'
```

仍在同一窗口，检查实际请求：

```powershell
(Invoke-RestMethod 'http://127.0.0.1:8088/api/v1/health').data.status
Start-Process 'http://127.0.0.1:8088/'
```

预期输出 `ok`，浏览器显示公开广场。后端端口冲突时，对应修改 `PHOTO_ALBUM_ADDR` 及配置中两处 `proxy_pass`。网关端口冲突时修改 `listen`，并用修改后的入口地址验证。渲染配置后用同一工作目录、前缀和配置文件操作：

```powershell
.\nginx.exe -p . -c conf/photo-album.conf -t
.\nginx.exe -p . -c conf/photo-album.conf -s reload
.\nginx.exe -p . -c conf/photo-album.conf -s quit
```

前两条用于修改后检查和重载，最后一条用于验收结束时平滑停止。错误、访问日志位于当前 Nginx 解压目录的 `logs\photo-album-error.log`、`logs\photo-album-access.log`。

## 4. 浏览器最终验收

所有场景均从 `http://127.0.0.1:8088/` 开始。准备两个演示账号 A、B 及可公开展示的图片。

| 步骤 | 操作 | 通过标准 |
| --- | --- | --- |
| 1 ★ | 匿名打开首页 | 显示公开广场，地址栏使用 8088 |
| 2 ★ | 注册 B、上传头像、登录 | 登录后顶部显示 B 的用户名和头像，刷新仍保持登录 |
| 3 ★ | B 创建私有相册并上传一张照片 | 「我的相册」可见，公开广场无此相册。B 能打开照片和图片直链 |
| 4 ★ | B 创建公开相册并上传照片 | 游客可在公开广场看到相册、创建者名称和头像；详情显示照片。刷新详情 URL 仍能加载 |
| 5 ★ | 退出 B 后重开私有直链，再登录 A 并打开 B 的详情与私有图片直链 | 匿名直链返回 401；A 的详情显示权限提示且列表请求与图片直链均返回 403。A 查看 B 的公开相册时无上传入口 |
| 6 ★ | F12 → Network，选择一个业务 API 和图片请求 | Request URL 和页面使用相同的 `127.0.0.1:8088`，例如 `/api/v1/albums/public` 与 `/uploads/photos/…` |
| 7 | 分别准备 13 条本人相册、公开相册及同一相册内的照片 | 前两页分别显示 12、1，切页无重复。桌面照片网格满页为 6 + 6 |
| 8 | 保持浏览器登录，重启 Gin 并重新请求本人相册 | 使用同一密钥、数据库和上传目录，未过期登录及旧数据仍可用 |
| 9 | 另建一个可丢弃的相册和照片。先取消删除，再确认删除 | 取消时相册仍在；确认后本人列表更新，公开相册也会从广场消失，旧相册 API 和旧图片直链返回 404 |
| 10 | 上传非图片及超过 5 MiB 的文件 | 页面分别提示格式错误和大小超限，列表总数不变 |

建议按表顺序保留带 ★ 的视频和截图。截图中保留状态码和 URL，遮住浏览器 Network 的 Authorization 和 Cookie。

## 若发布到云服务器

构建前端并上传 `frontend/dist`，在目标操作系统构建或安装 Gin 二进制。配置持久化数据库和上传卷，从秘密管理服务为后端注入固定 JWT 密钥。用系统服务管理 Gin 进程，保持其监听回环地址。将 Nginx 根路径指向构建产物，配置域名和 TLS 证书，HTTPS 下设置 `PHOTO_ALBUM_COOKIE_SECURE=true`。检查配置后启动网关，从正式域名验证三类路径及登录和图片权限。
