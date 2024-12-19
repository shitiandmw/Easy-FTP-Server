package main

import (
	"context"
	"easyftp/ftpserver"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "embed"

	"github.com/getlantern/systray"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var iconBytes []byte

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 加载配置
	config, err := a.LoadConfig()
	if err == nil && config.AutoStart {
		// 如果配置了自动启动，则启动服务器
		ftpServer = ftpserver.NewFTPServer(config.RootDir)
		ftpServer.SetCredentials(config.Username, config.Password)
		ftpServer.SetPort(config.Port)
		go ftpServer.Start()
	}

	// 监听窗口关闭事件，改为最小化而不是退出
	runtime.EventsOn(ctx, "window-close-requested", func(optionalData ...interface{}) {
		runtime.WindowHide(ctx)
	})

	// 启动系统托盘
	go systray.Run(a.onSystrayReady, a.onSystrayExit)

	// 如果是开机启动，等待一小段时间让托盘图标准备好，然后再最小化
	if config.AutoStart {
		time.Sleep(time.Millisecond * 500) // 给托盘图标一些初始化时间
		runtime.WindowHide(ctx)
	}
}

// 系统托盘准备就绪
func (a *App) onSystrayReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("Easy FTP Server")
	systray.SetTooltip("Easy FTP Server")

	mShow := systray.AddMenuItem("显示主窗口", "显示主窗口")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出应用")

	// 处理菜单点击
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				a.showMainWindow()
				// 重新启用菜单项
				mShow.Enable()
			case <-mQuit.ClickedCh:
				systray.Quit()
				runtime.Quit(a.ctx)
				return
			}
		}
	}()
}

// 显示主窗口
func (a *App) showMainWindow() {
	runtime.WindowShow(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// 系统托盘退出
func (a *App) onSystrayExit() {
	// 清理工作
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// 关闭系统托盘
	systray.Quit()

	// 如果服务器正在运行，停止它
	if ftpServer != nil {
		ftpServer.Stop()
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

type FTPConfig struct {
	RootDir   string `json:"RootDir"`
	Username  string `json:"Username"`
	Password  string `json:"Password"`
	Port      string `json:"Port"`
	AutoStart bool   `json:"AutoStart"`
}

type DefaultConfig struct {
	RootDir   string `json:"RootDir"`
	Username  string `json:"Username"`
	Password  string `json:"Password"`
	Port      string `json:"Port"`
	AutoStart bool   `json:"AutoStart"`
}

var ftpServer *ftpserver.FTPServer

func (a *App) StartFTPServer(config FTPConfig) string {
	if ftpServer != nil {
		return "FTP服务器已经在运行"
	}

	ftpServer = ftpserver.NewFTPServer(config.RootDir)
	ftpServer.SetCredentials(config.Username, config.Password)
	ftpServer.SetPort(config.Port)

	err := ftpServer.Start()
	if err != nil {
		return "启动FTP服务器失败: " + err.Error()
	}

	return "FTP服务器启动成功"
}

func (a *App) StopFTPServer() string {
	if ftpServer == nil {
		return "FTP服务器未运行"
	}

	err := ftpServer.Stop()
	if err != nil {
		return "停止FTP服务器失败: " + err.Error()
	}

	ftpServer = nil
	return "FTP服务器已停止"
}

func (a *App) OpenDirectoryDialog() string {
	dialog := runtime.OpenDialogOptions{
		Title: "选择FTP服务器根目录",
	}
	result, err := runtime.OpenDirectoryDialog(a.ctx, dialog)
	if err != nil {
		return ""
	}
	return result
}

func (a *App) GetDefaultConfig() DefaultConfig {
	// 获取程序的执行路径作为默认目录
	exePath, err := os.Executable()
	if err != nil {
		// 如果获取失败，使用当前工作目录
		exePath, _ = os.Getwd()
	}
	rootDir := filepath.Dir(exePath)

	return DefaultConfig{
		RootDir:  rootDir,
		Username: "admin",  // 默认用户名
		Password: "123456", // 默认密码
		Port:     "2121",   // 默认端口
	}
}

// 获取配置文件路径
func getConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		exePath, _ = os.Getwd()
	}
	return filepath.Join(filepath.Dir(exePath), "config.json")
}

// 保存配置到文件
func (a *App) SaveConfig(config FTPConfig) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0644)
}

// 加载配置文件
func (a *App) LoadConfig() (DefaultConfig, error) {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果配置文件不存在，返回默认配置
			return a.GetDefaultConfig(), nil
		}
		return DefaultConfig{}, err
	}

	var config DefaultConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return DefaultConfig{}, err
	}
	return config, nil
}

// 设置开机启动
func (a *App) SetAutoStart(enable bool) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// 获取当前用户的启动文件夹路径
	startupPath := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	linkPath := filepath.Join(startupPath, "EasyFTPServer.lnk")

	if enable {
		// 创建启动文件夹（如果不存在）
		err = os.MkdirAll(startupPath, 0755)
		if err != nil {
			return err
		}

		// 初始化 COM
		err = ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
		if err != nil {
			return err
		}
		defer ole.CoUninitialize()

		// 创建 Shell 对象
		shell, err := oleutil.CreateObject("WScript.Shell")
		if err != nil {
			return err
		}
		defer shell.Release()

		// 获取 IDispatch
		shellDispatch, err := shell.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return err
		}
		defer shellDispatch.Release()

		// 创建快捷方式
		shortcut, err := oleutil.CallMethod(shellDispatch, "CreateShortcut", linkPath)
		if err != nil {
			return err
		}
		defer shortcut.ToIDispatch().Release()

		// 设置快捷方式属性
		oleutil.PutProperty(shortcut.ToIDispatch(), "TargetPath", exePath)
		oleutil.PutProperty(shortcut.ToIDispatch(), "WorkingDirectory", filepath.Dir(exePath))
		oleutil.PutProperty(shortcut.ToIDispatch(), "Description", "Easy FTP Server")

		// 保存快捷方式
		oleutil.CallMethod(shortcut.ToIDispatch(), "Save")
	} else {
		// 删除快捷方式文件
		_ = os.Remove(linkPath)
		// 删除可能存在的旧的批处理文件（兼容性清理）
		_ = os.Remove(filepath.Join(startupPath, "EasyFTPServer.bat"))
	}

	return nil
}

// 检查是否开机启动
func (a *App) CheckAutoStart() bool {
	startupPath := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "EasyFTPServer.lnk")
	_, err := os.Stat(startupPath)
	return err == nil
}

// 检查服务器是否正在运行
func (a *App) IsServerRunning() bool {
	return ftpServer != nil
}

// 最小化到托盘
func (a *App) MinimizeToTray() {
	runtime.WindowHide(a.ctx)
}

// GetServerIP returns the server's IP address
func (a *App) GetServerIP() string {
	if ftpServer == nil {
		return ""
	}
	return ftpServer.GetServerIP()
}
