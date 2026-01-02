package git

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config 定义 Git 检测的配置参数
// 调用方需将 internal/config 的 GitConfig 映射到此结构
type Config struct {
	Remote   string
	Username string
	Password string
	Branches []string
	Tags     bool
}

// Result 保存检测结果
type Result struct {
	Triggered bool
	Logs      []string
}

// Check 执行 Git 状态检测
func Check(ctx context.Context, cfgDir string, cfg Config) (*Result, error) {
	res := &Result{Logs: []string{}}

	// 1. 定位仓库
	repoPath, err := findRepoPath(cfgDir)
	if err != nil {
		res.Logs = append(res.Logs, fmt.Sprintf("git repo not found: %v", err))
		return res, nil // 仓库未找到不视为错误，只是跳过检测
	}

	// 2. 确定远端名称
	remoteName := cfg.Remote
	if remoteName == "" {
		remoteName = "origin"
	}

	// 3. 获取基础远端 URL (使用系统 git 配置)
	// 设置网络操作超时
	netCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rawURL, err := runGitCmd(netCtx, repoPath, "remote", "get-url", remoteName)
	if err != nil {
		res.Logs = append(res.Logs, fmt.Sprintf("failed to get remote url for '%s': %v", remoteName, err))
		return res, nil
	}

	// 4. 如果配置了 HTTPS 凭据，构造带认证的 URL
	remoteURL := rawURL
	if cfg.Username != "" && cfg.Password != "" && strings.HasPrefix(rawURL, "http") {
		remoteURL = buildAuthURL(rawURL, cfg.Username, cfg.Password)
	}

	// 5. 检测分支更新
	if len(cfg.Branches) > 0 {
		// 获取远端分支 Heads
		// git ls-remote --heads <url> branch1 branch2 ...
		args := []string{"ls-remote", "--heads", remoteURL}
		args = append(args, cfg.Branches...)
		out, err := runGitCmd(netCtx, repoPath, args...)
		if err != nil {
			res.Logs = append(res.Logs, fmt.Sprintf("git ls-remote heads error: %v", err))
			return res, nil
		}
		remoteHeads := parseRemoteRefs(out, "refs/heads/")

		for _, branch := range cfg.Branches {
			remoteSHA, remoteExists := remoteHeads[branch]
			if !remoteExists {
				continue // 远端不存在该分支，跳过
			}

			// 获取本地 SHA
			localSHA, err := getLocalSHA(ctx, repoPath, "refs/heads/"+branch)
			if err != nil {
				// 本地获取失败通常意味着本地没有该分支
				localSHA = "missing"
			}

			if localSHA != remoteSHA {
				res.Triggered = true
				res.Logs = append(res.Logs, fmt.Sprintf("git branch %s changed: local %s -> remote %s", branch, shortSHA(localSHA), shortSHA(remoteSHA)))
			} else {
				res.Logs = append(res.Logs, fmt.Sprintf("git branch %s no change", branch))
			}
		}
	}

	// 6. 检测新标签
	if cfg.Tags {
		// 获取远端标签
		out, err := runGitCmd(netCtx, repoPath, "ls-remote", "--tags", remoteURL)
		if err != nil {
			res.Logs = append(res.Logs, fmt.Sprintf("git ls-remote tags error: %v", err))
			return res, nil
		}
		remoteTags := parseRemoteRefs(out, "refs/tags/")

		// 获取本地标签
		localOut, err := runGitCmd(ctx, repoPath, "tag", "--list")
		if err != nil {
			res.Logs = append(res.Logs, fmt.Sprintf("git tag list error: %v", err))
			return res, nil
		}
		localTags := make(map[string]bool)
		for _, line := range strings.Split(localOut, "\n") {
			if t := strings.TrimSpace(line); t != "" {
				localTags[t] = true
			}
		}

		// 对比差异 (Remote - Local)
		var newTags []string
		for tag := range remoteTags {
			if !localTags[tag] {
				newTags = append(newTags, tag)
			}
		}

		if len(newTags) > 0 {
			res.Triggered = true
			res.Logs = append(res.Logs, fmt.Sprintf("git new tags: %s", strings.Join(newTags, ", ")))
		} else {
			res.Logs = append(res.Logs, "git no new tags")
		}
	}

	return res, nil
}

// 辅助函数

func findRepoPath(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("root reached without finding .git")
		}
		dir = parent
	}
}

func runGitCmd(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	// 禁用交互式提示，防止卡死
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v, output: %s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func buildAuthURL(rawURL, user, pass string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.User = url.UserPassword(user, pass)
	return u.String()
}

func parseRemoteRefs(output, prefix string) map[string]string {
	refs := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		sha, ref := parts[0], parts[1]

		// 忽略解引用对象 (peeled tags)
		if strings.HasSuffix(ref, "^{}") {
			continue
		}

		if strings.HasPrefix(ref, prefix) {
			name := strings.TrimPrefix(ref, prefix)
			refs[name] = sha
		}
	}
	return refs
}

func getLocalSHA(ctx context.Context, dir, ref string) (string, error) {
	// 使用 --verify 确保 ref 存在
	return runGitCmd(ctx, dir, "rev-parse", "--verify", ref)
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
