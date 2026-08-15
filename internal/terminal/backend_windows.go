//go:build windows

package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsBackend struct {
	shell string
}

// windowsSession 直接管理 ConPTY 的两阶段拆卸：
//   - 阶段一（closePseudoConsole）：客户端进程退出后关闭伪控制台。部分
//     Windows 构建的 conhost 不会因客户端退出而关闭 ConPTY 输出管道，读
//     循环会永久阻塞；关闭伪控制台可让 conhost 冲刷剩余输出并结束管道，
//     读循环随之排空数据并收到管道结束错误。
//   - 阶段二（releaseResources）：读循环退出后释放属性表、管道与进程句柄。
//     提前关闭读端管道会丢弃 conhost 尚未冲刷的最终输出。
type windowsSession struct {
	id            string
	processHandle windows.Handle
	inWrite       windows.Handle
	outRead       windows.Handle
	pseudoConsole windows.Handle
	attrList      *windows.ProcThreadAttributeListContainer

	waitDone   chan struct{}
	waitResult ExitResult
	waitErr    error
	readClosed chan struct{}

	closeOnce      sync.Once
	ptyClosed      sync.Once
	resourceClosed sync.Once
	readSignaled   sync.Once
}

func NewBackend() Backend {
	return &windowsBackend{shell: os.Getenv("ComSpec")}
}

func (backend *windowsBackend) Start(request StartRequest) (Session, error) {
	if err := validateDirectory(request.Directory); err != nil {
		return nil, err
	}

	commandPath := request.Command
	arguments := append([]string(nil), request.Arguments...)
	environment := append([]string(nil), request.Environment...)
	if commandPath != "" && request.ShellPath != "" {
		invocation := CommandInvocationForPlatform("windows", request.ShellPath, commandPath, arguments)
		commandPath = invocation.Command
		arguments = invocation.Arguments
		environment = append(environment, invocation.EnvironmentEntries()...)
	} else if commandPath == "" {
		commandPath = request.ShellPath
	}
	if commandPath == "" {
		commandPath = backend.shell
	}
	if commandPath == "" {
		commandPath = os.Getenv("ComSpec")
	}
	if commandPath == "" {
		commandPath = "cmd.exe"
	}
	resolvedCommand, err := exec.LookPath(commandPath)
	if err != nil {
		return nil, fmt.Errorf("找不到 Windows 终端命令: %w", err)
	}

	session, err := startWindowsSession(resolvedCommand, append([]string{resolvedCommand}, arguments...), request.Directory, embeddedTerminalEnvironment(environment), int(normalizedDimension(request.Columns, 80)), int(normalizedDimension(request.Rows, 24)))
	if err != nil {
		return nil, err
	}
	session.id = sessionID(request.ID)
	go session.watchProcess()
	return session, nil
}

func startWindowsSession(command string, arguments []string, directory string, environment []string, columns, rows int) (*windowsSession, error) {
	session := &windowsSession{
		waitDone:   make(chan struct{}),
		readClosed: make(chan struct{}),
	}

	// 从机端管道交给伪控制台后立即关闭本进程副本，参考微软伪控制台示例。
	var conhostInRead, conhostOutWrite windows.Handle
	if err := windows.CreatePipe(&conhostInRead, &session.inWrite, nil, 0); err != nil {
		return nil, fmt.Errorf("创建 Windows ConPTY 输入管道失败: %w", err)
	}
	if err := windows.CreatePipe(&session.outRead, &conhostOutWrite, nil, 0); err != nil {
		_ = windows.CloseHandle(conhostInRead)
		_ = windows.CloseHandle(session.inWrite)
		return nil, fmt.Errorf("创建 Windows ConPTY 输出管道失败: %w", err)
	}
	if err := windows.CreatePseudoConsole(windows.Coord{X: int16(columns), Y: int16(rows)}, conhostInRead, conhostOutWrite, 0, &session.pseudoConsole); err != nil {
		_ = windows.CloseHandle(conhostInRead)
		_ = windows.CloseHandle(conhostOutWrite)
		_ = windows.CloseHandle(session.inWrite)
		_ = windows.CloseHandle(session.outRead)
		return nil, fmt.Errorf("创建 Windows ConPTY 失败: %w", err)
	}
	_ = windows.CloseHandle(conhostInRead)
	_ = windows.CloseHandle(conhostOutWrite)

	attributeList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		_ = session.releaseResources()
		return nil, fmt.Errorf("创建 Windows 终端属性表失败: %w", err)
	}
	// 伪控制台属性要求把句柄值本身作为指针传入（与 Win32 文档示例及
	// charmbracelet/x/conpty 一致）；借助存储重解释避免 vet 的 unsafeptr 误报。
	pseudoConsolePointer := *(*unsafe.Pointer)(unsafe.Pointer(&session.pseudoConsole))
	if err := attributeList.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, pseudoConsolePointer, unsafe.Sizeof(session.pseudoConsole)); err != nil {
		_ = session.releaseResources()
		return nil, fmt.Errorf("更新 Windows 终端属性表失败: %w", err)
	}
	session.attrList = attributeList

	argv0, err := windows.UTF16PtrFromString(command)
	if err != nil {
		_ = session.releaseResources()
		return nil, fmt.Errorf("启动 Windows 终端失败: %w", err)
	}
	argv, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(arguments))
	if err != nil {
		_ = session.releaseResources()
		return nil, fmt.Errorf("启动 Windows 终端失败: %w", err)
	}
	var dirp *uint16
	if directory != "" {
		if dirp, err = windows.UTF16PtrFromString(directory); err != nil {
			_ = session.releaseResources()
			return nil, fmt.Errorf("启动 Windows 终端失败: %w", err)
		}
	}
	envBlock := environmentBlock(environment)

	startupInfo := &windows.StartupInfoEx{}
	// 与 charmbracelet/x/conpty 保持一致：虽然伪控制台属性决定了子进程的控
	// 制台宿主，标志位仍需按库的写法设置，否则子进程控制台子系统初始化失败
	// （STATUS_DLL_INIT_FAILED）。
	startupInfo.Flags = windows.STARTF_USESTDHANDLES
	startupInfo.Cb = uint32(unsafe.Sizeof(*startupInfo))
	startupInfo.ProcThreadAttributeList = attributeList.List()
	var processInformation windows.ProcessInformation
	creationFlags := uint32(windows.CREATE_UNICODE_ENVIRONMENT) | windows.EXTENDED_STARTUPINFO_PRESENT
	if err := windows.CreateProcess(argv0, argv, nil, nil, false, creationFlags, &envBlock[0], dirp, &startupInfo.StartupInfo, &processInformation); err != nil {
		_ = session.releaseResources()
		return nil, fmt.Errorf("启动 Windows 终端失败: %w", err)
	}
	_ = windows.CloseHandle(processInformation.Thread)
	session.processHandle = processInformation.Process
	return session, nil
}

// environmentBlock 组装 CREATE_UNICODE_ENVIRONMENT 需要的 UTF-16 环境块：
// 条目以 NUL 分隔，整体以双 NUL 结束；同名键只保留首个值。
func environmentBlock(environment []string) []uint16 {
	block := make([]uint16, 0, 256)
	seen := make(map[string]bool, len(environment))
	for _, entry := range environment {
		key, _, found := strings.Cut(entry, "=")
		if found && key != "" {
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		// StringToUTF16 自带结尾 NUL，即为条目分隔符。
		block = append(block, windows.StringToUTF16(entry)...)
	}
	block = append(block, 0)
	return block
}

func (session *windowsSession) ID() string { return session.id }

func (session *windowsSession) Read(data []byte) (int, error) {
	var read uint32
	err := windows.ReadFile(session.outRead, data, &read, nil)
	if err != nil {
		// 管道已结束（或被中止）：读循环不会再读取，收尾释放全部句柄。
		session.markReadClosed()
		_ = session.releaseResources()
	}
	return int(read), err
}

func (session *windowsSession) Write(data []byte) (int, error) {
	var written uint32
	err := windows.WriteFile(session.inWrite, data, &written, nil)
	return int(written), err
}

func (session *windowsSession) Resize(columns, rows uint16) error {
	if err := windows.ResizePseudoConsole(session.pseudoConsole, windows.Coord{X: int16(columns), Y: int16(rows)}); err != nil {
		return fmt.Errorf("调整 Windows ConPTY 尺寸失败: %w", err)
	}
	return nil
}

func (session *windowsSession) Wait() (ExitResult, error) {
	<-session.waitDone
	return session.waitResult, session.waitErr
}

// watchProcess 等待客户端进程退出并缓存退出结果，随后关闭伪控制台以结束
// 读循环（见 windowsSession 的两阶段拆卸说明）。
func (session *windowsSession) watchProcess() {
	event, waitErr := windows.WaitForSingleObject(session.processHandle, windows.INFINITE)
	if waitErr != nil {
		session.waitErr = fmt.Errorf("等待 Windows 终端进程失败: %w", waitErr)
	} else if event != windows.WAIT_OBJECT_0 {
		session.waitErr = fmt.Errorf("等待 Windows 终端进程返回意外结果: %w", waitErr)
	} else {
		var exitCode uint32
		if err := windows.GetExitCodeProcess(session.processHandle, &exitCode); err != nil {
			session.waitErr = fmt.Errorf("获取 Windows 终端退出码失败: %w", err)
		} else {
			session.waitResult = exitResultFromCode(int(exitCode))
		}
	}
	session.closePseudoConsole()
	close(session.waitDone)
}

func (session *windowsSession) closePseudoConsole() {
	session.ptyClosed.Do(func() {
		windows.ClosePseudoConsole(session.pseudoConsole)
	})
}

func (session *windowsSession) markReadClosed() {
	session.readSignaled.Do(func() {
		close(session.readClosed)
	})
}

func (session *windowsSession) releaseResources() error {
	var closeError error
	session.resourceClosed.Do(func() {
		if session.attrList != nil {
			session.attrList.Delete()
		}
		closeError = errors.Join(
			windows.CloseHandle(session.inWrite),
			windows.CloseHandle(session.outRead),
			windows.CloseHandle(session.processHandle),
		)
	})
	return closeError
}

func (session *windowsSession) Close() error {
	var closeError error
	session.closeOnce.Do(func() {
		// 先关伪控制台解除读阻塞，让读循环排空 conhost 的最终输出。
		session.closePseudoConsole()
		if session.waitErr == nil && !session.processExited() {
			if err := windows.TerminateProcess(session.processHandle, 1); err != nil && !session.processExited() {
				closeError = fmt.Errorf("结束 Windows 终端进程失败: %w", err)
			}
		}
		// 等读循环取完剩余输出后再释放句柄；读循环异常未退出时兜底超时。
		select {
		case <-session.readClosed:
		case <-time.After(2 * time.Second):
			session.markReadClosed()
		}
		closeError = errors.Join(closeError, session.releaseResources())
	})
	return closeError
}

func (session *windowsSession) processExited() bool {
	select {
	case <-session.waitDone:
		return true
	default:
		return false
	}
}
