import {useState, useEffect, useCallback} from 'react';
import './App.css';
import {AuthRefreshSession, AuthStartLogin, AuthLogout, PlayMinecraft} from "../wailsjs/go/main/App";
import {Quit, WindowMinimise, EventsOn} from "../wailsjs/runtime/runtime";
import {SettingsModal} from './components/SettingsModal';
import {NewsFeed} from './components/NewsFeed';
import {ProgressOverlay} from './components/ProgressOverlay';
import {GrassBlockScene} from './components/GrassBlockScene';
import {UpdateModal, LauncherUpdateInfo} from './components/UpdateModal';

function App() {
    const [authenticated, setAuthenticated] = useState(false);
    const [nickname, setNickname] = useState('');
    const [settingsOpen, setSettingsOpen] = useState(false);
    const [newsKey, setNewsKey] = useState(0);
    const [progress, setProgress] = useState({visible: false, title: '', description: ''});
    const [authError, setAuthError] = useState('');
    const [playError, setPlayError] = useState('');
    const [playHovered, setPlayHovered] = useState(false);
    const [updateInfo, setUpdateInfo] = useState<LauncherUpdateInfo | null>(null);
    const [updateVisible, setUpdateVisible] = useState(false);
    const [updateInProgress, setUpdateInProgress] = useState(false);
    const [updateError, setUpdateError] = useState('');
    const [updateCheckDone, setUpdateCheckDone] = useState(false);

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

    // Проверка обновления лаунчера при старте.
    useEffect(() => {
        const w = window as any;
        const api = w?.go?.main?.App;
        if (!api || typeof api.CheckLauncherUpdate !== 'function') {
            setUpdateCheckDone(true);
            return;
        }
        api.CheckLauncherUpdate()
            .then((info: LauncherUpdateInfo | null) => {
                if (info && info.version && info.version !== info.current_version) {
                    setUpdateInfo(info);
                    setUpdateVisible(true);
                }
                setUpdateCheckDone(true);
            })
            .catch(() => {
                // Тихо игнорируем ошибки проверки обновления
                setUpdateCheckDone(true);
            });
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

    function handleUpdateNow() {
        if (!updateInfo) return;
        setUpdateError('');
        setUpdateInProgress(true);
        const w = window as any;
        const api = w?.go?.main?.App;
        if (!api || typeof api.ApplyLauncherUpdate !== 'function') {
            setUpdateError('Механизм автообновления недоступен.');
            setUpdateInProgress(false);
            return;
        }
        api.ApplyLauncherUpdate()
            .then(() => {
                setUpdateInProgress(false);
                // После успешного старта нового лаунчера пробуем аккуратно закрыть текущий.
                Quit();
            })
            .catch((err: any) => {
                setUpdateInProgress(false);
                const msg = (err?.message || (typeof err === 'string' ? err : err?.toString?.() || String(err))).trim();
                setUpdateError(msg || 'Не удалось установить обновление.');
            });
    }

    function handleUpdateSkip() {
        setUpdateVisible(false);
    }

    return (
        <div id="App">
            <header className="titlebar">
                <span className="titlebar-title">
                        <span className="titlebar-icon" aria-hidden title="Minecraft">
                            <svg viewBox="0 0 16 16" width="18" height="18">
                                <rect x="2" y="2" width="12" height="12" fill="#8B5A2B"/>
                                <rect x="2" y="2" width="12" height="5" fill="#55a532"/>
                                <rect x="4" y="4" width="2" height="2" fill="#7cb342"/>
                                <rect x="10" y="4" width="2" height="2" fill="#7cb342"/>
                            </svg>
                        </span>
                        Майнкрафт online
                    </span>
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
                {authenticated && (
                    <aside className="user-bar">
                        <div className="user-bar-inner">
                            <span className="user-bar-nickname">{nickname}</span>
                            <button className="btn btn-logout" onClick={handleLogout}>Выйти</button>
                        </div>
                    </aside>
                )}

                <section className="hero">
                    {authenticated ? (
                        <>
                            <button
                                className="btn btn-play-hero"
                                onClick={handlePlay}
                                onMouseEnter={() => setPlayHovered(true)}
                                onMouseLeave={() => setPlayHovered(false)}
                                aria-label="Запустить игру"
                                disabled={!updateCheckDone || (updateInfo != null && updateInfo.mandatory)}
                            >
                                <span className="btn-play-icon" aria-hidden>
                                    <svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>
                                </span>
                                <span className="btn-play-text">Играть</span>
                            </button>
                            {playError && <p className="hero-error">{playError}</p>}
                        </>
                    ) : (
                        <>
                            <div className="hero-login-block">
                                <button className="btn btn-login-hero" onClick={handleLogin}>
                                    <span className="btn-play-icon" aria-hidden>
                                        <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>
                                    </span>
                                    <span className="btn-play-text">Войти через Telegram</span>
                                </button>
                                <p className="hero-login-hint">Сервер не хранит пароли</p>
                            </div>
                            {authError && <p className="hero-error">{authError}</p>}
                        </>
                    )}
                </section>

                <footer className="news-section">
                    <NewsFeed key={newsKey} />
                </footer>
            </main>
            <SettingsModal
                isOpen={settingsOpen}
                onClose={() => setSettingsOpen(false)}
                onSaved={() => { setNewsKey((k) => k + 1); refreshAuth(); }}
            />
            {updateInfo && (
                <UpdateModal
                    info={updateInfo}
                    visible={updateVisible}
                    inProgress={updateInProgress}
                    error={updateError}
                    onUpdate={handleUpdateNow}
                    onSkip={handleUpdateSkip}
                />
            )}
            <ProgressOverlay
                visible={progress.visible}
                title={progress.title}
                description={progress.description}
            />
            <GrassBlockScene visible={authenticated} playHovered={playHovered} />
        </div>
    )
}

export default App
