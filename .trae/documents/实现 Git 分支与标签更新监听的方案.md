## 目标
- 使用系统 Git CLI 实现“远端 vs 本地”的即时对比：分支 head 是否变化、是否出现新标签
- 配置均为可选；只要某一项启用并检测到变更，就触发脚本；若所有启用项都未变更，则不触发

## 配置（保持你的格式）
```yaml
watchs:
  docker:        # 可选；已实现（镜像 digest 对比）
    images: [...]
  git:           # 可选
    remote: origin # 可选；默认 origin
    username: xxx  # 仅 HTTPS 可选；不填则走系统 git 的凭据
    password: xxx  # 仅 HTTPS 可选；不填则走系统 git 的凭据
    branches: [main, dev]  # 可选；为空则不检查分支
    tags: true             # 可选；false/缺省则不检查标签
```
- 禁用逻辑：
  - `git` 未配置或 `branches` 为空且 `tags` 未启用 → 不执行 git 检测
  - `docker` 未配置或 `images` 为空 → 不执行 docker 检测
- 触发逻辑：任意启用监控项出现变更 → 触发；若所有启用项均无变更 → 不触发

## 仓库定位
- 从 `config.yml` 所在目录起，向上查找最近的 `.git` 目录，作为 `repoPath`
- 若未找到 `.git`：记录日志 `git repo not found`，跳过 git 检测

## 远端确定与认证
- 远端名：使用配置中的 `git.remote`，默认为 `origin`
- 使用系统 Git：`git -C <repoPath> remote get-url <remoteName>` 读取远端 URL
- 认证策略：
  - SSH：无需配置；系统 git 自动使用 SSH Agent / ~/.ssh/config / 私钥
  - HTTPS：
    - 若 `username/password` 提供：构造仅用于查询的 URL（在内存中拼接，不写日志），如 `https://username:password@host/path.git`
    - 否则：依赖系统 git 的凭据（~/.git-credentials 或交互/askpass，CI 环境一般无需交互）

## 检测实现（仅查询，不修改仓库）
- **超时控制**：所有涉及网络的 Git 命令（如 `ls-remote`）需设置超时（如 30秒），防止进程僵死
- 分支（当 `branches` 非空时）：
  - 远端 head：`git ls-remote --heads <remoteURL> <branch>` → `<sha>\trefs/heads/<branch>`
  - 本地 head：`git -C <repoPath> rev-parse refs/heads/<branch>`（若本地不存在分支，视为“local missing”）
  - 对比：`remoteSHA != localSHA` → 分支有更新
- 标签（当 `tags` 为真时）：
  - 逻辑说明：**仅监听新标签**（基于标签名是否存在），不监听标签移动（即不对比标签 SHA）
  - 远端标签集合：`git ls-remote --tags <remoteURL>` → 收集 `refs/tags/<name>`（忽略 `^{} ` 解引用或按最终对象去重）
  - 本地标签集合：`git -C <repoPath> tag --list`
  - 对比：`remoteTags - localTags` 非空 → 新标签
- 不使用 state 历史；每次都对比当前远端状态与当前本地状态

## 日志输出
- 分支：`git branch <name> changed local <localSHA|missing> -> remote <remoteSHA>` 或 `git branch <name> no change`
- 标签：`git new tags: <t1>, <t2>` 或 `git no new tags`
- 错误：`git check error: <stderr>`（不触发）
- 不打印 `username/password`

## 集成点
- 新增模块：`internal/check/git`（封装 CLI 调用与对比）
  - `FindRepoPath(cfgDir) (string, error)`
  - `GetRemoteURL(repoPath) (string, error)`
  - `BuildRemoteURLWithCreds(baseURL, user, pass) (string)`（HTTPS 可选）
  - `ListRemoteHeads(remoteURL, branches) (map[string]string, error)`
  - `GetLocalHeads(repoPath, branches) (map[string]string, error)`
  - `ListRemoteTags(remoteURL) (set[string], error)`
  - `ListLocalTags(repoPath) (set[string], error)`
  - `CompareBranches(local, remote)` / `CompareTags(local, remote)` 返回差异与触发布尔
- 在 `internal/run/run.go`：
  - Docker 检测后调用 Git 检测（仅当配置启用）
  - 合并 `triggered` 与明细日志；若任一启用项触发则执行脚本

## 安全性说明
- HTTPS 凭据：若在配置中明文填写 `username/password`，构造的 URL 可能在进程列表（`ps`）中短暂可见。建议仅在受信任环境使用，或优先使用 SSH / 系统级凭据助手（此时无需在配置填账号密码）。

## 边界与错误处理
- 网络/认证错误（含超时）：记录日志，不触发
- 本地分支不存在：记为 `missing`，若远端存在则判定“更新”
- 远端标签过多：只按标签名集合做存在性对比；性能开销可控

## 验证
- SSH 仓库（无配置）：推进远端分支或新建标签，检测到变更
- HTTPS 仓库（提供 `username/password`）：同上
- 未启用 git 或 docker：`dg run` 只记录“no changes; nothing to do”

## 变更范围
- 新增 `internal/check/git`、`internal/config` 可选字段解析（`username/password`）
- 修改 `internal/run` 集成调用

确认后我将开始实现与本地验证。