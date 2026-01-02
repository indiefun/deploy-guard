package run

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dg/internal/check/docker"
	"dg/internal/check/git"
	"dg/internal/config"
	"dg/internal/logger"
	"dg/internal/scripts"
	"dg/internal/state"
)

func Run(cfgAbs string) int {
	cfg, root, err := config.Load(cfgAbs)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	lg, err := logger.Open(root)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	defer lg.Close()
	_ = logger.Cleanup(root, cfg.Logs.RetainDays)

	st, err := state.Read(root)
	if err != nil {
		logger.Error(lg.Log, "read state: %v", err)
		return 1
	}
	exists, _ := state.ProcessExists(st.PID)
	if st.PID > 0 && exists {
		logger.Info(lg.Log, "another run is active pid=%d; skip", st.PID)
		return 0
	}

	// write current pid immediately after concurrency check
	st.PID = os.Getpid()
	st.StartedAt = time.Now().Format(time.RFC3339)
	_ = state.Write(root, st)

	// signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		st.PID = 0
		st.FinishedAt = time.Now().Format(time.RFC3339)
		st.LastResult = "error"
		_ = state.Write(root, st)
		os.Exit(1)
	}()

	// checks
	triggered := false
	if len(cfg.Watchs.Docker.Images) > 0 {
		res, err := docker.Check(cfg.Watchs.Docker.Images)
		if err != nil {
			logger.Error(lg.Log, "docker check error: %v", err)
			st.PID = 0
			st.FinishedAt = time.Now().Format(time.RFC3339)
			st.LastResult = "error"
			_ = state.Write(root, st)
			return 1
		}
		if res.Updated {
			triggered = true
			for _, d := range res.Details {
				logger.Info(lg.Log, d)
			}
		}
	}
	if len(cfg.Watchs.Git.Branches) > 0 || cfg.Watchs.Git.Tags {
		gitCfg := git.Config{
			Remote:   cfg.Watchs.Git.Remote,
			Username: cfg.Watchs.Git.Username,
			Password: cfg.Watchs.Git.Password,
			Branches: cfg.Watchs.Git.Branches,
			Tags:     cfg.Watchs.Git.Tags,
		}
		if res, err := git.Check(context.Background(), root, gitCfg); err != nil {
			logger.Error(lg.Log, "git check error: %v", err)
		} else {
			for _, l := range res.Logs {
				logger.Info(lg.Log, l)
			}
			if res.Triggered {
				triggered = true
			}
		}
	}

	if triggered {
		logger.Info(lg.Log, "changes detected; running scripts")
		if err := scripts.RunSequential(root, cfg.Scripts, lg.File, lg.File); err != nil {
			logger.Error(lg.Log, "scripts error: %v", err)
			st.PID = 0
			st.FinishedAt = time.Now().Format(time.RFC3339)
			st.LastResult = "error"
			_ = state.Write(root, st)
			return 1
		}
	} else {
		logger.Info(lg.Log, "no changes; nothing to do")
	}

	st.PID = 0
	st.FinishedAt = time.Now().Format(time.RFC3339)
	st.LastResult = "success"
	_ = state.Write(root, st)
	return 0
}
