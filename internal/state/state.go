package state

import (
    "errors"
    "io/ioutil"
    "os"
    "path/filepath"
    "syscall"

    "gopkg.in/yaml.v3"
)

type State struct {
    PID        int    `yaml:"pid"`
    StartedAt  string `yaml:"started_at"`
    FinishedAt string `yaml:"finished_at"`
    LastResult string `yaml:"last_result"`
}

func path(root string) string {
    return filepath.Join(root, "state.yml")
}

func Read(root string) (*State, error) {
    p := path(root)
    b, err := ioutil.ReadFile(p)
    if err != nil {
        if os.IsNotExist(err) {
            return &State{}, nil
        }
        return nil, err
    }
    var s State
    if err := yaml.Unmarshal(b, &s); err != nil {
        return nil, err
    }
    return &s, nil
}

func Write(root string, s *State) error {
    p := path(root)
    if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
        return err
    }
    b, err := yaml.Marshal(s)
    if err != nil {
        return err
    }
    return ioutil.WriteFile(p, b, 0o644)
}

func ProcessExists(pid int) (bool, error) {
    if pid <= 0 {
        return false, nil
    }
    err := syscall.Kill(pid, 0)
    if err == nil {
        return true, nil
    }
    if errors.Is(err, syscall.ESRCH) {
        return false, nil
    }
    if errors.Is(err, syscall.EPERM) {
        return true, nil
    }
    return false, err
}
