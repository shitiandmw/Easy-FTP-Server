import { useState, useEffect } from 'react';
import { StartFTPServer, StopFTPServer, OpenDirectoryDialog, LoadConfig, SaveConfig, SetAutoStart, CheckAutoStart, IsServerRunning, MinimizeToTray } from "../wailsjs/go/main/App";
import { WindowMinimise, Quit } from "../wailsjs/runtime/runtime";

function App() {
    const [config, setConfig] = useState({
        RootDir: '',
        Username: '',
        Password: '',
        Port: '',
        AutoStart: false
    });
    const [status, setStatus] = useState('');
    const [isRunning, setIsRunning] = useState(false);
    const [configLoaded, setConfigLoaded] = useState(false);

    useEffect(() => {
        if (!configLoaded) {
            const loadSavedConfig = async () => {
                try {
                    const savedConfig = await LoadConfig();
                    console.log('Saved config:', savedConfig);
                    setConfig(savedConfig);
                    
                    // 检查服务器状态
                    const running = await IsServerRunning();
                    setIsRunning(running);
                    if (running) {
                        setStatus('服务器正在运行');
                    }
                    
                    setConfigLoaded(true);
                } catch (error) {
                    console.error('Load config error:', error);
                    setStatus('加载配置失败: ' + error);
                }
            };
            loadSavedConfig();
        }
    }, [configLoaded]);

    // 当配置改变时保存
    useEffect(() => {
        if (configLoaded) {
            const saveConfig = async () => {
                try {
                    await SaveConfig(config);
                    console.log('Config saved');
                } catch (error) {
                    console.error('Save config error:', error);
                    setStatus('保存配置失败: ' + error);
                }
            };
            saveConfig();
        }
    }, [config, configLoaded]);

    const handleAutoStartChange = async (e) => {
        const checked = e.target.checked;
        try {
            await SetAutoStart(checked);
            setConfig(prev => ({ ...prev, AutoStart: checked }));
        } catch (error) {
            console.error('Set auto start error:', error);
            setStatus('设置开机启动失败: ' + error);
        }
    };

    const handleSelectFolder = async () => {
        try {
            const selectedDir = await OpenDirectoryDialog();
            if (selectedDir) {
                setConfig(prev => ({ ...prev, RootDir: selectedDir }));
            }
        } catch (error) {
            setStatus('选择目录失败: ' + error);
        }
    };

    const handleStartServer = async () => {
        try {
            await StartFTPServer(config);
            setIsRunning(true);
            setStatus('服务器已启动');
        } catch (error) {
            setStatus('启动失败: ' + error);
        }
    };

    const handleStopServer = async () => {
        try {
            await StopFTPServer();
            setIsRunning(false);
            setStatus('服务器已停止');
        } catch (error) {
            setStatus('停止失败: ' + error);
        }
    };

    const handleMinimize = async () => {
        try {
            await MinimizeToTray();
        } catch (error) {
            console.error('Minimize error:', error);
        }
    };

    return (
        <div className="h-screen bg-gradient-to-br from-blue-50 to-indigo-100 select-none">
            <div className="h-full max-w-2xl mx-auto bg-white/80 backdrop-blur-xl shadow-2xl flex flex-col">
                {/* 标题栏 */}
                <div className="bg-white/50 backdrop-blur-sm px-4 py-2 flex justify-between items-center" style={{ "--wails-draggable": "drag"}}>
                    <div className="text-gray-700 font-semibold">Easy FTP Server</div>
                    <div className="flex space-x-2">
                        <button
                            onClick={handleMinimize}
                            className="p-2 hover:bg-gray-200/50 rounded-lg transition-colors duration-150"
                        >
                            <svg className="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
                            </svg>
                        </button>
                        <button
                            onClick={Quit}
                            className="p-2 hover:bg-red-100/50 rounded-lg transition-colors duration-150"
                        >
                            <svg className="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                            </svg>
                        </button>
                    </div>
                </div>

                {/* 主要内容 */}
                <div className="flex-1 p-8 overflow-y-auto">
                    <div className="text-center mb-8">
                        <h1 className="text-2xl font-bold text-gray-800">FTP 服务器配置</h1>
                        <p className="mt-2 text-sm text-gray-600">配置您的 FTP 服务器参数</p>
                    </div>
                    
                    <div className="space-y-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">根目录</label>
                            <div className="flex space-x-2">
                                <input
                                    type="text"
                                    value={config.RootDir}
                                    onChange={(e) => setConfig(prev => ({ ...prev, RootDir: e.target.value }))}
                                    disabled={isRunning}
                                    className="flex-1 px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                    focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                    disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                    placeholder="选择 FTP 根目录"
                                />
                                <button
                                    onClick={handleSelectFolder}
                                    disabled={isRunning}
                                    className="px-4 py-2.5 bg-indigo-500 text-white rounded-xl hover:bg-indigo-600 
                                    focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 
                                    disabled:bg-gray-400 disabled:cursor-not-allowed
                                    shadow-sm transition-all duration-150 ease-in-out"
                                >
                                    浏览
                                </button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">用户名</label>
                            <input
                                type="text"
                                value={config.Username}
                                onChange={(e) => setConfig(prev => ({ ...prev, Username: e.target.value }))}
                                disabled={isRunning}
                                className="w-full px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                placeholder="设置 FTP 用户名"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">密码</label>
                            <input
                                type="password"
                                value={config.Password}
                                onChange={(e) => setConfig(prev => ({ ...prev, Password: e.target.value }))}
                                disabled={isRunning}
                                className="w-full px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                placeholder="设置 FTP 密码"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">端口</label>
                            <input
                                type="text"
                                value={config.Port}
                                onChange={(e) => setConfig(prev => ({ ...prev, Port: e.target.value }))}
                                disabled={isRunning}
                                className="w-full px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                placeholder="设置 FTP 端口"
                            />
                        </div>

                        <div className="flex items-center">
                            <input
                                type="checkbox"
                                id="autoStart"
                                checked={config.AutoStart}
                                onChange={handleAutoStartChange}
                                className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
                            />
                            <label htmlFor="autoStart" className="ml-2 block text-sm text-gray-700">
                                开机自动启动
                            </label>
                        </div>

                        <div className="">
                            {!isRunning ? (
                                <button
                                    onClick={handleStartServer}
                                    className="w-full py-3 bg-gradient-to-r from-indigo-600 to-blue-600 text-white rounded-xl 
                                    hover:from-indigo-700 hover:to-blue-700 focus:outline-none 
                                  shadow-lg transition-all duration-150 ease-in-out"
                                >
                                    启动服务器
                                </button>
                            ) : (
                                <button
                                    onClick={handleStopServer}
                                    className="w-full py-3 bg-gradient-to-r from-red-600 to-pink-600 text-white rounded-xl 
                                    hover:from-red-700 hover:to-pink-700 focus:outline-none 
                                    shadow-lg transition-all duration-150 ease-in-out"
                                >
                                    停止服务器
                                </button>
                            )}
                        </div>

                        {/* {status && (
                            <div className={`mt-4 p-4 rounded-xl ${
                                status.includes('成功') ? 'bg-green-50 text-green-700' : 
                                status.includes('失败') ? 'bg-red-50 text-red-700' : 
                                'bg-blue-50 text-blue-700'
                            }`}>
                                {status}
                            </div>
                        )} */}
                    </div>
                </div>
            </div>
        </div>
    );
}

export default App;