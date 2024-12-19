import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { StartFTPServer, StopFTPServer, OpenDirectoryDialog, LoadConfig, SaveConfig, SetAutoStart, CheckAutoStart, IsServerRunning, MinimizeToTray } from "../wailsjs/go/main/App";
import { WindowMinimise, Quit } from "../wailsjs/runtime/runtime";
import LanguageSwitch from './components/LanguageSwitch';
import './i18n';

function App() {
    const { t } = useTranslation();
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
                        setStatus(t('messages.serverStarted'));
                    }
                    
                    setConfigLoaded(true);
                } catch (error) {
                    console.error('Load config error:', error);
                    setStatus(t('messages.loadConfigError') + ': ' + error);
                }
            };
            loadSavedConfig();
        }
    }, [configLoaded, t]);

    // 当配置改变时保存
    useEffect(() => {
        if (configLoaded) {
            const saveConfig = async () => {
                try {
                    await SaveConfig(config);
                    console.log('Config saved');
                } catch (error) {
                    console.error('Save config error:', error);
                    setStatus(t('messages.saveConfigError') + ': ' + error);
                }
            };
            saveConfig();
        }
    }, [config, configLoaded, t]);

    const handleAutoStartChange = async (e) => {
        const checked = e.target.checked;
        try {
            await SetAutoStart(checked);
            setConfig(prev => ({ ...prev, AutoStart: checked }));
        } catch (error) {
            console.error('Set auto start error:', error);
            setStatus(t('messages.setAutoStartError') + ': ' + error);
        }
    };

    const handleSelectFolder = async () => {
        try {
            const selectedDir = await OpenDirectoryDialog();
            if (selectedDir) {
                setConfig(prev => ({ ...prev, RootDir: selectedDir }));
            }
        } catch (error) {
            setStatus(t('messages.selectDirError') + ': ' + error);
        }
    };

    const handleStartServer = async () => {
        try {
            await StartFTPServer(config);
            setIsRunning(true);
            setStatus(t('messages.serverStarted'));
        } catch (error) {
            setStatus(t('messages.startServerError') + ': ' + error);
        }
    };

    const handleStopServer = async () => {
        try {
            await StopFTPServer();
            setIsRunning(false);
            setStatus(t('messages.serverStopped'));
        } catch (error) {
            setStatus(t('messages.stopServerError') + ': ' + error);
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
                    <div className="text-gray-700 font-semibold">{t('title')}</div>
                    <div className="flex items-center space-x-2">
                        <LanguageSwitch />
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
                        <h1 className="text-2xl font-bold text-gray-800">{t('title')}</h1>
                        <p className="mt-2 text-sm text-gray-600">{t('messages.configureServer')}</p>
                    </div>
                    
                    <div className="space-y-6">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">{t('settings.rootDir')}</label>
                            <div className="flex space-x-2">
                                <input
                                    type="text"
                                    value={config.RootDir}
                                    onChange={(e) => setConfig(prev => ({ ...prev, RootDir: e.target.value }))}
                                    disabled={isRunning}
                                    className="flex-1 px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                    focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                    disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                    placeholder={t('tooltips.rootDir')}
                                />
                                <button
                                    onClick={handleSelectFolder}
                                    disabled={isRunning}
                                    className="px-4 py-2.5 bg-indigo-500 text-white rounded-xl hover:bg-indigo-600 
                                    focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 
                                    disabled:bg-gray-400 disabled:cursor-not-allowed
                                    shadow-sm transition-all duration-150 ease-in-out"
                                >
                                    {t('settings.selectDir')}
                                </button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">{t('settings.username')}</label>
                            <input
                                type="text"
                                value={config.Username}
                                onChange={(e) => setConfig(prev => ({ ...prev, Username: e.target.value }))}
                                disabled={isRunning}
                                className="w-full px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                placeholder={t('settings.username')}
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">{t('settings.password')}</label>
                            <input
                                type="password"
                                value={config.Password}
                                onChange={(e) => setConfig(prev => ({ ...prev, Password: e.target.value }))}
                                disabled={isRunning}
                                className="w-full px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                placeholder={t('settings.password')}
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1.5">{t('settings.port')}</label>
                            <input
                                type="text"
                                value={config.Port}
                                onChange={(e) => setConfig(prev => ({ ...prev, Port: e.target.value }))}
                                disabled={isRunning}
                                className="w-full px-4 py-2.5 bg-white text-gray-900 border border-gray-300 rounded-xl text-sm shadow-sm placeholder-gray-400
                                focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500
                                disabled:bg-gray-50 disabled:text-gray-500 transition-colors"
                                placeholder={t('tooltips.port')}
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
                                {t('settings.autoStart')}
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
                                    {t('settings.start')}
                                </button>
                            ) : (
                                <button
                                    onClick={handleStopServer}
                                    className="w-full py-3 bg-gradient-to-r from-red-600 to-pink-600 text-white rounded-xl 
                                    hover:from-red-700 hover:to-pink-700 focus:outline-none 
                                    shadow-lg transition-all duration-150 ease-in-out"
                                >
                                    {t('settings.stop')}
                                </button>
                            )}
                        </div>

                        {status && (
                            <div className={`mt-4 p-4 rounded-xl ${
                                status.includes(t('messages.serverStarted')) ? 'bg-green-50 text-green-700' : 
                                status.includes(t('messages.serverStopped')) ? 'bg-red-50 text-red-700' : 
                                'bg-blue-50 text-blue-700'
                            }`}>
                                {status}
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}

export default App;