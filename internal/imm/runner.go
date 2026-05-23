package imm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	DefaultTimeout        = 3 * time.Second
	DefaultMaxSourceBytes = 64 * 1024
	DefaultMaxOutputBytes = 64 * 1024
)

type Runner struct {
	BinaryPath          string
	Timeout             time.Duration
	MaxSourceBytes      int
	MaxOutputBytes      int
	DisableMacOSSandbox bool
}

type Request struct {
	Source    string
	Args      []string
	RawArgs   string
	UserID    string
	ChannelID string
	GuildID   string
	Trace     bool
}

type Result struct {
	Stdout          string
	Stderr          string
	ExitCode        int
	TimedOut        bool
	OutputTruncated bool
	Duration        time.Duration
}

func NewRunner(binaryPath string) *Runner {
	if strings.TrimSpace(binaryPath) == "" {
		binaryPath = "imm"
	}
	return &Runner{
		BinaryPath:     binaryPath,
		Timeout:        DefaultTimeout,
		MaxSourceBytes: DefaultMaxSourceBytes,
		MaxOutputBytes: DefaultMaxOutputBytes,
	}
}

func (r *Runner) Run(ctx context.Context, req Request) (Result, error) {
	return r.execute(ctx, "run", BuildSource(req), req.Trace)
}

func (r *Runner) Check(ctx context.Context, req Request) (Result, error) {
	return r.execute(ctx, "check", BuildSource(req), false)
}

func BuildSource(req Request) string {
	prefix := strings.Join([]string{
		"stash bot_args = " + immArray(req.Args),
		"stash bot_raw = " + immString(req.RawArgs),
		"stash bot_user_id = " + immString(req.UserID),
		"stash bot_channel_id = " + immString(req.ChannelID),
		"stash bot_guild_id = " + immString(req.GuildID),
		"",
	}, "\n")

	source := strings.TrimSpace(stripCodeFence(req.Source))
	if hasEntrypoint(source) {
		return prefix + source + "\n"
	}

	return prefix + "marmot main {\n" + indentSource(source) + "\n}\n"
}

func stripCodeFence(source string) string {
	text := strings.TrimSpace(source)
	if !strings.HasPrefix(text, "```") || !strings.HasSuffix(text, "```") {
		return source
	}
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		return source
	}
	return strings.Join(lines[1:len(lines)-1], "\n")
}

func hasEntrypoint(source string) bool {
	return strings.Contains(source, "marmot main")
}

func indentSource(source string) string {
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func immArray(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, immString(value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func immString(value string) string {
	return strconv.Quote(value)
}

func (r *Runner) execute(ctx context.Context, mode, source string, trace bool) (Result, error) {
	started := time.Now()
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	maxSourceBytes := r.MaxSourceBytes
	if maxSourceBytes <= 0 {
		maxSourceBytes = DefaultMaxSourceBytes
	}
	maxOutputBytes := r.MaxOutputBytes
	if maxOutputBytes <= 0 {
		maxOutputBytes = DefaultMaxOutputBytes
	}

	if strings.TrimSpace(source) == "" {
		return Result{}, fmt.Errorf("IMM source is required")
	}
	if len([]byte(source)) > maxSourceBytes {
		return Result{}, fmt.Errorf("IMM source must be %d bytes or less", maxSourceBytes)
	}

	tempDir, err := os.MkdirTemp("", "nkmzbot-imm-")
	if err != nil {
		return Result{}, fmt.Errorf("create IMM temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)
	_ = os.Chmod(tempDir, 0o700)

	sourcePath := filepath.Join(tempDir, "main.imm")
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		return Result{}, fmt.Errorf("write IMM source: %w", err)
	}

	command, args, err := r.command(tempDir, mode, sourcePath, trace)
	if err != nil {
		return Result{}, err
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, command, args...)
	cmd.Dir = tempDir
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + tempDir,
		"TMPDIR=" + tempDir,
		"TEMP=" + tempDir,
		"TMP=" + tempDir,
		"NO_COLOR=1",
		"RUST_BACKTRACE=0",
		"IMM_BOT_SANDBOX=1",
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("open IMM stdout: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, fmt.Errorf("open IMM stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("start IMM: %w", err)
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-runCtx.Done():
			killProcessGroup(cmd.Process)
		case <-done:
		}
	}()

	stdout := newLimitBuffer(maxOutputBytes, cancel)
	stderr := newLimitBuffer(maxOutputBytes, cancel)
	var wg sync.WaitGroup
	wg.Add(2)
	go copyOutput(&wg, stdout, stdoutPipe)
	go copyOutput(&wg, stderr, stderrPipe)

	waitErr := cmd.Wait()
	close(done)
	wg.Wait()

	result := Result{
		Stdout:          stdout.String(),
		Stderr:          stderr.String(),
		ExitCode:        -1,
		TimedOut:        runCtx.Err() == context.DeadlineExceeded,
		OutputTruncated: stdout.Truncated() || stderr.Truncated(),
		Duration:        time.Since(started),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if waitErr != nil && result.ExitCode == -1 && !result.TimedOut && !result.OutputTruncated {
		return result, fmt.Errorf("run IMM: %w", waitErr)
	}
	return result, nil
}

func (r *Runner) command(tempDir, mode, sourcePath string, trace bool) (string, []string, error) {
	args := []string{mode, sourcePath}
	if mode == "run" && trace {
		args = append(args, "--trace")
	}

	binary := strings.TrimSpace(r.BinaryPath)
	if binary == "" {
		binary = "imm"
	}

	realBinary, err := resolveBinary(binary)
	if err != nil {
		return "", nil, err
	}
	if runtime.GOOS == "darwin" && !r.DisableMacOSSandbox && canUseMacSandbox() {
		sandboxBinary := filepath.Join(tempDir, filepath.Base(realBinary))
		if err := copyFile(realBinary, sandboxBinary, 0o700); err != nil {
			return "", nil, fmt.Errorf("prepare IMM sandbox binary: %w", err)
		}
		profilePath := filepath.Join(tempDir, "sandbox.sb")
		if err := os.WriteFile(profilePath, []byte(macSandboxProfile(tempDir)), 0o600); err != nil {
			return "", nil, fmt.Errorf("write IMM sandbox profile: %w", err)
		}
		return "/usr/bin/sandbox-exec", append([]string{"-f", profilePath, sandboxBinary}, args...), nil
	}

	return realBinary, args, nil
}

func resolveBinary(binary string) (string, error) {
	if filepath.IsAbs(binary) {
		return filepath.EvalSymlinks(binary)
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("IMM binary not found: %s", binary)
	}
	return filepath.EvalSymlinks(path)
}

func canUseMacSandbox() bool {
	if os.Getenv("IMM_BOT_DISABLE_OS_SANDBOX") == "1" {
		return false
	}
	info, err := os.Stat("/usr/bin/sandbox-exec")
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func macSandboxProfile(tempDir string) string {
	homeDir, _ := os.UserHomeDir()
	return strings.Join([]string{
		"(version 1)",
		"(deny default)",
		"(allow process-exec)",
		"(allow process-fork)",
		"(allow signal)",
		"(allow sysctl-read)",
		"(allow mach-lookup)",
		"(allow file-read*)",
		"(deny file-read* (subpath " + strconv.Quote(homeDir) + "))",
		"(deny file-read* (subpath " + strconv.Quote("/Users") + "))",
		"(allow file-read* (subpath " + strconv.Quote(tempDir) + "))",
		"(allow file-write* (subpath " + strconv.Quote(tempDir) + "))",
		"(deny network*)",
		"",
	}, "\n")
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

func killProcessGroup(process *os.Process) {
	if process == nil {
		return
	}
	if process.Pid > 0 {
		_ = syscall.Kill(-process.Pid, syscall.SIGKILL)
	}
	_ = process.Kill()
}

type limitBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
	cancel    context.CancelFunc
}

func newLimitBuffer(limit int, cancel context.CancelFunc) *limitBuffer {
	return &limitBuffer{limit: limit, cancel: cancel}
}

func (b *limitBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		b.cancel()
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		b.cancel()
		return len(p), nil
	}
	_, _ = b.buf.Write(p)
	return len(p), nil
}

func (b *limitBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *limitBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}

func copyOutput(wg *sync.WaitGroup, dst io.Writer, src io.Reader) {
	defer wg.Done()
	_, _ = io.Copy(dst, src)
}
