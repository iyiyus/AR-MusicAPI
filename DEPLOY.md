# AR-MusicAPI 部署（解压即用 · 6 步）

部署包解压后就是 `api / db / web` 三个目录，**不用搬文件**：

```
/www/wwwroot/music.example.com/
├── api/          ← 后端（ar-musicapi、config.json、uploads/）
├── db/           ← schema.sql
└── web/          ← 前端（index.html、assets/）
```

站点根目录指向 `web`，所以 `api`、`db` 天然在网站根目录之外，`config.json` 不会被浏览器下载。

---

## 1. 上传并解压

宝塔 → 文件 → 进入 `/www/wwwroot/music.example.com/`（没有就新建）→ 上传 `ar-musicapi-linux-amd64.zip` → 右键解压。

解压后确认三个目录都在：`api/`、`db/`、`web/`。**到此文件部分就完成了**，不需要 `cp`、不需要移动。

## 2. 建库导表

宝塔 → 数据库 → 添加数据库：库名 `music_admin`、用户名 `music_admin`、字符集 `utf8mb4`，记下密码。

导入表结构（`root密码` 换成面板里显示的 MySQL root 密码）：

```bash
mysql -uroot -p'root密码' music_admin < /www/wwwroot/music.example.com/db/schema.sql
```

或用 phpMyAdmin 选中 `music_admin` → 导入 → 选 `db/schema.sql`。

## 3. 改配置

```bash
cd /www/wwwroot/music.example.com/api
cp config.example.json config.json
sed -i "s/your-password/第2步的数据库密码/" config.json
sed -i "s/please-change-this-secret/$(openssl rand -hex 32)/" config.json
chmod +x ar-musicapi && chmod 600 config.json
chown -R www:www /www/wwwroot/music.example.com
```

再用宝塔文件管理打开 `api/config.json`，确认两处：

- `"mode": "release"`（原来是 `debug`）
- `"default_password": "你的管理员密码"`

## 4. 加守护进程

软件商店装「进程守护管理器」→ 添加：

| 项 | 填什么 |
| --- | --- |
| 名称 | `ar-musicapi` |
| 运行目录 | `/www/wwwroot/music.example.com/api` |
| 启动命令 | `./ar-musicapi -config config.json` |
| 运行用户 | `www` |
| 进程数 | `1` |
| 开机自启 | 开 |

启动后验证：

```bash
curl http://127.0.0.1:8080/api/health
```

有返回就说明后端活了。

## 5. 建站点 + 替换 Nginx 配置

### ① 添加站点

网站 → 添加站点：

| 项 | 填什么 |
| --- | --- |
| 域名 | `music.example.com` |
| 根目录 | `/www/wwwroot/music.example.com/web` ← 注意是 `web` 这一层 |
| PHP 版本 | **纯静态** |

### ② 整段替换配置文件

站点设置 → 配置文件 → **把里面的内容全选删除**，粘贴下面这一整段，保存：

```nginx
server
{
    listen 80;
    listen [::]:80;
    server_name music.example.com;

    # 站点根目录指向 web，api / db 在根目录之外，config.json 不会被下载
    root /www/wwwroot/music.example.com/web;
    index index.html;

    #CERT-APPLY-CHECK--START
    # 用于SSL证书申请时的文件验证相关配置 -- 请勿删除
    include /www/server/panel/vhost/nginx/well-known/music.example.com.conf;
    #CERT-APPLY-CHECK--END

    include /www/server/panel/vhost/nginx/extension/music.example.com/*.conf;

    #SSL-START SSL相关配置，请勿删除或修改下一行带注释的404规则
    #error_page 404/404.html;
    #SSL-END

    #ERROR-PAGE-START  错误页配置，可以注释、删除或修改
    error_page 404 /404.html;
    #error_page 502 /502.html;
    #ERROR-PAGE-END

    #REWRITE-START URL重写规则引用,修改后将导致面板设置的伪静态规则失效
    include /www/server/panel/vhost/rewrite/music.example.com.conf;
    #REWRITE-END

    # 上传站点图标接口，放宽请求体积
    client_max_body_size 10m;

    # 前端：找不到文件时回落到 index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 后端接口：^~ 不能省，否则会被下面的图片/JS 规则截走
    location ^~ /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 15s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 后端上传的站点图标
    location ^~ /uploads/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        expires 7d;
    }

    # 禁止访问的敏感文件
    location ~* (\.user.ini|\.htaccess|\.htpasswd|\.env.*|\.project|\.bashrc|\.bash_profile|\.bash_logout|\.DS_Store|\.gitignore|\.gitattributes|LICENSE|README\.md|CLAUDE\.md|CHANGELOG\.md|CHANGELOG|CONTRIBUTING\.md|TODO\.md|FAQ\.md|composer\.json|composer\.lock|package(-lock)?\.json|yarn\.lock|pnpm-lock\.yaml|\.\w+~|\.swp|\.swo|\.bak(up)?|\.old|\.tmp|\.temp|\.log|\.sql(\.gz)?|docker-compose\.yml|docker\.env|Dockerfile|\.csproj|\.sln|Cargo\.toml|Cargo\.lock|go\.mod|go\.sum|phpunit\.xml|pom\.xml|build\.gradl|pyproject\.toml|requirements\.txt|application(-\w+)?\.(ya?ml|properties))$
    {
        return 404;
    }

    # 禁止访问的敏感目录
    location ~* /(\.git|\.svn|\.bzr|\.vscode|\.claude|\.idea|\.ssh|\.github|\.npm|\.yarn|\.pnpm|\.cache|\.husky|\.turbo|\.next|\.nuxt|node_modules|runtime)/ {
        return 404;
    }

    #一键申请SSL证书验证目录相关设置
    location ~ \.well-known{
        allow all;
    }

    #禁止在证书验证目录放入敏感文件
    if ( $uri ~ "^/\.well-known/.*\.(php|jsp|py|js|css|lua|ts|go|zip|tar\.gz|rar|7z|sql|bak)$" ) {
        return 403;
    }

    location ~ .*\.(gif|jpg|jpeg|png|bmp|swf)$
    {
        expires      30d;
        error_log /dev/null;
        access_log /dev/null;
    }

    location ~ .*\.(js|css)?$
    {
        expires      12h;
        error_log /dev/null;
        access_log /dev/null;
    }
    access_log  /www/wwwlogs/music.example.com.log;
    error_log  /www/wwwlogs/music.example.com.error.log;
}
```

保存后在面板点「重载配置」（或 `nginx -t && nginx -s reload`）。

这份配置就是宝塔默认配置改出来的，只动了三处：`root` 指到 `web`、去掉 PHP、加上 `/api/` 与 `/uploads/` 反代。

### ③ 开 HTTPS

站点设置 → SSL → Let's Encrypt 申请 → 打开「强制 HTTPS」。

证书由面板自动写入上面保留的 `#SSL-START` 位置，不需要手工改配置。

> 如果之前已经申请过证书，替换配置后回到 SSL 页面点一次「保存」即可恢复 443 配置。

### ④ 验证

```bash
curl https://music.example.com/api/health
```

## 6. 登录后台

- 首页：`https://music.example.com/`
- 后台：`https://music.example.com/#/dashboard`
- 账号：`admin` / 第 3 步设置的密码

登录后做三件事：

1. 系统设置 → 改密码、填站点名称、上传站点图标
2. Cookie 池 → 按音源粘贴登录 Cookie（QQ 要绿钻才有播放直链）
3. 搜一首歌，确认能播放

---

## 出问题只看这三条

| 现象 | 原因 |
| --- | --- |
| 502 Bad Gateway | 守护进程没启动，或运行目录/启动命令填错。先 `curl 127.0.0.1:8080/api/health` |
| 页面白屏 | 站点根目录填成了 `music.example.com`，应填 `music.example.com/web` |
| 页面能开，接口 404 | 配置文件没整段替换成功，或替换后没点「重载配置」 |

## 更新版本

解压新包得到 `api/`、`web/`，直接覆盖同名目录：

```bash
cd /www/wwwroot/music.example.com
# 前端
rm -rf web && mv 新包/web .
# 后端
cp 新包/api/ar-musicapi api/ar-musicapi && chmod +x api/ar-musicapi
```

`config.json` 和 `uploads/` 保留不动。然后在面板里重启守护进程，浏览器 `Ctrl+F5` 强刷。
