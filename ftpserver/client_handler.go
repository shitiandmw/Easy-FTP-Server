package ftpserver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

type ClientHandler struct {
	conn            net.Conn
	reader          *bufio.Reader
	writer          *bufio.Writer
	rootDir         string
	workDir         string
	username        string
	password        string
	authenticated   bool
	dataPort        int
	dataHost        string
	dataConn        net.Conn
	passiveListener net.Listener
	transferType    string // ASCII or BINARY
	timeout         time.Duration
	renameFrom      string // 用于存储RNFR命令的源文件路径
}

func NewClientHandler(conn net.Conn, rootDir string, username string, password string) *ClientHandler {
	return &ClientHandler{
		conn:          conn,
		reader:        bufio.NewReader(conn),
		writer:        bufio.NewWriter(conn),
		rootDir:       rootDir,
		workDir:       rootDir,
		username:      username,
		password:      password,
		authenticated: false,
		transferType:  "BINARY",        // 默认使用二进制模式
		timeout:       time.Minute * 5, // 默认5分钟超时
	}
}

func (c *ClientHandler) writeResponse(code int, message string) {
	// 将所有响应消息转换为英文，避免编码问题
	var englishMessage string
	switch message {
	case "欢迎使用 Go FTP 服务器":
		englishMessage = "Welcome to Go FTP Server"
	case "请输入密码":
		englishMessage = "Please enter password"
	case "登录成功":
		englishMessage = "Login successful"
	case "再见":
		englishMessage = "Goodbye"
	case "开始传输目录列表":
		englishMessage = "Starting directory list transfer"
	case "传输完成":
		englishMessage = "Transfer complete"
	case "连接已关闭":
		englishMessage = "Connection closed"
	case "无法建立数据连接":
		englishMessage = "Cannot establish data connection"
	case "切换到ASCII模式":
		englishMessage = "Switching to ASCII mode"
	case "切换到二进制模式":
		englishMessage = "Switching to binary mode"
	case "未知命令":
		englishMessage = "Unknown command"
	default:
		// 如果是文件路径或其他消息，保持原样
		englishMessage = message
	}

	response := fmt.Sprintf("%d %s\r\n", code, englishMessage)
	_, err := c.writer.Write([]byte(response))
	if err != nil {
		log.Printf("Failed to write response: %v\n", err)
	}
	c.writer.Flush()
}

func (c *ClientHandler) handleUser(username string) {
	c.username = username
	c.authenticated = false // 重置认证状态
	c.writeResponse(331, "Please enter password")
}

func (c *ClientHandler) handlePass(password string) {
	// 验证用户名和密码
	if c.username == "" {
		c.writeResponse(503, "Login with USER first")
		return
	}

	if c.password == "" {
		// 如果没有设置密码，允许任何密码
		c.authenticated = true
		c.writeResponse(230, "Login successful")
		return
	}

	if password == c.password {
		c.authenticated = true
		c.writeResponse(230, "Login successful")
	} else {
		c.authenticated = false
		c.writeResponse(530, "Invalid username or password")
	}
}

func (c *ClientHandler) checkAuth() bool {
	if !c.authenticated {
		c.writeResponse(530, "Please login with USER and PASS")
		return false
	}
	return true
}

func (c *ClientHandler) Handle() {
	defer c.conn.Close()

	c.writeResponse(220, "Welcome to Go FTP Server")

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Failed to read command: %v\n", err)
			}
			return
		}

		line = strings.TrimSpace(line)
		cmd := strings.Fields(line)
		if len(cmd) == 0 {
			continue
		}

		command := strings.ToUpper(cmd[0])
		params := ""
		if len(cmd) > 1 {
			params = strings.Join(cmd[1:], " ")
		}

		log.Printf("Received command: %s %s\n", command, params)

		// 这些命令不需要认证
		if command == "USER" || command == "PASS" || command == "QUIT" {
			switch command {
			case "USER":
				c.handleUser(params)
			case "PASS":
				c.handlePass(params)
			case "QUIT":
				c.writeResponse(221, "Goodbye")
				return
			}
			continue
		}

		// 其他所有命令都需要认证
		if !c.checkAuth() {
			continue
		}

		switch command {
		case "PWD":
			c.handlePwd()
		case "CWD":
			c.handleCwd(params)
		case "LIST":
			c.handleList()
		case "PORT":
			c.handlePort(params)
		case "STOR":
			c.handleStor(params)
		case "RETR":
			c.handleRetr(params)
		case "TYPE":
			c.handleType(params)
		case "PASV":
			c.handlePasv()
		case "EPSV":
			c.handleEpsv()
		case "EPRT":
			c.handleEprt(params)
		case "SIZE":
			c.handleSize(params)
		case "MDTM":
			c.handleMdtm(params)
		case "DELE":
			c.handleDele(params)
		case "RNFR":
			c.handleRnfr(params)
		case "RNTO":
			c.handleRnto(params)
		default:
			c.writeResponse(500, "Unknown command")
		}
	}
}

func (c *ClientHandler) handlePwd() {
	workDir := strings.TrimPrefix(c.workDir, c.rootDir)
	if workDir == "" {
		workDir = "/"
	}
	workDir = filepath.ToSlash(workDir)
	c.writeResponse(257, fmt.Sprintf("\"%s\" is current directory", workDir))
}

func (c *ClientHandler) handleCwd(path string) {
	path = filepath.FromSlash(path)
	newPath := filepath.Join(c.workDir, path)
	newPath = filepath.Clean(newPath)

	if !strings.HasPrefix(newPath, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	info, err := os.Stat(newPath)
	if err != nil || !info.IsDir() {
		c.writeResponse(550, "Directory does not exist")
		return
	}

	c.workDir = newPath
	c.writeResponse(250, "Directory changed")
}

func (c *ClientHandler) handleList() {
	if c.dataConn == nil {
		if c.passiveListener != nil {
			// 等待数据连接建立，增加等待时间至15秒
			deadline := time.Now().Add(time.Second * 15)
			for c.dataConn == nil && time.Now().Before(deadline) {
				time.Sleep(100 * time.Millisecond)
			}
			if c.dataConn == nil {
				c.writeResponse(425, "Cannot establish data connection")
				return
			}
		} else {
			if err := c.openDataConn(); err != nil {
				c.writeResponse(425, "Cannot establish data connection")
				return
			}
		}
	}

	// 创建一个函数来清理连接
	cleanup := func() {
		if c.dataConn != nil {
			c.dataConn.Close()
			c.dataConn = nil
		}
		if c.passiveListener != nil {
			c.passiveListener.Close()
			c.passiveListener = nil
		}
	}
	defer cleanup()

	files, err := os.ReadDir(c.workDir)
	if err != nil {
		c.writeResponse(550, "Cannot list directory")
		return
	}

	c.writeResponse(150, "Starting directory list transfer")

	// 使用带缓冲的写入器
	writer := bufio.NewWriter(c.dataConn)

	// 设置数据连接超时
	c.dataConn.SetDeadline(time.Now().Add(time.Minute))

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		// 使用标准格式输出文件信息，将文件名转换为 GBK 编码
		size := info.Size()
		date := info.ModTime().Format("Jan _2 15:04")
		name := utf8ToGBK(info.Name())
		perms := info.Mode().String()

		line := fmt.Sprintf("%s %8d %s %s\r\n", perms, size, date, name)
		_, err = writer.WriteString(line)
		if err != nil {
			log.Printf("Failed to write directory list: %v\n", err)
			c.writeResponse(426, "Connection closed")
			c.dataConn.Close()
			c.dataConn = nil
			return
		}
	}

	// 刷新缓冲区
	err = writer.Flush()
	if err != nil {
		log.Printf("Failed to flush directory list: %v\n", err)
		c.writeResponse(426, "Connection closed")
	} else {
		// 等待数据完全发送
		time.Sleep(100 * time.Millisecond)
		c.writeResponse(226, "Transfer complete")
	}

	// 关闭数据连接
	c.dataConn.Close()
	c.dataConn = nil
}

func (c *ClientHandler) handlePort(params string) {
	nums := strings.Split(params, ",")
	if len(nums) != 6 {
		c.writeResponse(500, "Invalid PORT command")
		return
	}

	// 解析IP地址和端口
	portPart1, _ := strconv.Atoi(nums[4])
	portPart2, _ := strconv.Atoi(nums[5])
	c.dataPort = portPart1*256 + portPart2
	c.dataHost = strings.Join(nums[0:4], ".")

	c.writeResponse(200, "PORT command successful")
}

func (c *ClientHandler) handleType(params string) {
	switch strings.ToUpper(params) {
	case "A":
		c.transferType = "ASCII"
		c.writeResponse(200, "Switching to ASCII mode")
	case "I", "L", "L 8":
		// 所有二进制模式都使用相同的处理方式
		c.transferType = "BINARY"
		c.writeResponse(200, "Switching to binary mode")
	default:
		c.writeResponse(504, "Unsupported transfer type")
	}
}

func (c *ClientHandler) openDataConn() error {
	if c.dataPort <= 0 {
		return fmt.Errorf("未设置数据端口")
	}

	// 设置连接超时
	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", c.dataHost, c.dataPort))
	if err != nil {
		log.Printf("数据连接失败: %v\n", err)
		return err
	}

	c.dataConn = conn
	return nil
}

func (c *ClientHandler) handlePasv() {
	// 创建监听器，指定超时
	config := &net.ListenConfig{
		KeepAlive: time.Minute,
	}
	listener, err := config.Listen(context.Background(), "tcp", ":0")
	if err != nil {
		c.writeResponse(425, "Cannot enter passive mode")
		return
	}

	c.passiveListener = listener

	// 获取服务器IP地址
	host, _, _ := net.SplitHostPort(c.conn.LocalAddr().String())
	if host == "::" || host == "0.0.0.0" || strings.HasPrefix(host, "127.") {
		// 如果是本地地址，尝试获取一个可用的外部地址
		addrs, err := net.InterfaceAddrs()
		if err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					host = ipnet.IP.String()
					break
				}
			}
		}
	}

	addr := listener.Addr().(*net.TCPAddr)
	port1 := addr.Port / 256
	port2 := addr.Port % 256

	// 替换IP地址中的点为逗号
	hostParts := strings.Split(host, ".")
	c.writeResponse(227, fmt.Sprintf("Entering passive mode (%s,%d,%d)", strings.Join(hostParts, ","), port1, port2))

	// 在新的goroutine中等待连接
	go func() {
		defer listener.Close()
		// 设置接受连接的超时时间
		listener.(*net.TCPListener).SetDeadline(time.Now().Add(c.timeout))
		dataConn, err := listener.Accept()
		if err != nil {
			log.Printf("接受被动连接失败: %v\n", err)
			return
		}
		c.dataConn = dataConn
	}()

	// 等待数据连接建立或超时
	deadline := time.Now().Add(c.timeout)
	for c.dataConn == nil && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}

	if c.dataConn == nil {
		c.writeResponse(425, "Establishing data connection timed out")
		return
	}
}

func (c *ClientHandler) handleEpsv() {
	// 创建监听器，指定超时
	config := &net.ListenConfig{
		KeepAlive: time.Minute,
	}
	listener, err := config.Listen(context.Background(), "tcp", ":0")
	if err != nil {
		c.writeResponse(425, "Cannot enter extended passive mode")
		return
	}

	c.passiveListener = listener

	addr := listener.Addr().(*net.TCPAddr)
	// EPSV 响应格式: 229 Entering Extended Passive Mode (|||port|)
	c.writeResponse(229, fmt.Sprintf("Entering Extended Passive Mode (|||%d|)", addr.Port))

	// 在新的goroutine中等待连接
	go func() {
		defer listener.Close()
		// 设置接受连接的超时时间
		listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second * 30))
		dataConn, err := listener.Accept()
		if err != nil {
			log.Printf("接受扩展被动连接失败: %v\n", err)
			return
		}
		c.dataConn = dataConn
	}()
}

func (c *ClientHandler) handleEprt(params string) {
	// EPRT 格式: |{协议}|{IP}|{端口}|
	// 例如: |2|::1|1234| 表示 IPv6
	parts := strings.Split(params, "|")
	if len(parts) != 5 || parts[0] != "" || parts[4] != "" {
		c.writeResponse(500, "Invalid EPRT command")
		return
	}

	proto := parts[1]
	host := parts[2]
	port, err := strconv.Atoi(parts[3])
	if err != nil {
		c.writeResponse(500, "Invalid port number")
		return
	}

	// 只支持 IPv4 (1) 和 IPv6 (2)
	if proto != "1" && proto != "2" {
		c.writeResponse(522, "Unsupported network protocol")
		return
	}

	c.dataHost = host
	c.dataPort = port

	c.writeResponse(200, "EPRT command successful")
}

func (c *ClientHandler) handleSize(params string) {
	if params == "" {
		c.writeResponse(501, "Missing parameter")
		return
	}

	path := filepath.Clean(filepath.Join(c.workDir, params))
	if !strings.HasPrefix(path, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.writeResponse(550, "File does not exist")
		} else {
			c.writeResponse(550, "Cannot get file information")
		}
		return
	}

	if stat.IsDir() {
		c.writeResponse(550, "Not a file")
		return
	}

	c.writeResponse(213, fmt.Sprintf("%d", stat.Size()))
}

func (c *ClientHandler) handleMdtm(params string) {
	if params == "" {
		c.writeResponse(501, "Missing parameter")
		return
	}

	path := filepath.Clean(filepath.Join(c.workDir, params))
	if !strings.HasPrefix(path, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.writeResponse(550, "File does not exist")
		} else {
			c.writeResponse(550, "Cannot get file information")
		}
		return
	}

	if stat.IsDir() {
		c.writeResponse(550, "Not a file")
		return
	}

	// 返回格式：YYYYMMDDHHMMSS
	modTime := stat.ModTime().UTC().Format("20060102150405")
	c.writeResponse(213, modTime)
}

func (c *ClientHandler) handleStor(params string) {
	if params == "" {
		c.writeResponse(500, "No file name specified")
		return
	}

	// 将文件名从 GBK 转换为 UTF-8
	filename := gbkToUTF8(params)
	filename = filepath.Clean(filepath.Join(c.workDir, filename))
	if !strings.HasPrefix(filename, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	// 检查文件扩展名，确定是否为二进制文件
	ext := strings.ToLower(filepath.Ext(filename))
	binaryExts := map[string]bool{
		".xlsx": true, ".xls": true, ".doc": true, ".docx": true,
		".pdf": true, ".zip": true, ".rar": true, ".7z": true,
		".exe": true, ".dll": true, ".jpg": true, ".jpeg": true,
		".png": true, ".gif": true, ".bmp": true, ".mp3": true,
		".mp4": true, ".avi": true, ".mov": true,
	}

	// 如果是二进制文件，强制使用二进制模式
	if binaryExts[ext] && c.transferType != "BINARY" {
		log.Printf("Forcing binary mode for file type: %s\n", ext)
		c.transferType = "BINARY"
	}

	if c.passiveListener == nil && c.dataPort > 0 {
		if err := c.openDataConn(); err != nil {
			c.writeResponse(425, "Cannot open data connection")
			return
		}
	}

	if c.dataConn == nil {
		// 等待数据连接建立
		deadline := time.Now().Add(time.Second * 5)
		for c.dataConn == nil && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		if c.dataConn == nil {
			c.writeResponse(425, "No data connection established")
			return
		}
	}

	// 创建一个函数来清理连接
	cleanup := func() {
		if c.dataConn != nil {
			c.dataConn.Close()
			c.dataConn = nil
		}
		if c.passiveListener != nil {
			c.passiveListener.Close()
			c.passiveListener = nil
		}
	}
	defer cleanup()

	// 设置数据连接超时
	c.dataConn.SetDeadline(time.Now().Add(time.Minute * 5))

	// 以二进制模式打开文件
	var flag int = os.O_CREATE | os.O_WRONLY
	if c.transferType == "BINARY" {
		flag |= os.O_TRUNC // 二进制模式下截断文件
	}
	file, err := os.OpenFile(filename, flag, 0666)
	if err != nil {
		log.Printf("Failed to create file: %v\n", err)
		c.writeResponse(550, "Cannot create file")
		return
	}
	defer file.Close()

	c.writeResponse(150, "Starting file transfer")

	var n int64
	if c.transferType == "ASCII" {
		// ASCII模式：处理行结束符
		reader := bufio.NewReader(c.dataConn)
		writer := bufio.NewWriter(file)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				log.Printf("Failed to read data: %v\n", err)
				c.writeResponse(550, "File transfer failed")
				return
			}
			n += int64(len(line))
			if err == io.EOF {
				if len(line) > 0 {
					writer.WriteString(line)
				}
				break
			}
			// 统一行结束符为系统默认
			line = strings.TrimRight(line, "\r\n") + "\n"
			writer.WriteString(line)
		}
		writer.Flush()
	} else {
		// 二进制模式：使用大缓冲区直接复制
		buf := make([]byte, 1024*1024) // 1MB 缓冲区
		n, err = io.CopyBuffer(file, c.dataConn, buf)
		if err != nil {
			log.Printf("File transfer failed: %v\n", err)
			c.writeResponse(550, "File transfer failed")
			return
		}
	}

	c.writeResponse(226, fmt.Sprintf("Transfer complete, %d bytes received", n))
}

func (c *ClientHandler) handleRetr(params string) {
	if params == "" {
		c.writeResponse(500, "No file name specified")
		return
	}

	// 将文件名从 GBK 转换为 UTF-8
	filename := gbkToUTF8(params)
	filename = filepath.Clean(filepath.Join(c.workDir, filename))
	if !strings.HasPrefix(filename, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	// 检查文件是否存在
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			c.writeResponse(550, "File not found")
		} else {
			c.writeResponse(550, "Cannot access file")
		}
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		c.writeResponse(550, "Cannot get file information")
		return
	}

	if fileInfo.IsDir() {
		c.writeResponse(550, "Cannot download directory")
		return
	}

	// 检查文件扩展名，确定是否为二进制文件
	ext := strings.ToLower(filepath.Ext(filename))
	binaryExts := map[string]bool{
		".xlsx": true, ".xls": true, ".doc": true, ".docx": true,
		".pdf": true, ".zip": true, ".rar": true, ".7z": true,
		".exe": true, ".dll": true, ".jpg": true, ".jpeg": true,
		".png": true, ".gif": true, ".bmp": true, ".mp3": true,
		".mp4": true, ".avi": true, ".mov": true,
	}

	// 如果是二进制文件，强制使用二进制模式
	if binaryExts[ext] && c.transferType != "BINARY" {
		log.Printf("Forcing binary mode for file type: %s\n", ext)
		c.transferType = "BINARY"
	}

	if c.passiveListener == nil && c.dataPort > 0 {
		if err := c.openDataConn(); err != nil {
			c.writeResponse(425, "Cannot open data connection")
			return
		}
	}

	if c.dataConn == nil {
		// 等待数据连接建立
		deadline := time.Now().Add(time.Second * 5)
		for c.dataConn == nil && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		if c.dataConn == nil {
			c.writeResponse(425, "No data connection established")
			return
		}
	}
	defer c.dataConn.Close()

	// 设置数据连接超时
	c.dataConn.SetDeadline(time.Now().Add(time.Minute * 5))

	c.writeResponse(150, fmt.Sprintf("Opening %s mode data connection for %s (%d bytes)",
		c.transferType, filepath.Base(filename), fileInfo.Size()))

	var n int64
	if c.transferType == "ASCII" {
		// ASCII模式：逐行读取并转换行结束符
		reader := bufio.NewReader(file)
		writer := bufio.NewWriter(c.dataConn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				log.Printf("Failed to read file: %v\n", err)
				c.writeResponse(550, "File transfer failed")
				return
			}
			if len(line) > 0 {
				// 确保使用 CRLF 作为行结束符
				line = strings.TrimRight(line, "\r\n") + "\r\n"
				_, err = writer.WriteString(line)
				if err != nil {
					log.Printf("Failed to write data: %v\n", err)
					c.writeResponse(550, "File transfer failed")
					return
				}
				n += int64(len(line))
			}
			if err == io.EOF {
				break
			}
		}
		err = writer.Flush()
		if err != nil {
			log.Printf("Failed to flush data: %v\n", err)
			c.writeResponse(550, "File transfer failed")
			return
		}
	} else {
		// 二进制模式：使用大缓冲区直接复制
		buf := make([]byte, 1024*1024) // 1MB 缓冲区
		n, err = io.CopyBuffer(c.dataConn, file, buf)
		if err != nil {
			log.Printf("File transfer failed: %v\n", err)
			c.writeResponse(550, "File transfer failed")
			return
		}
	}

	c.writeResponse(226, fmt.Sprintf("Transfer complete, %d bytes sent", n))
}

func (c *ClientHandler) handleDele(params string) {
	if params == "" {
		c.writeResponse(501, "Missing parameter")
		return
	}

	// 将GBK编码的文件名转换为UTF-8编码
	params = gbkToUTF8(params)
	path := filepath.Clean(filepath.Join(c.workDir, params))
	if !strings.HasPrefix(path, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.writeResponse(550, "File not found")
		return
	}

	// 删除文件
	err := os.Remove(path)
	if err != nil {
		c.writeResponse(550, "Could not delete file: "+err.Error())
		return
	}

	c.writeResponse(250, "File deleted")
	c.writeResponse(226, "ABOR command successful")
}

func (c *ClientHandler) handleRnfr(params string) {
	if params == "" {
		c.writeResponse(501, "Missing parameter")
		return
	}

	// 将GBK编码的文件名转换为UTF-8编码
	params = gbkToUTF8(params)
	path := filepath.Clean(filepath.Join(c.workDir, params))
	if !strings.HasPrefix(path, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.writeResponse(550, "File not found")
		return
	}

	c.renameFrom = path
	c.writeResponse(350, "Ready for destination name")
}

func (c *ClientHandler) handleRnto(params string) {
	if params == "" {
		c.writeResponse(501, "Missing parameter")
		return
	}

	if c.renameFrom == "" {
		c.writeResponse(503, "Bad sequence of commands")
		return
	}

	// 将GBK编码的文件名转换为UTF-8编码
	params = gbkToUTF8(params)
	newPath := filepath.Clean(filepath.Join(c.workDir, params))
	if !strings.HasPrefix(newPath, c.rootDir) {
		c.writeResponse(550, "Access denied")
		return
	}

	// 执行重命名
	err := os.Rename(c.renameFrom, newPath)
	if err != nil {
		c.writeResponse(550, "Rename failed: "+err.Error())
		return
	}

	c.renameFrom = "" // 清除重命名源路径
	c.writeResponse(250, "Rename successful")
}

// 转换 UTF-8 到 GBK
func utf8ToGBK(text string) string {
	encoder := simplifiedchinese.GBK.NewEncoder()
	gbkBytes, err := encoder.Bytes([]byte(text))
	if err != nil {
		log.Printf("转换到GBK失败: %v\n", err)
		return text
	}
	return string(gbkBytes)
}

// 转换 GBK 到 UTF-8
func gbkToUTF8(text string) string {
	decoder := simplifiedchinese.GBK.NewDecoder()
	utf8Bytes, err := decoder.Bytes([]byte(text))
	if err != nil {
		log.Printf("转换到UTF8失败: %v\n", err)
		return text
	}
	return string(utf8Bytes)
}
