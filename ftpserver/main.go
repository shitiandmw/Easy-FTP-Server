package ftpserver

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	DEFAULT_PORT = ":2121"
)

type FTPServer struct {
	listener   net.Listener
	rootDir    string
	port       string
	username   string
	password   string
	clientConn map[string]*ClientHandler
}

func NewFTPServer(rootDir string) *FTPServer {
	// 确保路径使用正确的分隔符
	rootDir = filepath.Clean(rootDir)
	return &FTPServer{
		rootDir:    rootDir,
		port:       DEFAULT_PORT, // 默认端口
		clientConn: make(map[string]*ClientHandler),
	}
}

func (s *FTPServer) SetPort(port string) {
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	s.port = port
}

func (s *FTPServer) SetCredentials(username, password string) {
	s.username = username
	s.password = password
}

func (s *FTPServer) Start() error {
	// 创建日志文件
	logFile, err := os.OpenFile("ftpserver.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
	}

	listener, err := net.Listen("tcp", s.port)
	if err != nil {
		return err
	}
	s.listener = listener

	log.Printf("FTP 服务器启动在端口%s\n", s.port)
	log.Printf("根目录: %s\n", s.rootDir)

	// 在新的 goroutine 中接受连接
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					log.Printf("接受连接错误: %v\n", err)
				}
				return
			}

			clientHandler := NewClientHandler(conn, s.rootDir, s.username, s.password)
			s.clientConn[conn.RemoteAddr().String()] = clientHandler
			go clientHandler.Handle()
		}
	}()

	return nil
}

func (s *FTPServer) Stop() error {
	if s.listener != nil {
		err := s.listener.Close()
		if err != nil {
			return err
		}

		// 关闭所有客户端连接
		for _, handler := range s.clientConn {
			handler.conn.Close()
		}
		s.clientConn = make(map[string]*ClientHandler)
	}
	return nil
}

func main() {
	// 使用可执行文件所在目录作为 FTP 根目录
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	rootDir := filepath.Dir(exePath)

	server := NewFTPServer(rootDir)
	server.SetPort(":2222")
	server.SetCredentials("username", "password")

	// 打印启动信息到控制台
	println("FTP 服务器正在启动...")
	println("根目录:", rootDir)
	println("端口:", server.port)
	println("日志文件: ftpserver.log")

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
