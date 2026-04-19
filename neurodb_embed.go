package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type NeuroDBEmbed struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	baseDir string
	port    int
	running bool
}

func NewNeuroDBEmbed(port int) *NeuroDBEmbed {
	homeDir, _ := os.UserHomeDir()
	return &NeuroDBEmbed{
		baseDir: filepath.Join(homeDir, ".relationship-analyzer", "neurodb"),
		port:    port,
	}
}

func (e *NeuroDBEmbed) findServerBinary() string {
	exeName := "NEURO_SERVER"
	if runtime.GOOS == "windows" {
		exeName = "NEURO_SERVER.exe"
	}

	candidates := []string{
		filepath.Join(e.baseDir, "bin", exeName),
		filepath.Join(e.baseDir, exeName),
	}

	execPath, _ := os.Executable()
	if execPath != "" {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "neurodb", "bin", exeName),
			filepath.Join(execDir, "neurodb", exeName),
			filepath.Join(execDir, exeName),
		)
	}

	cwd, _ := os.Getwd()
	if cwd != "" {
		candidates = append(candidates,
			filepath.Join(cwd, "neurodb", "bin", exeName),
			filepath.Join(cwd, "neurodb", exeName),
		)
	}

	if p, err := exec.LookPath(exeName); err == nil {
		candidates = append(candidates, p)
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func (e *NeuroDBEmbed) ensureDirs() {
	for _, sub := range []string{"bin", "data", "logs", "import"} {
		os.MkdirAll(filepath.Join(e.baseDir, sub), 0755)
	}
}

func (e *NeuroDBEmbed) writeConfig() error {
	conf := fmt.Sprintf(`# NeuroDB embedded config
port %d
max-idletime 0
save-strategy 1
log-level 3
query-timeout 10
`, e.port)

	confPath := filepath.Join(e.baseDir, "bin", "neuro.conf")
	return os.WriteFile(confPath, []byte(conf), 0644)
}

func (e *NeuroDBEmbed) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	serverBin := e.findServerBinary()
	if serverBin == "" {
		return fmt.Errorf("未找到 NEURO_SERVER 二进制文件。请将 NeuroDB 放置到: %s", filepath.Join(e.baseDir, "bin/"))
	}

	e.ensureDirs()
	if err := e.writeConfig(); err != nil {
		return fmt.Errorf("写入配置失败: %w", err)
	}

	if e.isPortInUse() {
		log.Printf("NeuroDB 端口 %d 已被占用，尝试连接已有实例", e.port)
		e.running = true
		return nil
	}

	binDir := filepath.Dir(serverBin)

	absServerBin, err := filepath.Abs(serverBin)
	if err != nil {
		absServerBin = serverBin
	}

	e.cmd = exec.Command(absServerBin)
	e.cmd.Dir = binDir
	e.cmd.Stdout = e.openLogFile("stdout.log")
	e.cmd.Stderr = e.openLogFile("stderr.log")

	if err := os.Chmod(absServerBin, 0755); err != nil {
		log.Printf("chmod 失败: %v", err)
	}

	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("启动 NEURO_SERVER 失败: %w", err)
	}

	log.Printf("NeuroDB 子进程已启动 (PID: %d, 端口: %d)", e.cmd.Process.Pid, e.port)

	go func() {
		if err := e.cmd.Wait(); err != nil {
			log.Printf("NeuroDB 子进程退出: %v", err)
		}
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	if err := e.waitReady(10 * time.Second); err != nil {
		e.Stop()
		return fmt.Errorf("NeuroDB 启动超时: %w", err)
	}

	e.running = true
	log.Printf("NeuroDB 已就绪 (端口: %d)", e.port)
	return nil
}

func (e *NeuroDBEmbed) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running || e.cmd == nil || e.cmd.Process == nil {
		e.running = false
		return
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", e.port), 2*time.Second)
	if err == nil {
		conn.Write([]byte("shutdown\r\n"))
		conn.Close()
		time.Sleep(500 * time.Millisecond)
	}

	done := make(chan struct{})
	go func() {
		e.cmd.Process.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("NeuroDB 子进程已正常退出")
	case <-time.After(5 * time.Second):
		log.Println("NeuroDB 子进程强制终止")
		e.cmd.Process.Kill()
	}

	e.running = false
}

func (e *NeuroDBEmbed) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

func (e *NeuroDBEmbed) GetPort() int {
	return e.port
}

func (e *NeuroDBEmbed) GetBaseDir() string {
	return e.baseDir
}

func (e *NeuroDBEmbed) GetInstallPath() string {
	return filepath.Join(e.baseDir, "bin/")
}

func (e *NeuroDBEmbed) isPortInUse() bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", e.port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (e *NeuroDBEmbed) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", e.port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("等待 %s 超时", addr)
}

func (e *NeuroDBEmbed) openLogFile(name string) *os.File {
	path := filepath.Join(e.baseDir, "logs", name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return os.Stdout
	}
	return f
}

func (e *NeuroDBEmbed) StatusInfo() string {
	serverBin := e.findServerBinary()
	parts := []string{}
	if serverBin != "" {
		parts = append(parts, fmt.Sprintf("二进制: %s", serverBin))
	} else {
		parts = append(parts, fmt.Sprintf("二进制: 未找到 (请放置到 %s)", e.GetInstallPath()))
	}
	parts = append(parts, fmt.Sprintf("端口: %d", e.port))
	parts = append(parts, fmt.Sprintf("数据目录: %s", e.baseDir))
	if e.IsRunning() {
		parts = append(parts, "状态: 运行中")
		if e.cmd != nil && e.cmd.Process != nil {
			parts = append(parts, fmt.Sprintf("PID: %d", e.cmd.Process.Pid))
		}
	} else {
		parts = append(parts, "状态: 未运行")
	}
	return strings.Join(parts, "\n")
}
