package scripts

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

func RunSequential(root string, scripts []string, stdout, stderr *os.File) error {
    for _, s := range scripts {
        if _, err := os.Stat(s); err != nil {
            return fmt.Errorf("script not found: %s", s)
        }
        if err := os.Chmod(s, 0o755); err != nil {
            // ignore chmod errors; script may already be executable
        }
        cmd := exec.Command(s)
        cmd.Dir = root
        cmd.Stdout = stdout
        cmd.Stderr = stderr
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("script failed: %s: %w", filepath.Base(s), err)
        }
    }
    return nil
}
