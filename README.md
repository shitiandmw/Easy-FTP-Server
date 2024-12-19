# 项目说明

基础的Go桌面应用，使用Wails框架开发。
## 技术栈

本项目使用以下技术栈：
- [Wails](https://wails.io/) - Go桌面应用开发框架
- [React](https://react.dev/) - 前端UI框架
- [Vite](https://vitejs.dev/) - 前端构建工具
- [Tailwind CSS](https://tailwindcss.com/) - CSS框架

## 环境要求

在开始之前，请确保已安装以下工具：
- Go 1.18+
- Node.js 14+
- npm 或 yarn
- Wails CLI

## 开发说明

### 项目启动

1. 安装依赖
```bash
# 前端依赖安装
cd frontend
npm install  # 或 yarn

# 回到项目根目录
cd ..
```

2. 开发模式启动
```bash
wails dev
```

此命令会启动：
- Vite 开发服务器，支持前端代码热重载
- 后端服务，地址为 http://localhost:34115，可在浏览器中调试Go方法

### 项目测试

```bash
# 运行前端测试
cd frontend
npm test  # 或 yarn test

# 运行后端测试
go test ./...
```

### 项目编译

1. 开发环境编译
```bash
wails build
```

2. 生产环境编译
```bash
# Windows
wails build -platform windows/amd64 -clean

# MacOS
wails build -platform darwin/universal -clean

# Linux
wails build -platform linux/amd64 -clean
```

编译后的文件将输出到 `build` 目录。

## 项目配置

- 项目配置文件：`wails.json`
- 更多配置说明请参考：https://wails.io/docs/reference/project-config

## 目录结构

```
.
├── frontend/          # 前端代码目录
│   ├── src/          # React源码
│   ├── public/       # 静态资源
│   └── package.json  # 前端依赖配置
├── app.go            # 后端入口
└── main.go          # 主程序入口
