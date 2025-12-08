package cron

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dg/internal/config"
)

func ruleLine(pathToDG, cfgAbs, cronExpr string) string {
	return fmt.Sprintf("%s %s run -config %s", cronExpr, pathToDG, cfgAbs)
}

func Install(cfgAbs string) error {
	// ensure absolute
	abs, err := filepath.Abs(cfgAbs)
	if err != nil {
		return err
	}
	// load config to get cron
	cfg, _, err := config.Load(abs)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Cron) == "" {
		return errors.New("cron expression required in config")
	}
	expr, err := normalizeCronExpr(cfg.Cron)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}
	pathToDG := currentDGPath()
	// build rule
	rl := ruleLine(pathToDG, abs, expr)
	// read existing crontab
	currentOut, _ := exec.Command("bash", "-lc", "crontab -l || true").CombinedOutput()
	lines := strings.Split(string(currentOut), "\n")
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if strings.Contains(l, "-config "+abs) {
			continue
		}
		out = append(out, l)
	}
	out = append(out, rl)
	buf := bytes.NewBufferString(strings.Join(out, "\n") + "\n")
	cmd := exec.Command("bash", "-lc", "crontab -")
	cmd.Stdin = buf
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("crontab apply failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func Uninstall(cfgAbs string) error {
	abs, err := filepath.Abs(cfgAbs)
	if err != nil {
		return err
	}
	currentOut, _ := exec.Command("bash", "-lc", "crontab -l || true").CombinedOutput()
	lines := strings.Split(string(currentOut), "\n")
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if strings.Contains(l, "-config "+abs) {
			continue
		}
		out = append(out, l)
	}
	buf := bytes.NewBufferString(strings.Join(out, "\n") + "\n")
	cmd := exec.Command("bash", "-lc", "crontab -")
	cmd.Stdin = buf
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("crontab apply failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func currentDGPath() string {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		if p, err2 := filepath.EvalSymlinks(exe); err2 == nil && p != "" {
			return p
		}
		abs, err3 := filepath.Abs(exe)
		if err3 == nil {
			return abs
		}
		return exe
	}
	return "/usr/bin/env dg"
}

func normalizeCronExpr(expr string) (string, error) {
	e := strings.TrimSpace(expr)
	e = strings.Trim(e, "'\"")
	fields := strings.Fields(e)
	if len(fields) != 5 {
		return "", fmt.Errorf("expect 5 fields, got %d", len(fields))
	}
	return strings.Join(fields, " "), nil
}
