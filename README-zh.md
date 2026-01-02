# dg — 部署守卫 (Go)

[English](README.md)

dg 是一款基于 Go 语言开发的轻量且实用的部署守卫工具，旨在为项目部署流程提供可靠保障。它能够实时检测 Docker 镜像更新与 Git 仓库变更，通过 cron 定时任务触发自定义脚本执行，并具备每日日志轮转、基于 PID 的安全并发控制等核心能力，帮助开发者自动化管理部署流程，减少人工干预成本。

## 核心功能特性

dg 围绕“可靠检测、安全执行、便捷管理”三大目标设计，核心功能如下：

- **多维度更新检测**：支持 Docker 镜像（对比远程与本地镜像摘要）、Git 分支（检测 head 变更）、Git 标签（检测新标签）三种维度的更新触发。

- **有序脚本执行**：按配置顺序执行自定义脚本，统一捕获 stdout/stderr 输出，脚本非零退出码时将中止执行并记录错误。

- **智能日志管理**：默认按“YYYY-MM-DD.log”格式每日轮转日志，支持通过配置指定日志保留天数（超出自动清理）。

- **安全并发控制**：通过 `.dg/state.yml` 中的 PID 语义实现并发保护——PID=0 表示空闲可执行，PID>0 且进程存在时自动跳过当前运行。

- **灵活 cron 集成**：安装/卸载 cron 规则时，严格通过 `-config ，仅操作目标项目的定时任务，不影响其他规则。

- **清晰 CLI 交互**：提供 `run`（手动触发运行）、`install`（安装 cron 规则）、`uninstall`（卸载 cron 规则）、`help`（查看帮助）、`version`（查看版本）5 个核心命令，操作直观。

- **安全凭据处理**：不记录任何机密信息，Docker 仓库认证通过 `go-containerregistry` 调用系统默认 Docker 密钥环（`~/.docker/config.json`）；Git HTTPS 凭据仅在内存中使用（注：构造的 URL 可能在 `ps` 进程列表中短暂可见，建议优先使用 SSH 或系统凭据助手）。

## 快速使用指南

以下是从安装到部署的完整流程，适用于初次使用 dg 的用户。

### 1. 安装 dg

通过一键脚本即可将 dg 安装或更新到 `/usr/local/bin`（系统级路径，需确保权限）：

```Bash
curl -fsSL https://raw.githubusercontent.com/indiefun/deploy-guard/main/install.sh | bash
```

### 2. 项目配置

在目标项目根目录下，需创建 `.dg` 目录并配置两个核心文件：`config.yml`（主配置）和自定义脚本（如 `script1.sh`）。

#### 2.1 配置 config.yml

创建路径：`<project>/.dg/config.yml`，配置示例如下（关键字段已标注必填/可选）：

```YAML
# 定时执行规则（必填，5个字段：分 时 日 月 周，不支持秒级，需秒级精度请用外部调度器）
cron: '*/1 * * * *'        
# 监控配置（至少启用一项：docker.images/git.branches/git.tags）
watchs:
  docker:
    # 需监控的 Docker 镜像列表（示例）
    images:                
      - postgres:17
      - redis:8
  git:
    # 需监控的 Git 分支列表（示例）
    branches: [main, dev]  
    # 是否监控 Git 新标签（true/false）
    tags: true             
    # Git 远程仓库名（可选，默认：origin）
    remote: origin         
    # Git HTTPS 用户名（可选，仅 HTTPS 协议需要）
    username: myuser       
    # Git HTTPS 密码（可选，仅 HTTPS 协议需要）
    password: mypass       
# 触发更新后执行的脚本列表（必填，至少一个脚本，支持绝对路径/相对路径）
scripts:                   
  - /absolute/path/script1.sh  # 绝对路径：直接指向脚本
  - ./relative/to/config.yml/dir/script2.sh  # 相对路径：相对于 config.yml 所在目录
# 日志配置（可选）
logs:
  # 日志保留天数（可选，默认不清理，超过天数的日志自动删除）
  retain_days: 7           
```

**配置规则说明**：

- 相对路径解析：所有脚本、日志、状态文件的相对路径，均以 `config.yml` 所在目录（即 `）为基准。

- 日志与状态文件路径：

    - 日志：`/logs/YYYY-MM-DD.log`（按日期轮转）

    - 状态：`dg/state.yml`（记录 PID、执行时间戳、结果）

- 必选校验：`cron` 字段、`scripts` 列表、`watchs` 下至少一项监控，三者缺一不可。

#### 2.2 编写自定义脚本

以 `script1.sh` 为例，创建路径：`script1.sh`，脚本内容可根据业务需求自定义（示例如下）：

```Bash
# 示例：输出执行完成信息
echo 'dg 检测到更新，脚本执行完成！'
# 实际场景可添加：镜像拉取、服务重启、部署验证等逻辑
```

### 3. 测试运行

配置完成后，可先手动触发一次运行，验证配置是否正确：

```Bash
# 进入项目根目录（或直接指定 config.yml 路径，如 dg run -config /path/to/.dg/config.yml）
cd 
# 手动运行 dg
dg run
# 查看运行日志（验证执行结果）
cat .dg/logs/$(date +%Y-%m-%d).log
```

### 4. 安装 cron 定时任务

测试通过后，将 dg 配置为 cron 定时任务，实现自动化检测与执行：

```Bash
# 安装 cron 规则（自动读取 config.yml 中的 cron 表达式）
dg install
# 验证 cron 规则是否生效（查看当前用户的 cron 列表）
crontab -l
```

cron 规则格式：`表达式> /usr/local/bin/dg -config >/.dg/config.yml`

### 5. 卸载 cron 定时任务

若需停止自动化执行，可卸载对应的 cron 规则（仅移除当前项目的 dg 任务，不影响其他 cron 规则）：

```Bash
# 卸载 cron 规则
dg uninstall
# 验证卸载结果
crontab -l
```

## 开发与进阶说明

本节适用于需要二次开发、自定义构建或了解 dg 内部运行机制的开发者。

### 1. 本地构建与安装

dg 提供 Makefile 简化构建流程，需先确保本地已安装 Go 1.18+ 环境。

#### 1.1 核心 Make 命令

```Bash
# 1. 构建二进制文件（输出到项目根目录的 bin/dg）
make build

# 2. 安装到系统路径（/usr/local/bin/dg，需 sudo 权限）
sudo make install

# 3. 从系统中卸载 dg（删除 /usr/local/bin/dg）
sudo make uninstall
```

#### 1.2 版本控制与发布

dg 提供版本管理命令，支持语义化版本（SemVer）升级，版本常量定义在 `internal/version/version.go` 中。

##### 1.2.1 版本升级命令

```Bash
# 补丁版本升级（vX.Y.Z → vX.Y.(Z+1)，如 v1.0.0 → v1.0.1）
make bump-patch

# 次要版本升级（vX.Y.Z → vX.(Y+1).0，如 v1.0.1 → v1.1.0）
make bump-minor

# 主要版本升级（vX.Y.Z → v(X+1).0.0，如 v1.1.0 → v2.0.0）
make bump-major

# 自定义版本发布（指定版本号，自动更新 version.go、提交代码、打标签并推送）
make release VERSION=v1.0.0
```

##### 1.2.2 GitHub Releases 自动构建（CI）

当推送标签 `vX.Y.Z` 到 GitHub 仓库时，CI 流程会自动触发：

- 构建适用于 `amd64` 和 `arm64` 架构的 Linux 压缩包（如 `dg_v1.0.0_linux_amd64.tar.gz`）。

- 通过 `changelogithub` 生成发布说明，并将压缩包上传到 GitHub Releases 页面。

### 2. 核心运行机制

了解 dg 的运行行为，有助于排查问题和自定义脚本逻辑：

#### 2.1 并发控制逻辑

- 每次执行前，dg 会检查 `/state.yml` 中的 PID：

    - 若 PID=0：表示空闲，可正常执行。

    - 若 PID>0 且对应进程存在：表示正在运行，自动跳过当前执行（避免并发冲突）。

#### 2.2 更新检测逻辑

- **Docker 镜像检测**：对比远程仓库中镜像的摘要（digest）与本地已拉取镜像的摘要，若不一致则判定为更新。

- **Git 分支检测**：拉取远程分支最新 commit，与本地记录的上次 commit 对比，若不一致则判定为更新。

- **Git 标签检测**：拉取远程所有标签，与本地记录的上次标签列表对比，若新增标签则判定为更新。

#### 2.3 脚本执行与信号处理

- 若检测到任何更新，dg 会按 `config.yml` 中 `scripts` 的顺序执行脚本，前一个脚本非零退出码时，后续脚本将中止执行。

- 收到 SIGINT（Ctrl+C）或 SIGTERM 信号时，dg 会优雅停止当前执行，回写状态到 `state.yml`。

- 执行完成后（成功/失败），会更新 `state.yml`：PID 设为 0、记录本次执行时间戳、标记执行结果（成功/失败）。

#### 2.4 日志记录规则

- 日志格式：`[时间戳] [日志级别] [模块前缀（可选）] 日志内容`，例如：`2024-05-20T14:30:00Z INFO docker-detector 镜像 postgres:17 无更新`。

- 日志轮转：默认按日期生成日志文件（`YYYY-MM-DD.log`），每日零点自动切换新文件。

- 日志清理：若 `config.yml` 中配置了 `logs.retain_days`，则每日会自动删除超出保留天数的日志文件。

### 3. 依赖项说明

dg 依赖以下第三方库和工具，构建时会自动拉取：

- **YAML 解析**：`gopkg.in/yaml.v3`（处理 `config.yml` 和 `state.yml` 的解析与生成）。

- **Docker 镜像摘要获取**：`github.com/google/go-containerregistry`（对接 Docker 远程仓库，获取镜像摘要）。

- **系统工具依赖**：Docker CLI（本地镜像检测需通过 `docker image inspect` 命令获取本地镜像摘要，需确保系统已安装 Docker 且可执行）。

## 许可证

dg 基于 **Apache License 2.0** 开源，详见项目根目录下的 `LICENSE` 文件。