package main

import (
	"context"
	"easyftp/ftpserver"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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

	// 如果是开机启动，则默认最小化
	if config.AutoStart {
		runtime.WindowHide(ctx)
	}

	// 启动系统托盘
	go systray.Run(a.onSystrayReady, a.onSystrayExit)
}

// 系统托盘准备就绪
func (a *App) onSystrayReady() {
	iconPath := "build/windows/icon.ico"
	iconBytes, err := os.ReadFile(iconPath)
	if err == nil {
		systray.SetIcon(iconBytes)
	}
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

		// 创建一个批处理文件来启动程序
		batPath := filepath.Join(startupPath, "EasyFTPServer.bat")
		batContent := fmt.Sprintf("@echo off\nstart \"\" \"%s\"", exePath)
		err = os.WriteFile(batPath, []byte(batContent), 0644)
		if err != nil {
			return err
		}
	} else {
		// 删除批处理文件
		batPath := filepath.Join(startupPath, "EasyFTPServer.bat")
		_ = os.Remove(batPath)
		_ = os.Remove(linkPath) // 同时尝试删除可能存在的快捷方式文件
	}

	return nil
}

// 检查是否开机启动
func (a *App) CheckAutoStart() bool {
	startupPath := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "EasyFTPServer.bat")
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
