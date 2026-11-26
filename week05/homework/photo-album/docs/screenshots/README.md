# 云端 Chromium 验收截图

以下截图来自 Nginx `http://127.0.0.1:8088` 上的双账号实测。账号和图片均为验收数据。

| 文件 | 截图内容 |
| --- | --- |
| [01-nginx-home.png](01-nginx-home.png) | 匿名首页及空公开广场 |
| [02-login-avatar.png](02-login-avatar.png) | 登录后的用户名、头像和本人相册 |
| [03-my-albums.png](03-my-albums.png) | 本人列表中的公开、私有标识 |
| [04-upload-rendered.png](04-upload-rendered.png) | 公开相册上传成功和真实图片渲染 |
| [05-public-creator.png](05-public-creator.png) | 匿名广场与创建者用户名、头像 |
| [06-private-forbidden.png](06-private-forbidden.png) | B 打开 A 的私有相册时的权限提示 |
| [07-delete-album.png](07-delete-album.png) | 主人确认删除后的成功提示 |

本机提交时请补拍带地址栏的 8088 首页，以及浏览器 Network 中显示 Request URL 和状态码的同源 API、图片和 403 证据。录屏步骤见 [Windows 部署与验收](../../deploy/README.md)。
