# WitchHunt 服务器部署指引

## 架构总览

```
互联网 → 服务器 :80
              └── Nginx 容器 (frontend)
                    ├── /* → 返回 Vue 静态文件
                    └── /api/* → 反向代理 → Go 容器 (backend:8080)
                                                └── MySQL 容器 (mysql:3306)
```

三个 Docker 容器，一个 `docker-compose.yml` 搞定。

## 第一步：服务器环境准备

服务器需要安装 **Docker** 和 **Docker Compose**。

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# 重新登录让 docker 组生效
```

确认安装成功：

```bash
docker --version
docker compose version
```

## 第二步：上传代码到服务器

**方式一：Git 拉取（推荐）**

```bash
ssh your-server
git clone <你的仓库地址> /opt/witchhunt
cd /opt/witchhunt
```

**方式二：本地打包上传**

```bash
# 本地执行
rsync -avz --exclude node_modules --exclude .git . user@server:/opt/witchhunt/
```

## 第三步：配置环境变量

```bash
cd /opt/witchhunt
cp .env.example .env
```

编辑 `.env`，**必须修改以下三个值**：

```
DB_ROOT_PASSWORD=<一个强密码>
DB_PASSWORD=<另一个强密码>
JWT_SECRET=<随机字符串>
```

快速生成随机 secret：

```bash
openssl rand -hex 32
```

## 第四步：构建并启动

```bash
docker compose up -d --build
```

这条命令会：

1. 编译 Go 后端 → 打包成约 20MB 的 Alpine 镜像
2. 编译 Vue 前端 → 打包成 Nginx 镜像（静态文件 + 反向代理）
3. 拉取 MySQL 8.0 镜像
4. 启动三个容器，按依赖顺序（MySQL 健康检查通过 → 后端启动 → 前端启动）

首次构建大约 2-3 分钟。完成后访问 `http://你的服务器IP` 即可。

## 第五步：验证

```bash
# 查看容器状态，三个都应该是 Up
docker compose ps

# 查看后端日志
docker compose logs backend

# 查看前端/Nginx 日志
docker compose logs frontend
```

## 日常运维

```bash
# 更新代码后重新部署
cd /opt/witchhunt
git pull
docker compose up -d --build

# 只重建后端（前端没改的话）
docker compose up -d --build backend

# 查看实时日志
docker compose logs -f backend

# 停止所有服务
docker compose down

# 停止并清除数据库数据（慎用）
docker compose down -v
```

## 可选：配置域名 + HTTPS

如果你有域名，修改 `web/nginx.conf` 第 3 行：

```nginx
server_name yourdomain.com;
```

然后用 Certbot 加 HTTPS，最简单的方式是在宿主机装 Nginx 做一层外层代理：

```bash
sudo apt install nginx certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com
```

外层 Nginx 监听 443，反向代理到 Docker 的 80 端口。或者把 `.env` 中的 `HOST_PORT` 改成别的端口（如 8000），外层 Nginx 代理到 8000。

## 可选：简易 CI/CD

### 方案 A：手动部署脚本

在服务器上创建 `/opt/witchhunt/deploy.sh`：

```bash
#!/bin/bash
cd /opt/witchhunt
git pull origin main
docker compose up -d --build
docker image prune -f
```

push 后 SSH 上去跑一下 `bash deploy.sh` 即可。

### 方案 B：GitHub Actions 自动部署

在仓库中创建 `.github/workflows/deploy.yml`，push 到 main 时自动 SSH 到服务器执行部署脚本。需要在 GitHub 仓库 Settings → Secrets 中配置 `SSH_HOST`、`SSH_USER`、`SSH_KEY`。

## 部署相关文件清单

| 文件 | 作用 |
|------|------|
| `server/Dockerfile` | Go 多阶段构建，最终镜像约 20MB |
| `web/Dockerfile` | Vue 构建 + Nginx 运行时 |
| `web/nginx.conf` | 静态文件服务 + API 反向代理 + WebSocket 支持 + Gzip |
| `docker-compose.yml` | 编排 mysql / backend / frontend 三个服务 |
| `.env.example` | 环境变量模板 |
| `.gitignore` | 排除 `.env` 等敏感文件 |
