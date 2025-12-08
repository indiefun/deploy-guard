package logger

import (
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
    "time"
)

type Logger struct {
    File *os.File
    Log  *log.Logger
}

func Open(root string) (*Logger, error) {
    dir := filepath.Join(root, "logs")
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return nil, err
    }
    name := time.Now().Format("2006-01-02") + ".log"
    p := filepath.Join(dir, name)
    f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
    if err != nil {
        return nil, err
    }
    l := log.New(f, "", log.LstdFlags)
    return &Logger{File: f, Log: l}, nil
}

func (l *Logger) Close() error {
    if l.File != nil {
        return l.File.Close()
    }
    return nil
}

func (l *Logger) Writer(prefix string) io.Writer {
    return &lineWriter{l: l.Log, prefix: prefix}
}

type lineWriter struct {
    l      *log.Logger
    prefix string
}

func (w *lineWriter) Write(p []byte) (int, error) {
    s := string(p)
    for len(s) > 0 {
        idx := -1
        for i, c := range s {
            if c == '\n' {
                idx = i
                break
            }
        }
        if idx == -1 {
            w.l.Printf("%s %s", w.prefix, s)
            break
        }
        w.l.Printf("%s %s", w.prefix, s[:idx])
        s = s[idx+1:]
    }
    return len(p), nil
}

func Cleanup(root string, retainDays int) error {
    if retainDays <= 0 {
        return nil
    }
    dir := filepath.Join(root, "logs")
    d, err := os.ReadDir(dir)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }
    cutoff := time.Now().Add(-time.Duration(retainDays) * 24 * time.Hour)
    for _, e := range d {
        if e.IsDir() {
            continue
        }
        // parse YYYY-MM-DD.log
        name := e.Name()
        if len(name) < len("2006-01-02.log") {
            continue
        }
        ts := name[:10]
        t, err := time.Parse("2006-01-02", ts)
        if err != nil {
            continue
        }
        if t.Before(cutoff) {
            _ = os.Remove(filepath.Join(dir, name))
        }
    }
    return nil
}

func Info(l *log.Logger, msg string, args ...interface{}) {
    l.Printf("INFO %s", fmt.Sprintf(msg, args...))
}

func Error(l *log.Logger, msg string, args ...interface{}) {
    l.Printf("ERROR %s", fmt.Sprintf(msg, args...))
}
