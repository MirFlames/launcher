import {useState, useEffect} from 'react';
import {GetBuildDefaults, GetConfig, GetLauncherVersion, SaveConfig} from '../../wailsjs/go/main/App';
import {main} from '../../wailsjs/go/models';
import './SettingsModal.css';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSaved?: () => void;
}

// Профили окружения: «Прод» — адреса из сборки (.env), «Dev» — адреса тестового
// стенда. Поля ниже редактируют активный профиль; неактивный лежит в конфиге и
// ждёт своего часа, чтобы не перепечатывать адреса при каждом переключении.
type EnvName = 'prod' | 'dev';

const ENV_PROD: EnvName = 'prod';
const ENV_DEV: EnvName = 'dev';

type EnvProfiles = Record<string, main.EnvProfile>;

function portToText(port?: number): string {
    return port != null && port > 0 ? String(port) : '';
}

export function SettingsModal({isOpen, onClose, onSaved}: SettingsModalProps) {
    const [loadedConfig, setLoadedConfig] = useState<main.Config | null>(null);
    const [apiBaseUrl, setApiBaseUrl] = useState('');
    const [serverHost, setServerHost] = useState('');
    const [serverPort, setServerPort] = useState('');
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [launcherVersion, setLauncherVersion] = useState<string>('');
    const [devUnlocked, setDevUnlocked] = useState(false);
    const [devSecretTaps, setDevSecretTaps] = useState(0);
    const [authlibInjectorDebug, setAuthlibInjectorDebug] = useState(false);
    const [skipServerModSync, setSkipServerModSync] = useState(false);
    const [devMarketingSessionId, setDevMarketingSessionId] = useState('');
    const [devSourceCode, setDevSourceCode] = useState('');
    const [devUtmSource, setDevUtmSource] = useState('');
    const [devUtmMedium, setDevUtmMedium] = useState('');
    const [devUtmCampaign, setDevUtmCampaign] = useState('');
    const [devUtmContent, setDevUtmContent] = useState('');
    const [devLandingPath, setDevLandingPath] = useState('');
    const [env, setEnv] = useState<EnvName>(ENV_PROD);
    const [profiles, setProfiles] = useState<EnvProfiles>({});
    const [buildDefaults, setBuildDefaults] = useState<main.EnvProfile | null>(null);

    useEffect(() => {
        if (isOpen) {
            setDevUnlocked(false);
            setDevSecretTaps(0);
            GetBuildDefaults()
                .then((d) => setBuildDefaults(d))
                .catch(() => setBuildDefaults(null));
            GetConfig()
                .then((cfg) => {
                    if (cfg) {
                        setLoadedConfig(cfg);
                        const activeEnv: EnvName = cfg.env === ENV_DEV ? ENV_DEV : ENV_PROD;
                        const loadedProfiles: EnvProfiles = {...(cfg.env_profiles || {})};
                        setEnv(activeEnv);
                        setProfiles(loadedProfiles);
                        // Поля показывают активный профиль, а не эффективные значения:
                        // пустое поле у прода = «взять из сборки» (в placeholder).
                        const active = loadedProfiles[activeEnv];
                        setApiBaseUrl((active?.apiBaseUrl || '').trim());
                        setServerHost((active?.server_host || '').trim());
                        setServerPort(portToText(active?.server_port));
                        setAuthlibInjectorDebug(!!cfg.authlib_injector_debug);
                        setSkipServerModSync(!!cfg.skip_server_mod_sync);
                        setDevMarketingSessionId((cfg.dev_marketing_session_id || '').trim());
                        setDevSourceCode((cfg.dev_source_code || '').trim());
                        setDevUtmSource((cfg.dev_utm_source || '').trim());
                        setDevUtmMedium((cfg.dev_utm_medium || '').trim());
                        setDevUtmCampaign((cfg.dev_utm_campaign || '').trim());
                        setDevUtmContent((cfg.dev_utm_content || '').trim());
                        setDevLandingPath((cfg.dev_landing_path || '').trim());
                    }
                })
                .catch(() => {});
            GetLauncherVersion()
                .then((v) => setLauncherVersion(v || ''))
                .catch(() => setLauncherVersion(''));
        }
    }, [isOpen]);

    // Снимок полей в профиль. Socks-поля UI не показывает — переносим их как есть,
    // иначе переключение окружения тихо стирало бы прокси активного профиля.
    function fieldsToProfile(): main.EnvProfile {
        const n = parseInt(serverPort.trim(), 10);
        return main.EnvProfile.createFrom({
            ...(profiles[env] || {}),
            apiBaseUrl: apiBaseUrl.trim(),
            server_host: serverHost.trim(),
            server_port: Number.isFinite(n) && n > 0 ? n : 0,
        });
    }

    function handleEnvSwitch(next: EnvName) {
        if (next === env) return;
        setError('');
        const merged: EnvProfiles = {...profiles, [env]: fieldsToProfile()};
        setProfiles(merged);
        const target = merged[next];
        setApiBaseUrl((target?.apiBaseUrl || '').trim());
        setServerHost((target?.server_host || '').trim());
        setServerPort(portToText(target?.server_port));
        setEnv(next);
    }

    function handleSave() {
        setError('');
        const api = apiBaseUrl.trim();
        if (api !== '') {
            const low = api.toLowerCase();
            if (!low.startsWith('http://') && !low.startsWith('https://')) {
                setError('Backend API: укажите URL с http:// или https://.');
                return;
            }
        }
        const host = serverHost.trim();
        const portStr = serverPort.trim();
        let portNum = 0;
        if (host !== '') {
            if (portStr === '') {
                setError('Укажите порт сервера или очистите поле адреса.');
                return;
            }
            const n = parseInt(portStr, 10);
            if (!Number.isFinite(n) || n < 1 || n > 65535) {
                setError('Порт должен быть числом от 1 до 65535.');
                return;
            }
            portNum = n;
        } else if (portStr !== '') {
            setError('Укажите адрес сервера или очистите порт.');
            return;
        }
        setSaving(true);
        const profile = main.EnvProfile.createFrom({
            ...(profiles[env] || {}),
            apiBaseUrl: api,
            server_host: host,
            server_port: portNum,
        });
        const nextProfiles: EnvProfiles = {...profiles, [env]: profile};
        const cfg = main.Config.createFrom({
            ...(loadedConfig || {}),
            env,
            env_profiles: nextProfiles,
            // Эффективные адреса Go всё равно пересоберёт из активного профиля
            // (Config.ApplyProfile) — здесь они лишь для консистентности объекта.
            apiBaseUrl: api,
            server_host: host,
            server_port: portNum,
            authlib_injector_debug: authlibInjectorDebug,
            skip_server_mod_sync: skipServerModSync,
            dev_marketing_session_id: devMarketingSessionId.trim(),
            dev_source_code: devSourceCode.trim(),
            dev_utm_source: devUtmSource.trim(),
            dev_utm_medium: devUtmMedium.trim(),
            dev_utm_campaign: devUtmCampaign.trim(),
            dev_utm_content: devUtmContent.trim(),
            dev_landing_path: devLandingPath.trim(),
        });
        SaveConfig(cfg)
            .then(() => {
                onSaved?.();
                onClose();
            })
            .catch((e) => setError(e?.message || 'Ошибка сохранения'))
            .finally(() => setSaving(false));
    }

    function handleLauncherVersionSecretTap() {
        setDevSecretTaps((n) => {
            const next = n + 1;
            if (next >= 5) {
                setDevUnlocked(true);
                return 0;
            }
            return next;
        });
    }

    if (!isOpen) return null;

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal" onClick={(e) => e.stopPropagation()}>
                <div className="modal-header">
                    <h2>Настройки</h2>
                    <button className="modal-close" onClick={onClose}>×</button>
                </div>
                <div className="modal-body">
                    {devUnlocked && (
                        <details className="modal-dev-submenu" open>
                            <summary className="modal-dev-submenu-summary">Для разработчиков</summary>
                            <div className="modal-dev-submenu-body">
                                <div className="modal-dev-warning" role="status">
                                    <strong>Только для разработки.</strong>
                                    Здесь задаётся адрес бэкенда и переопределение игрового сервера для автоподключения. Для обычной игры не открывайте этот раздел или оставьте значения по умолчанию из сборки.
                                </div>
                                <div className="modal-env-switch-block">
                                    <span className="modal-env-switch-title" id="settings-env-label">Профиль подключения</span>
                                    <div
                                        className={`modal-env-switch${env === ENV_DEV ? ' modal-env-switch-dev' : ''}`}
                                        role="radiogroup"
                                        aria-labelledby="settings-env-label"
                                    >
                                        <button
                                            type="button"
                                            role="radio"
                                            aria-checked={env === ENV_PROD}
                                            className={`modal-env-switch-btn${env === ENV_PROD ? ' modal-env-switch-btn-active' : ''}`}
                                            onClick={() => handleEnvSwitch(ENV_PROD)}
                                        >
                                            Прод
                                        </button>
                                        <button
                                            type="button"
                                            role="radio"
                                            aria-checked={env === ENV_DEV}
                                            className={`modal-env-switch-btn${env === ENV_DEV ? ' modal-env-switch-btn-active' : ''}`}
                                            onClick={() => handleEnvSwitch(ENV_DEV)}
                                        >
                                            Dev-стенд
                                        </button>
                                    </div>
                                    <p className="modal-hint modal-hint-dev">
                                        Прод — адреса из сборки. Dev — адреса тестового стенда: введите их ниже
                                        один раз, дальше переключайтесь одной кнопкой. Смена профиля меняет
                                        бэкенд, поэтому потребуется войти заново.
                                    </p>
                                </div>
                                <label htmlFor="settings-api-base-url">Backend API</label>
                                <input
                                    id="settings-api-base-url"
                                    type="url"
                                    autoComplete="off"
                                    placeholder={env === ENV_PROD ? (buildDefaults?.apiBaseUrl || 'из сборки') : 'например http://localhost:80'}
                                    value={apiBaseUrl}
                                    onChange={(e) => setApiBaseUrl(e.target.value)}
                                />
                                <label htmlFor="settings-server-host">IP или хост игрового сервера</label>
                                <input
                                    id="settings-server-host"
                                    type="text"
                                    autoComplete="off"
                                    placeholder={env === ENV_PROD ? (buildDefaults?.server_host || 'из сборки') : 'например 127.0.0.1'}
                                    value={serverHost}
                                    onChange={(e) => setServerHost(e.target.value)}
                                />
                                <label htmlFor="settings-server-port">Порт игрового сервера</label>
                                <input
                                    id="settings-server-port"
                                    className="modal-input-port"
                                    type="text"
                                    inputMode="numeric"
                                    autoComplete="off"
                                    placeholder={env === ENV_PROD ? (portToText(buildDefaults?.server_port) || 'из сборки') : 'например 25565'}
                                    value={serverPort}
                                    onChange={(e) => setServerPort(e.target.value)}
                                />
                                <label className="modal-checkbox modal-checkbox-first modal-checkbox-dev">
                                    <input
                                        type="checkbox"
                                        checked={skipServerModSync}
                                        onChange={(e) => setSkipServerModSync(e.target.checked)}
                                    />
                                    <span>Не синхронизировать моды с сервером</span>
                                </label>
                                <p className="modal-hint modal-hint-dev">
                                    Не скачивать моды и конфиги модов из API; используйте локальную папку mods.
                                </p>
                                <label className="modal-checkbox modal-checkbox-dev">
                                    <input
                                        type="checkbox"
                                        checked={authlibInjectorDebug}
                                        onChange={(e) => setAuthlibInjectorDebug(e.target.checked)}
                                    />
                                    <span>Отладка authlib-injector (-Dauthlibinjector.debug=verbose,authlib)</span>
                                </label>
                                <p className="modal-hint modal-hint-dev">
                                    Подробный лог в консоли Java; для диагностики сети и Yggdrasil.
                                </p>
                                <div className="modal-dev-divider" />
                                <p className="modal-hint modal-hint-dev modal-hint-dev-strong">
                                    Dev-only marketing args. Эти значения будут проброшены в Minecraft как `--marketing-*` и `--utm-*`.
                                </p>
                                <label htmlFor="settings-dev-marketing-session-id">Marketing session id</label>
                                <input
                                    id="settings-dev-marketing-session-id"
                                    type="text"
                                    autoComplete="off"
                                    placeholder="например browser-session-123"
                                    value={devMarketingSessionId}
                                    onChange={(e) => setDevMarketingSessionId(e.target.value)}
                                />
                                <label htmlFor="settings-dev-source-code">Source code</label>
                                <input
                                    id="settings-dev-source-code"
                                    type="text"
                                    autoComplete="off"
                                    placeholder="например telegram-post-01"
                                    value={devSourceCode}
                                    onChange={(e) => setDevSourceCode(e.target.value)}
                                />
                                <label htmlFor="settings-dev-utm-source">UTM source</label>
                                <input
                                    id="settings-dev-utm-source"
                                    type="text"
                                    autoComplete="off"
                                    placeholder="telegram"
                                    value={devUtmSource}
                                    onChange={(e) => setDevUtmSource(e.target.value)}
                                />
                                <label htmlFor="settings-dev-utm-medium">UTM medium</label>
                                <input
                                    id="settings-dev-utm-medium"
                                    type="text"
                                    autoComplete="off"
                                    placeholder="community"
                                    value={devUtmMedium}
                                    onChange={(e) => setDevUtmMedium(e.target.value)}
                                />
                                <label htmlFor="settings-dev-utm-campaign">UTM campaign</label>
                                <input
                                    id="settings-dev-utm-campaign"
                                    type="text"
                                    autoComplete="off"
                                    placeholder="phase-0"
                                    value={devUtmCampaign}
                                    onChange={(e) => setDevUtmCampaign(e.target.value)}
                                />
                                <label htmlFor="settings-dev-utm-content">UTM content</label>
                                <input
                                    id="settings-dev-utm-content"
                                    type="text"
                                    autoComplete="off"
                                    placeholder="creative-a"
                                    value={devUtmContent}
                                    onChange={(e) => setDevUtmContent(e.target.value)}
                                />
                                <label htmlFor="settings-dev-landing-path">Landing path</label>
                                <input
                                    id="settings-dev-landing-path"
                                    type="text"
                                    autoComplete="off"
                                    placeholder="/minecraft/"
                                    value={devLandingPath}
                                    onChange={(e) => setDevLandingPath(e.target.value)}
                                />
                            </div>
                        </details>
                    )}
                    {error && <p className="modal-error">{error}</p>}
                </div>
                {launcherVersion && (
                    <div className="modal-version-row">
                        <span className="modal-version-label">Версия лаунчера</span>
                        <button
                            type="button"
                            className="modal-version-value modal-version-secret-trigger"
                            onClick={handleLauncherVersionSecretTap}
                        >
                            {launcherVersion}
                        </button>
                    </div>
                )}
                <div className="modal-footer">
                    <button className="btn btn-secondary" onClick={onClose}>Отмена</button>
                    <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
                        {saving ? 'Сохранение…' : 'Сохранить'}
                    </button>
                </div>
            </div>
        </div>
    );
}
