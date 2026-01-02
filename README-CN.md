# dg — 部署守卫 (Go)

[English](README.md)

一个用 Go 编写的简单、实用的部署守卫。它检测更新（Docker 镜像摘要）并通过 cron 运行您的脚本，具有每日日志记录和基于 pid 的安全并发控制。

## 功能特性
- Docker 镜像更新检测（对比远程摘要与本地镜像）。
- 有序执行脚本，并统一记录日志（stdout/stderr）。
- 每日日志轮转和保留清理。
- 通过 `.dg/state.yml` 中的 pid 语义实现安全并发（pid=0 表示空闲；pid>0 且进程存在 → 表示运行中）。
- Cron 集成：严格通过 `-config <绝对路径>` 过滤安装/卸载规则。
- 清晰的 CLI：`run`（运行）, `install`（安装）, `uninstall`（卸载）, `help`（帮助）, `version`（版本）。
- Git 监控：检测分支 head 变更及新标签。

## 安装

**Linux**

你可以使用一键安装脚本轻松安装或更新 `dg` 到最新版本。该脚本会自动检测系统架构，下载并安装到 `/usr/local/bin` 目录。

```bash
curl -fsSL https://raw.githubusercontent.com/indiefun/deploy-guard/main/install.sh | bash
```

## 快速开始
```bash
# 构建
make build
# 或者
go build -o dg ./cmd/dg

# 安装到系统（需要 sudo）
make install
# 验证
dg version

# 使用指定配置运行
dg run -config /absolute/path/to/project/.dg/config.yml

# 为此配置安装 cron 规则
dg install -config /absolute/path/to/project/.dg/config.yml

# 卸载此配置的 cron 规则
dg uninstall -config /absolute/path/to/project/.dg/config.yml

# 从系统中卸载程序
make uninstall
```

## 配置
在 `<project>/.dg/config.yml` 放置一个 YAML 配置文件：
```yaml
cron: '*/1 * * * *'
watchs:
  docker:
    images:
      - postgres:17
      - redis:8
  git:
    remote: origin         # 可选，默认：origin
    branches: [main, dev]  # 可选
    tags: true             # 可选，检测新标签
    username: myuser       # 可选，仅 HTTPS 需要
    password: mypass       # 可选，仅 HTTPS 需要
scripts:
  - /absolute/path/script1.sh
  - ./relative/to/config.yml/dir/script2.sh
logs:
  retain_days: 7
```

- 所有相对路径均相对于 `config.yml` 所在目录解析。
- 日志和状态文件位于配置文件的同级目录：
  - 日志：`<config-dir>/logs/YYYY-MM-DD.log`
  - 状态：`<config-dir>/state.yml`
- Cron 表达式必须包含 5 个字段（分 时 日 月 周）。如果需要秒级精度，请使用外部调度器。
- `scripts` 列表不能为空。
- 必须至少启用一项监控（`docker` 或 `git`）。

## 运行行为
- 如果 `<config-dir>/state.yml` 中 `pid>0` 且对应进程存在，则跳过当前运行。
- Docker 检查：比较远程仓库摘要 (digest) 与本地镜像摘要。
- 如果检测到任何更新，将按顺序执行脚本。非零退出码将中止执行并记录错误。
- 收到 SIGINT/SIGTERM 信号时会优雅停止并回写状态。
- 完成后，写入 `pid=0`、时间戳和最后结果。

## Cron 集成
- 安装：从配置中读取 `cron` 字段，并使用当前 `dg` 二进制路径写入规则：
  - `<cron> /absolute/path/to/dg -config <abs path to config.yml>`
- 卸载：仅移除包含 `-config <绝对路径>` 的规则；注释和其他规则保持不变。

## GitHub Releases (CI)
- 推送标签 `vX.Y.Z` 以触发 CI。
- CI 构建适用于 `amd64` 和 `arm64` 的 Linux tarballs：
  - `dg_<tag>_linux_amd64.tar.gz`
  - `dg_<tag>_linux_arm64.tar.gz`
- CI 通过 `changelogithub` 创建发布说明并将 tarballs 上传到 Release。

## 版本控制与发布
- 版本常量位于：`internal/version/version.go`。
- Makefile 目标：
```bash
make release VERSION=v1.0.0  # 提升 version.go 版本，提交，打标签，推送
make bump-patch              # vX.Y.Z → vX.Y.(Z+1)
make bump-minor              # vX.Y.Z → vX.(Y+1).0
make bump-major              # vX.Y.Z → v(X+1).0.0
```

## 日志记录
- 格式：`time level message`，相关时带有模块前缀。
- 按文件名每日轮转；根据 `retain_days` 进行保留清理。

## 安全说明
- 不会记录任何机密信息 (Secrets)。
- 仓库认证通过 `go-containerregistry` 默认密钥环使用 Docker 密钥环（`~/.docker/config.json`）。
- 配置中的 Git HTTPS 凭据仅在内存中使用，但构造的 URL 可能会在进程列表 (`ps`) 中短暂可见。如有顾虑，请使用 SSH 或系统凭据助手。

## 依赖项
- `gopkg.in/yaml.v3` — YAML 解析。
- `github.com/google/go-containerregistry` — 仓库摘要获取。
- 系统 Docker CLI — 通过 `docker image inspect` 检查本地镜像。

## 路线图 (Roadmap)
- 可选的校验和及签名发布产物。

## 许可证
Apache-2.0。详见 `LICENSE`。