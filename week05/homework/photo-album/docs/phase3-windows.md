# 第 3 阶段 Windows PowerShell 验收

把 GitHub 的 ZIP 直接解压到 `D:\`，确认 `D:\photo_album-phase3` 直接包含 `week05`。打开三个 PowerShell 窗口。使用 Go 1.25+ 和 Node.js 22.12+。

## 1. 启动后端

窗口 A。第一条命令可在任意目录执行，其余命令在后端目录执行。路径变量让本次验收使用新的临时数据库和上传目录。

```powershell
Set-Location 'D:\photo_album-phase3\week05\homework\photo-album\backend'
go test -v ./...
$runDir = Join-Path $env:TEMP ('photo-album-phase3-' + [guid]::NewGuid().ToString('N'))
$env:PHOTO_ALBUM_DB_PATH = Join-Path $runDir 'album.db'
$env:PHOTO_ALBUM_UPLOAD_DIR = Join-Path $runDir 'uploads'
$env:PHOTO_ALBUM_ADDR = '127.0.0.1:8080'
$env:PHOTO_ALBUM_JWT_SECRET = [guid]::NewGuid().ToString('N') + [guid]::NewGuid().ToString('N')
go run .
```

预期测试输出 `PASS`，Gin 监听 `127.0.0.1:8080`。如果 Go 下载依赖时连接 `proxy.golang.org` 超时，在同一窗口执行下列命令后重试 `go run .`。Go 依赖校验继续启用。

```powershell
$env:GOPROXY = 'https://goproxy.cn'
go mod download
go test -v ./...
```

## 2. 构建并启动 Vite

窗口 B。第一条命令可在任意目录执行，其余命令在前端目录执行。

```powershell
Set-Location 'D:\photo_album-phase3\week05\homework\photo-album\frontend'
npm.cmd ci
npm.cmd run build
npm.cmd run dev
```

预期生成 `dist/index.html`，Vite 监听 `127.0.0.1:5173`。浏览器打开 `http://127.0.0.1:5173/` 后显示「后端和数据库正常」。保留窗口 A、B 运行。

## 3. 准备两个账号和相册

窗口 C。切换到解压根目录，后续命令都在同一窗口执行并复用变量。

```powershell
Set-Location 'D:\photo_album-phase3'
$root = 'http://127.0.0.1:5173'
$api = "$root/api/v1"
(Invoke-RestMethod "$api/health").data.status

$files = Join-Path $env:TEMP ('photo-test-' + [guid]::NewGuid().ToString('N'))
$dir1 = (New-Item -ItemType Directory -Path (Join-Path $files 'first') -Force).FullName
$dir2 = (New-Item -ItemType Directory -Path (Join-Path $files 'second') -Force).FullName
$one = Join-Path $dir1 'same.png'
$two = Join-Path $dir2 'same.png'
[IO.File]::WriteAllBytes($one, [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGMwSJgAAAHkASHKbNIyAAAAAElFTkSuQmCC'))
[IO.File]::WriteAllBytes($two, [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGOYKaAAAAIPAMqLHCPIAAAAAElFTkSuQmCC'))
$pw = 'Local-Album-Test-123'
$suffix = [guid]::NewGuid().ToString('N').Substring(0, 8)
$nameA = "album-a-$suffix"
$nameB = "album-b-$suffix"
$a = (curl.exe -fsS -F "username=$nameA" -F "password=$pw" -F "avatar=@$one" "$api/register" | ConvertFrom-Json).data
$b = (curl.exe -fsS -F "username=$nameB" -F "password=$pw" -F "avatar=@$one" "$api/register" | ConvertFrom-Json).data
$a, $b | Format-Table id, username, avatar_url

$loginA = Invoke-RestMethod -Uri "$api/login" -Method Post -ContentType 'application/json' -Body (@{username=$nameA; password=$pw} | ConvertTo-Json)
$loginB = Invoke-RestMethod -Uri "$api/login" -Method Post -ContentType 'application/json' -Body (@{username=$nameB; password=$pw} | ConvertTo-Json)
$tokenA = $loginA.data.token
$tokenB = $loginB.data.token
$ha = @{Authorization="Bearer $tokenA"}
$hb = @{Authorization="Bearer $tokenB"}
function New-TestAlbum($name, $public, $headers) {
  $json = @{name=$name; is_public=$public} | ConvertTo-Json -Compress
  (Invoke-RestMethod -Uri "$api/albums" -Method Post -Headers $headers -ContentType 'application/json' -Body $json).data
}
$open = New-TestAlbum 'A-public' $true $ha
$closed = New-TestAlbum 'A-private' $false $ha
$onlyB = New-TestAlbum 'B-private' $false $hb
$publicPhotos = "$api/albums/$($open.id)/photos"
$privatePhotos = "$api/albums/$($closed.id)/photos"
(Invoke-RestMethod "$api/albums" -Headers $ha).data.items | Format-Table id, user_id, name, is_public
(Invoke-RestMethod "$api/albums" -Headers $hb).data.items | Format-Table id, user_id, name, is_public
(Invoke-RestMethod "$api/albums/public").data.items | ConvertTo-Json -Depth 6
```

预期健康状态为 `ok`，两个用户有不同的 ID 和头像地址。A 只看到自己的两个相册，B 只看到 `B-private`。匿名广场只返回 `A-public`，含 A 的用户名和头像地址。

## 4. 上传、分页和直链

仍在窗口 C 的 `D:\photo_album-phase3`。以下操作先测试权限，然后由 A 上传两张同名公开照片和一张私有照片。

```powershell
curl.exe -sS -o NUL -w '%{http_code}\n' -F "photo=@$one" "$publicPhotos"
curl.exe -sS -o NUL -w '%{http_code}\n' -H "Authorization: Bearer $tokenB" -F "photo=@$one" "$publicPhotos"
curl.exe -sS -o NUL -w '%{http_code}\n' -H "Authorization: Bearer $tokenB" -F "photo=@$one" "$privatePhotos"
$p1 = (curl.exe -fsS -H "Authorization: Bearer $tokenA" -F "photo=@$one" "$publicPhotos" | ConvertFrom-Json).data
$p2 = (curl.exe -fsS -H "Authorization: Bearer $tokenA" -F "photo=@$two" "$publicPhotos" | ConvertFrom-Json).data
$p3 = (curl.exe -fsS -H "Authorization: Bearer $tokenA" -F "photo=@$one" "$privatePhotos" | ConvertFrom-Json).data
$p1, $p2, $p3 | Format-Table id, album_id, file_size, file_path
$p1.file_path -ne $p2.file_path
1..3 | ForEach-Object {
  (Invoke-RestMethod ('{0}?page={1}&page_size=1' -f $publicPhotos, $_)).data | ConvertTo-Json -Depth 6
}
(Invoke-RestMethod "$privatePhotos" -Headers $ha).data | ConvertTo-Json -Depth 6
curl.exe -sS -o NUL -w '%{http_code}\n' "$privatePhotos"
curl.exe -sS -o NUL -w '%{http_code}\n' -H "Authorization: Bearer $tokenB" "$privatePhotos"
```

前三个状态码依次为 `401`、`403`、`403`。A 的三次上传都返回照片记录，两张同名公开照片的地址比较输出 `True`。匿名公开列表第 1 页为第二张、第 2 页为第一张、第 3 页为空，三页的 `total` 都为 2，`can_upload` 为 `false`。A 能读取私有列表，`can_upload` 为 `true`。匿名和 B 请求私有列表依次返回 `401`、`403`。

```powershell
$dl1 = Join-Path $files 'download1.png'
$dl2 = Join-Path $files 'download2.png'
curl.exe -fsS "$root$($p1.file_path)" -o "$dl1"
curl.exe -fsS "$root$($p2.file_path)" -o "$dl2"
(Get-FileHash $one).Hash -eq (Get-FileHash $dl1).Hash
(Get-FileHash $two).Hash -eq (Get-FileHash $dl2).Hash

$privateLink = "$root$($p3.file_path)"
curl.exe -sS -o NUL -w '%{http_code}\n' -b "photo_album_token=$tokenA" "$privateLink"
curl.exe -sS -o NUL -w '%{http_code}\n' "$privateLink"
curl.exe -sS -o NUL -w '%{http_code}\n' -b "photo_album_token=$tokenB" "$privateLink"
```

两个哈希比较都应为 `True`，证明匿名可以完整下载两张公开图片，且两张同名文件各自保留原内容。最后三行依次返回 `200`、`401`、`403`。这里用 curl 显式发送图片 Cookie，值是登录接口同时返回的 JWT。这样可避开 Windows PowerShell 的 Cookie 会话对 `/uploads/` 路径的处理差异。

## 5. 文件校验

仍在窗口 C 的 `D:\photo_album-phase3`。

```powershell
$fake = Join-Path $files 'fake.png'
$huge = Join-Path $files 'oversized.png'
[IO.File]::WriteAllText($fake, 'not an image')
[IO.File]::WriteAllBytes($huge, [byte[]]::new(5MB + 1))
curl.exe -sS -o NUL -w '%{http_code}\n' -H "Authorization: Bearer $tokenA" -F "photo=@$fake" "$publicPhotos"
curl.exe -sS -o NUL -w '%{http_code}\n' -H "Authorization: Bearer $tokenA" -F "photo=@$huge" "$publicPhotos"
(Invoke-RestMethod "$publicPhotos").data.total
```

预期依次为 `415`、`413`、`2`。验收完在窗口 A 和 B 分别按 `Ctrl+C` 停止服务。最终课程验收通过 Nginx 的 8088 统一入口。
