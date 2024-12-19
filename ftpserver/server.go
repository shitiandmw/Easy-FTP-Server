package ftpserver

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/goftp/server"
)

type FTPServer struct {
	rootDir  string
	username string
	password string
	port     string
	server   *server.Server
	mu       sync.Mutex
	logFile  *os.File
}

// 创建新的 FTP 服务器实例
func NewFTPServer(rootDir string) *FTPServer {
	return &FTPServer{
		rootDir:  rootDir,
		username: "admin",
		password: "123456",
		port:     "2121",
	}
}

// 设置认证凭据
func (s *FTPServer) SetCredentials(username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.username = username
	s.password = password
}

// 设置端口
func (s *FTPServer) SetPort(port string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.port = port
}

// 实现 Auth 接口
type ftpAuth struct {
	username string
	password string
}

func (auth *ftpAuth) CheckPasswd(user, pass string) (bool, error) {
	return user == auth.username && pass == auth.password, nil
}

// 自定义 FileInfo 实现
type fileInfo struct {
	os.FileInfo
}

func (f *fileInfo) Group() string {
	return "group"
}

func (f *fileInfo) Owner() string {
	return "owner"
}

func newFileInfo(info os.FileInfo) *fileInfo {
	return &fileInfo{FileInfo: info}
}

// 实现 DriverFactory 接口
type ftpDriverFactory struct {
	rootDir string
}

func (f *ftpDriverFactory) NewDriver() (server.Driver, error) {
	return &ftpDriver{rootDir: f.rootDir}, nil
}

// 实现 Driver 接口
type ftpDriver struct {
	rootDir string
	conn    *server.Conn
}

func (driver *ftpDriver) Init(conn *server.Conn) {
	driver.conn = conn
}

func (driver *ftpDriver) Stat(path string) (server.FileInfo, error) {
	info, err := os.Stat(driver.realPath(path))
	if err != nil {
		return nil, err
	}
	return newFileInfo(info), nil
}

func (driver *ftpDriver) ChangeDir(path string) error {
	if path == "/" {
		return nil
	}

	info, err := os.Stat(driver.realPath(path))
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func (driver *ftpDriver) ListDir(path string, callback func(server.FileInfo) error) error {
	dir, err := os.Open(driver.realPath(path))
	if err != nil {
		return err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := callback(newFileInfo(entry)); err != nil {
			return err
		}
	}
	return nil
}

func (driver *ftpDriver) DeleteDir(path string) error {
	return os.RemoveAll(driver.realPath(path))
}

func (driver *ftpDriver) DeleteFile(path string) error {
	return os.Remove(driver.realPath(path))
}

func (driver *ftpDriver) Rename(oldPath, newPath string) error {
	return os.Rename(driver.realPath(oldPath), driver.realPath(newPath))
}

func (driver *ftpDriver) MakeDir(path string) error {
	return os.MkdirAll(driver.realPath(path), 0755)
}

func (driver *ftpDriver) GetFile(path string, offset int64) (int64, io.ReadCloser, error) {
	file, err := os.Open(driver.realPath(path))
	if err != nil {
		return 0, nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return 0, nil, err
	}

	if offset > 0 {
		_, err = file.Seek(offset, io.SeekStart)
		if err != nil {
			file.Close()
			return 0, nil, err
		}
	}

	return info.Size() - offset, file, nil
}

func (driver *ftpDriver) PutFile(path string, data io.Reader, append bool) (int64, error) {
	var flag int
	if append {
		flag = os.O_WRONLY | os.O_APPEND | os.O_CREATE
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	file, err := os.OpenFile(driver.realPath(path), flag, 0644)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	bytes, err := io.Copy(file, data)
	if err != nil {
		return 0, err
	}

	return bytes, nil
}

func (driver *ftpDriver) realPath(path string) string {
	return filepath.Join(driver.rootDir, path)
}

// 启动 FTP 服务器
func (s *FTPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		return fmt.Errorf("server is already running")
	}

	// 设置日志
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// 每次启动时创建新的日志文件
	logPath := filepath.Join(logDir, "ftpserver.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}
	s.logFile = logFile

	// 设置日志输出到文件和控制台
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.Ldate | log.Ltime)

	// 创建驱动工厂
	factory := &ftpDriverFactory{
		rootDir: s.rootDir,
	}

	// 服务器配置
	opts := &server.ServerOpts{
		Name:    "Easy FTP Server",
		Factory: factory,
		Port: func() int {
			port, err := strconv.Atoi(s.port)
			if err != nil {
				return 2121 // 默认端口
			}
			return port
		}(),
		Hostname: "0.0.0.0",
		Auth:     &ftpAuth{username: s.username, password: s.password},
	}

	// 创建服务器
	srv := server.NewServer(opts)
	if srv == nil {
		return fmt.Errorf("failed to create server")
	}
	s.server = srv

	// 启动服务器
	log.Printf("Starting FTP server on port %s...\n", s.port)
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			log.Printf("FTP server error: %v\n", err)
		}
	}()

	return nil
}

// 停止 FTP 服务器
func (s *FTPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return fmt.Errorf("server is not running")
	}

	if err := s.server.Shutdown(); err != nil {
		return fmt.Errorf("failed to stop server: %v", err)
	}

	// 关闭日志文件
	if s.logFile != nil {
		s.logFile.Close()
		s.logFile = nil
	}

	s.server = nil
	return nil
}
