# 第 2 阶段 Windows PowerShell 验收

使用 Go 1.25 或更高版本和 Node.js 22.12 或更高版本。开三个 PowerShell 窗口，均从包含 `week05` 的目录开始。

## 1. 启动后端

窗口 A。从包含 `week05` 的目录执行第一条命令，其余命令在 `backend/` 执行。两条路径设置将这次测试隔离在临时目录，已有数据库和上传文件可继续用于日常启动。生成的密钥仅用于本次本地验收，后端重启时在同一窗口复用。

```powershell
Set-Location .\week05\homework\photo-album\backend
go test -v ./...
$runDir = Join-Path $env:TEMP ('photo-album-phase2-' + [guid]::NewGuid().ToString('N'))
$env:PHOTO_ALBUM_DB_PATH = Join-Path $runDir 'album.db'
$env:PHOTO_ALBUM_UPLOAD_DIR = Join-Path $runDir 'uploads'
$env:PHOTO_ALBUM_ADDR = '127.0.0.1:8080'
$env:PHOTO_ALBUM_JWT_SECRET = [guid]::NewGuid().ToString('N') + [guid]::NewGuid().ToString('N')
go run .
```

测试输出 `PASS`，后端日志显示监听 `127.0.0.1:8080`。保持窗口运行。

## 2. 构建并启动 Vite

窗口 B。从包含 `week05` 的目录执行第一条命令，其余命令在 `frontend/` 执行。

```powershell
Set-Location .\week05\homework\photo-album\frontend
npm.cmd ci
npm.cmd run build
npm.cmd run dev
```

依赖安装成功，构建生成 `dist/index.html`，Vite 显示 `http://127.0.0.1:5173/`。打开页面应显示「后端和数据库正常」，浏览器 Network 中的 `/api/v1/health` 返回 200。

## 3. 注册、头像和登录

窗口 C。以下验收通过 Vite 的 5173 端口调用 API，命令可在包含 `week05` 的目录执行。

```powershell
$root = 'http://127.0.0.1:5173'
$api = "$root/api/v1"
(Invoke-RestMethod "$api/health").data.status

$avatar = Join-Path $env:TEMP ('album-avatar-' + [guid]::NewGuid().ToString('N') + '.png')
[IO.File]::WriteAllBytes($avatar, [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGMwSJgAAAHkASHKbNIyAAAAAElFTkSuQmCC'))
$pw = 'Local-Album-Test-123'
$suffix = [guid]::NewGuid().ToString('N').Substring(0, 8)
$nameA = "album-a-$suffix"
$nameB = "album-b-$suffix"
$a = (curl.exe -fsS -F "username=$nameA" -F "password=$pw" -F "avatar=@$avatar" "$api/register" | ConvertFrom-Json).data
$b = (curl.exe -fsS -F "username=$nameB" -F "password=$pw" -F "avatar=@$avatar" "$api/register" | ConvertFrom-Json).data
$a, $b | Format-Table id, username, avatar_url
curl.exe -sS -o NUL -w '%{http_code}' "$root$($a.avatar_url)"
curl.exe -sS -o NUL -w '%{http_code}' -F "username=$nameA" -F "password=$pw" -F "avatar=@$avatar" "$api/register"

function Send-AlbumJson($path, $body, $headers = @{}) {
  Invoke-RestMethod -Method Post -Uri "$api$path" -Headers $headers -ContentType 'application/json' -Body (ConvertTo-Json -InputObject $body -Compress)
}
$tokenA = (Send-AlbumJson '/login' @{username=$nameA; password=$pw}).data.token
$tokenB = (Send-AlbumJson '/login' @{username=$nameB; password=$pw}).data.token
$ha = @{Authorization="Bearer $tokenA"}
$hb = @{Authorization="Bearer $tokenB"}
curl.exe -sS -o NUL -w '%{http_code}' "$api/albums"
```

健康状态为 `ok`。两个用户的 ID 和头像 URL 各不相同。三次状态码检查依次为 `200`（匿名读取头像）、`409`（重复用户名）、`401`（匿名请求我的相册），登录返回的两个 Token 用于下一步。

## 4. 创建和分页

仍在窗口 C 和包含 `week05` 的目录，继续使用上一步的变量。

```powershell
(Send-AlbumJson '/albums' @{name='A-private'} $ha).data
(Send-AlbumJson '/albums' @{name='A-public'; is_public=$true} $ha).data
(Send-AlbumJson '/albums' @{name='B-private'; is_public=$false} $hb).data
(Send-AlbumJson '/albums' @{name='B-public'; is_public=$true} $hb).data

function Read-AlbumPage($path, $page, $headers = @{}) {
  (Invoke-RestMethod -Uri ('{0}{1}?page={2}&page_size=1' -f $api, $path, $page) -Headers $headers).data | ConvertTo-Json -Depth 6
}
Read-AlbumPage '/albums' 1 $ha
Read-AlbumPage '/albums' 2 $ha
Read-AlbumPage '/albums' 1 $hb
Read-AlbumPage '/albums' 2 $hb
Read-AlbumPage '/albums/public' 1
Read-AlbumPage '/albums/public' 2
Read-AlbumPage '/albums/public' 3
```

四次创建分别返回 A 私有、A 公开、B 私有、B 公开，省略公开标记时为 `false`。A 的两页依次是 `A-public`、`A-private`，记录的 `user_id` 都是 A 的 ID。B 的两页依次是 `B-public`、`B-private`，记录的 `user_id` 都是 B 的 ID。两人的 `total` 均为 2。

匿名广场前两页依次是 `B-public`、`A-public`，嵌套的 `creator.username` 和 `creator.avatar_url` 与对应注册结果一致。第 3 页的 `items` 为空，`total` 仍为 2。

可在窗口 A 按 `Ctrl+C`，在同一窗口、同一目录重新运行 `go run .`，再从窗口 C 执行 `Read-AlbumPage '/albums' 1 $ha`。旧 Token 未过期时仍应取到 `A-public`。验收完，在窗口 A 和 B 分别按 `Ctrl+C` 停止服务。最终课程验收将通过 Nginx 的 8088 统一入口。
