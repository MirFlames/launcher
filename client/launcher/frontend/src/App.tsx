import {useState, useEffect, useCallback} from 'react';
import './App.css';
import {AuthRefreshSession, AuthStartLogin, AuthLogout, PlayMinecraft} from "../wailsjs/go/main/App";
import {Quit, WindowMinimise, EventsOn} from "../wailsjs/runtime/runtime";
import {SettingsModal} from './components/SettingsModal';
import {NewsFeed} from './components/NewsFeed';
import {ProgressOverlay} from './components/ProgressOverlay';

function App() {
    const [authenticated, setAuthenticated] = useState(false);
    const [nickname, setNickname] = useState('');
    const [settingsOpen, setSettingsOpen] = useState(false);
    const [newsKey, setNewsKey] = useState(0);
    const [progress, setProgress] = useState({visible: false, title: '', description: ''});
    const [authError, setAuthError] = useState('');
    const [playError, setPlayError] = useState('');

    function refreshAuth() {
        AuthRefreshSession()
            .then((s) => {
                setAuthenticated(s != null);
                setNickname(s?.nickname || '');
            })
            .catch(() => {
                setAuthenticated(false);
                setNickname('');
            });
    }

    useEffect(() => {
        refreshAuth();
    }, []);

    useEffect(() => {
        const unProgress = EventsOn('launch-progress', (ev: { stage?: string; status?: string; progress?: number }) => {
            setProgress((p) => ({
                ...p,
                visible: true,
                title: ev?.stage ?? p.title,
                description: ev?.status ?? p.description,
            }));
        });
        const unEnded = EventsOn('launch-ended', () => {
            setProgress({visible: false, title: '', description: ''});
        });
        return () => { unProgress?.(); unEnded?.(); };
    }, []);

    const handlePlay = useCallback(() => {
        setPlayError('');
        setProgress({visible: true, title: 'Подготовка', description: 'Запуск...'});
        PlayMinecraft()
            .then(() => {
                // Не скрываем оверлей — он остаётся с «Ожидание окна Minecraft...» до появления окна
            })
            .catch((err) => {
                setProgress({visible: false, title: '', description: ''});
                const msg = (err?.message || (typeof err === 'string' ? err : err?.toString?.() || String(err))).trim();
                setPlayError(msg || 'Ошибка запуска. Проверьте настройки и подключение.');
            });
    }, []);

    function handleLogin() {
        setAuthError('');
        setProgress({visible: true, title: 'Вход через Telegram', description: 'Откройте бота и введите код. Ожидание...'});
        AuthStartLogin()
            .then((session) => {
                setProgress({visible: false, title: '', description: ''});
                if (session) {
                    setAuthenticated(true);
                    setNickname(session.nickname);
                    setNewsKey((k) => k + 1); // перезагрузить новости с учётом сессии
                }
            })
            .catch((err) => {
                setProgress({visible: false, title: '', description: ''});
                const msg = (err?.message || (typeof err === 'string' ? err : err?.toString?.() || String(err))).trim();
                setAuthError(msg || 'Ошибка входа. Проверьте URL бэкенда в настройках.');
            });
    }

    function handleLogout() {
        AuthLogout()
            .then(() => {
                setAuthenticated(false);
                setNickname('');
                setNewsKey((k) => k + 1); // обновить новости (показать призыв войти)
            })
            .catch(() => {
                setAuthenticated(false);
                setNickname('');
            });
    }

    return (
        <div id="App">
            <header className="titlebar">
                <span className="titlebar-title">Launcher</span>
                <div className="titlebar-buttons">
                    <button className="titlebar-btn settings" onClick={() => setSettingsOpen(true)} title="Настройки">
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <circle cx="12" cy="12" r="3"/>
                            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
                        </svg>
                    </button>
                    <button className="titlebar-btn minimize" onClick={WindowMinimise} title="Свернуть">−</button>
                    <button className="titlebar-btn close" onClick={Quit} title="Закрыть">×</button>
                </div>
            </header>
            <main className="content">
                <div className="auth-block">
                    {authenticated ? (
                        <>
                            <p className="auth-nickname">{nickname}</p>
                            <div className="auth-buttons">
                                <button className="btn btn-play" onClick={handlePlay}>Играть</button>
                                <button className="btn btn-logout" onClick={handleLogout}>Выйти</button>
                            </div>
                            {playError && <p className="auth-error">{playError}</p>}
                        </>
                    ) : (
                        <>
                            <button className="btn btn-login" onClick={handleLogin}>Войти</button>
                            {authError && <p className="auth-error">{authError}</p>}
                        </>
                    )}
                </div>
                <NewsFeed key={newsKey} />
            </main>
            <SettingsModal
                isOpen={settingsOpen}
                onClose={() => setSettingsOpen(false)}
                onSaved={() => { setNewsKey((k) => k + 1); refreshAuth(); }}
            />
            <ProgressOverlay
                visible={progress.visible}
                title={progress.title}
                description={progress.description}
            />
        </div>
    )
}

export default App
