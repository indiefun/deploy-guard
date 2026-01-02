## 目标
- 提供一个简单直接的部署守卫程序 `dg`（Go 语言）
- 命令：`dg run [-config ./.dg/config.yml]`、`dg install [-config ./.dg/config.yml]`、`dg uninstall [-config ./.dg/config.yml]`、`dg help`、`dg version`
- 当前版本实现：Docker 镜像远端与本机 digest 对比；脚本顺序执行；日志按天轮转与留存清理；并发运行保护（以 pid 语义）；Cron 安装/卸载按您的最新要求
- 下一版本占位：Git branches/tags 检测接口（不实现）

## 目录结构
- `cmd/dg`：主程序入口与子命令解析
- `internal/config`：配置加载/校验与相对路径解析
- `internal/state`：`.dg/state.yml` 读写与 pid 并发检查
- `internal/logger`：按天日志与留存清理
- `internal/check/docker`：镜像 digest 比对（远端/本地）
- `internal/scripts`：脚本顺序执行与输出收集
- `internal/cron`：crontab 安装/卸载（匹配并移除仅当前配置的规则）
- `internal/version`：版本信息

## 配置与状态
- 配置文件（默认 `./.dg/config.yml`）：包含 `cron`、`watchs`、`scripts`、`logs.retain_days`
- 状态文件：`<root>/.dg/state.yml`
  - 字段：`pid`（0 表示 idle；>0 且进程存在表示 running）、`started_at`、`finished_at`、`last_result`（`success` | `error`）
  - 不记录 `status` 字段

## 运行流程（dg run）
- 解析配置路径（默认 `./.dg/config.yml`），计算 `cfgAbs` 与 `root := dir(cfgAbs)`
- 初始化日志：`<root>/.dg/logs/YYYY-MM-DD.log`，并清理过期日志（`retain_days`）
- 并发检查：读取 `<root>/.dg/state.yml`，若 `pid > 0` 且进程存在，则跳过本次检查并退出 0
- 立即写入当前进程 `pid = os.Getpid()` 与 `started_at = now`（在并发检查通过之后，脚本执行之前）
- 执行检测：Docker 镜像 digest 比对；Git branches/tags（当前版本返回未触发）
- 若任一检测触发：按顺序执行 `scripts`（`cwd = root`），收集 `stdout/stderr` 到当天日志（遇非 0 退出码停止并记录错误）
- 执行完成：写回 `pid = 0`、`finished_at = now`、`last_result = success|error`
- 退出码：成功 0；脚本或检测错误返回非 0

## Docker 检测
- 远端 digest：`github.com/google/go-containerregistry` 的 `remote.Head`/`remote.Get` 获取 `Docker-Content-Digest`（默认 keychain 支持 `~/.docker/config.json`）
- 本地 digest：`github.com/docker/docker/client` 的 `ImageInspectWithRaw` 读取 `RepoDigests`
- 比对规则：按镜像名匹配仓库，`@sha256:...` 不一致或本地缺失视为更新
- 远端失败重试：最多 3 次，指数退避

## 脚本执行
- 顺序执行；提前校验存在与可执行权限
- 运行目录：`root`（配置文件所在目录）
- 日志：子进程 `stdout/stderr` 重定向到 `<root>/.dg/logs/YYYY-MM-DD.log`
- 信号：捕获 SIGINT/SIGTERM，优雅终止并写回状态

## 日志与留存
- 文件：`<root>/.dg/logs/YYYY-MM-DD.log`，同日追加写
- 留存清理：删除早于 `retain_days` 的日志文件
- 格式：`time level module message`

## Cron 安装/卸载（按最新要求）
- 步骤一：解析 `-config`（默认 `./.dg/config.yml`）得到绝对路径 `cfgAbs`（直接使用该绝对路径）
- 步骤二：确定 `<path-to-dg>` 写法，优先使用 `/usr/bin/env dg`（从 cron 环境 PATH 查找 `dg`），若需要兼容回退则使用 `os.Executable()` 得到的绝对路径（安装时探测并记录，用于匹配卸载）
- 步骤三：规则模板固定为：`<cron> <path-to-dg> -config <cfgAbs>`（不使用 `cd`，完全用绝对配置路径）
- `dg install [-config ...]`：
  - 读取 `cfgAbs` 与 `cron`，生成 `ruleLine`（优先 `/usr/bin/env dg -config <cfgAbs>`）
  - 读取现有 `crontab -l`；仅移除包含 `-config <cfgAbs>` 且命令为 `dg` 的行（匹配两类：`/usr/bin/env dg` 或已探测到的绝对二进制路径），保留其他目录的规则，避免误伤
  - 追加写入新的 `ruleLine` 到 crontab
- `dg uninstall [-config ...]`：
  - 读取 `cfgAbs`
  - 读取现有 `crontab -l`；仅移除包含 `-config <cfgAbs>` 且命令为 `dg` 的行（同上匹配两类），保留其余规则
  - 写回更新后的 crontab（不追加新条目）
- 说明：默认优先使用 `/usr/bin/env dg` 以获得更好的环境兼容性；若卸载遇到非 env 写法（例如早期通过绝对路径安装），也能精确匹配并移除仅该配置对应的规则

## CLI 与版本
- 子命令：`run`、`install`、`uninstall`、`help`、`version`
- Flags：`-config`（支持绝对与相对路径）
- 版本：常量 `v0.1.0`

## 依赖
- `gopkg.in/yaml.v3`、`github.com/google/go-containerregistry`、`github.com/docker/docker/client`、标准库

## 下一版本占位
- `internal/check/git` 保留接口：branches head 与 tags 检测，当前返回未触发

## 验证
- 连续执行 `dg run`，第二次检测到 `pid` 在运行则跳过；通过后立即写入当前 pid
- 镜像变化触发脚本；日志与留存清理生效
- `dg install/uninstall`：安装只移除当前配置规则再写入新规则；卸载只移除当前配置规则，并保持其他目录规则不受影响