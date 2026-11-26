# 第 4 阶段 Windows 浏览器验收

把第 4 阶段 ZIP 解压到 `D:\`，确认 `D:\photo_album-phase4` 直接包含 `week05`。使用 Go 1.25+ 和 Node.js 22.12+，打开两个 PowerShell 窗口。

## 1. 启动

窗口 A。第一条命令可在任意目录执行，其余命令在后端目录执行。验收使用单独的临时数据库和上传目录。

```powershell
Set-Location 'D:\photo_album-phase4\week05\homework\photo-album\backend'
go test -v ./...
$runDir = Join-Path $env:TEMP ('photo-album-phase4-' + [guid]::NewGuid().ToString('N'))
$env:PHOTO_ALBUM_DB_PATH = Join-Path $runDir 'album.db'
$env:PHOTO_ALBUM_UPLOAD_DIR = Join-Path $runDir 'uploads'
$env:PHOTO_ALBUM_ADDR = '127.0.0.1:8080'
$env:PHOTO_ALBUM_JWT_SECRET = [guid]::NewGuid().ToString('N') + [guid]::NewGuid().ToString('N')
go run .
```

预期测试输出 `PASS`，Gin 监听 `127.0.0.1:8080`。保持窗口运行。若下载 Go 依赖时访问 `proxy.golang.org` 超时，在同一目录和窗口执行下列命令，再运行 `go run .`。

```powershell
$env:GOPROXY = 'https://goproxy.cn'
go mod download
go test -v ./...
```

窗口 B。第一条命令可在任意目录执行，其余命令在前端目录执行。

```powershell
Set-Location 'D:\photo_album-phase4\week05\homework\photo-album\frontend'
npm.cmd ci
npm.cmd run build
npm.cmd run dev
```

预期依赖安装完成，`dist/index.html` 生成，Vite 监听 `127.0.0.1:5173`。保持窗口运行。

## 2. 准备图片并打开网页

你可以使用本机的一张小于 5 MiB 的 PNG、JPEG 或 WebP。若没有图片，在第三个 PowerShell 窗口的 `D:\photo_album-phase4` 运行下列命令。预期生成桌面可选的临时 PNG 和用于错误提示验收的假 PNG，终端会显示两个文件的完整路径。

```powershell
Set-Location 'D:\photo_album-phase4'
$sample = Join-Path $env:TEMP 'photo-album-phase4-sample.png'
$fake = Join-Path $env:TEMP 'photo-album-phase4-fake.png'
[IO.File]::WriteAllBytes($sample, [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGMwSJgAAAHkASHKbNIyAAAAAElFTkSuQmCC'))
[IO.File]::WriteAllText($fake, 'not an image')
Get-Item $sample, $fake | Select-Object FullName
(Invoke-RestMethod 'http://127.0.0.1:5173/api/v1/health').data.status
Start-Process 'http://127.0.0.1:5173/'
```

预期健康检查输出 `ok`，浏览器打开「公开广场」。首次启动时显示空列表。以下步骤使用同一个浏览器配置和 `http://127.0.0.1:5173`。

## 3. 双账号浏览器验收

1. 游客点击「我的相册」，应跳转登录。点击「注册账号」，输入账号 `phase4-a`、至少 8 字节的测试密码并选择真实 PNG 作为头像。提交后应看到注册成功提示和预填的用户名。登录后进入「我的相册」，顶部显示 A 的用户名和头像。按 F5 后应保持登录。
2. 在「我的相册」创建 `A-private` 和 `A-public`，可见性分别选「私有」「公开」。检查卡片标识。进入 `A-private`，先应显示空照片列表。选择假 PNG 上传，应出现格式错误提示。换用真实图片上传，应出现成功提示并自动刷新，显示照片。
3. 记下 `A-private` 的页面地址。点击刚上传的照片，在新标签页打开，记下图片直链。A 登录时应能查看该图。刷新相册详情，仍应显示照片和上传入口。
4. 回到「我的相册」，进入 `A-public` 上传真实图片。点击「退出登录」，应回到公开广场。将刚才的私有图片直链重新打开或刷新，应返回 `401`。把私有相册页面地址重新打开，应提示使用相册主人的账号登录。已打开的直链标签页也可直接刷新验证。
5. 游客返回公开广场，应看到 `A-public` 及 A 的用户名、头像。点击进入后能看到公开照片，页面没有上传按钮。私有相册没有出现在广场。
6. 注册并登录 B（如 `phase4-b`）。检查「我的相册」为空。打开 A 的私有相册详情，应显示「无权访问此私有相册」。重新打开私有图片直链，应返回 `403`。打开 A 的公开相册，仍能看到照片，页面没有上传按钮。
7. 退出 B，再以 A 登录。为「我的相册」和公开广场准备至少 13 条记录（给 A 创建额外的公开相册即可），为任一相册上传至少 13 张照片。每页默认 12 条。三个列表分别点击「下一页」「上一页」，检查页码与记录随之切换，同一个列表的相邻两页不重复。

按 F12 打开浏览器 Network 可以核对 `401`、`403`，以及业务请求和图片均通过 `5173`。验收后在两个服务窗口分别按 `Ctrl+C` 停止。第 5 阶段再通过 Nginx 的 `8088` 统一入口做课程最终验收。
