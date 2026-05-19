# 部署指南 — modraw-server

## 目录结构

```
deploy/
  playbook.yaml             单文件 Ansible playbook（编译 + 部署）
  bin/                       编译产物输出目录（gitignored）
  build.sh                   可选：本地编译脚本
```

## 快速部署

### 1. 设置密钥环境变量

```bash
export PG_PASSWORD=你的数据库密码
export JWT_SECRET=$(openssl rand -hex 32)
```

### 2. 执行部署

在项目根目录下运行：

**首次部署（PostgreSQL + 应用）：**
```bash
PG_PASSWORD=xxx JWT_SECRET=yyy \
  ansible-playbook -i '服务器IP,' -u root deploy/playbook.yaml --tags setup,deploy
```

**后续更新（仅应用）：**
```bash
PG_PASSWORD=xxx JWT_SECRET=yyy \
  ansible-playbook -i '服务器IP,' -u root deploy/playbook.yaml --tags deploy
```

**覆盖默认参数：**
```bash
PG_PASSWORD=xxx JWT_SECRET=yyy \
  ansible-playbook -i '服务器IP,' -u root deploy/playbook.yaml \
  -e server_domain=gmc.example.com -e server_port=8080
```

## 可配置变量

在命令行通过 `-e` 参数覆盖，或直接修改 `playbook.yaml` 顶部的 `vars` 段：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `server_port` | `8088` | 应用监听端口 |
| `server_domain` | 服务器IP | Nginx server_name |
| `pg_version` | `14` | PostgreSQL 版本 |
| `pg_user` | `modraw` | 数据库用户名 |
| `pg_database` | `modraw` | 数据库名 |
| `app_path` | `/opt/modraw-server` | 应用安装路径 |
| `gin_mode` | `release` | Gin 运行模式 |
| `access_token_ttl` | `15m` | 访问令牌有效期 |
| `refresh_token_ttl` | `168h` | 刷新令牌有效期 |
| `cors_origins` | `*` | CORS 允许的源 |
| `ws_max_msg_size` | `4096` | WebSocket 最大消息大小 |

## Tag 说明

| Tag | 作用 |
|---|---|
| `build` | 在本地交叉编译 Go 二进制文件 (linux/amd64) |
| `setup` | 安装 PostgreSQL，创建用户和数据库 |
| `deploy` | 编译 + 上传二进制和迁移文件 + 部署 systemd/Nginx |

## 部署内容

| 组件 | 路径 |
|---|---|
| 二进制文件 | `/opt/modraw-server/modraw-server` |
| 数据库迁移 | `/opt/modraw-server/migrations/` |
| 环境变量 | `/opt/modraw-server/.env` (权限 0640) |
| systemd 单元 | `/etc/systemd/system/modraw-server.service` |
| Nginx 站点配置 | `/etc/nginx/sites-enabled/modraw-server` |
| 应用用户 | `modraw:modraw` (系统用户，无登录权限) |

## 部署后验证

```bash
# 查看服务状态
systemctl status modraw-server

# 查看日志
journalctl -u modraw-server -f

# 测试健康检查接口
curl http://localhost:8088/health

# 通过 Nginx 测试
curl http://你的服务器IP/health
```

## 更新流程

1. 修改代码
2. 重新部署：`ansible-playbook -i '服务器IP,' -u root deploy/playbook.yaml --tags deploy`

handler 会自动重启服务，无需手动操作。

## 故障排查

**服务无法启动：**
```bash
journalctl -u modraw-server -n 50 --no-pager
```

**数据库连接失败：**
```bash
# 检查 PostgreSQL 是否运行
systemctl status postgresql

# 测试连接
sudo -u postgres psql -c "\l"
sudo -u postgres psql -c "SELECT 1" -U modraw -d modraw
```

**Nginx 502 Bad Gateway：**
```bash
# 确认应用端口在监听
ss -tlnp | grep 8088

# 查看 Nginx 错误日志
tail -f /var/log/nginx/error.log
```

**上传权限被拒绝：**
确认 SSH 用户具有 sudo 权限，或在服务器上配置免密码 sudo。
