import {useState, useEffect} from 'react';
import {GetConfig, GetLauncherVersion, SaveConfig} from '../../wailsjs/go/main/App';
import {main} from '../../wailsjs/go/models';
import './SettingsModal.css';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
    onSaved?: () => void;
}

export function SettingsModal({isOpen, onClose, onSaved}: SettingsModalProps) {
    const [loadedConfig, setLoadedConfig] = useState<main.Config | null>(null);
    const [syncClientSettings, setSyncClientSettings] = useState(true);
    const [serverHost, setServerHost] = useState('');
    const [serverPort, setServerPort] = useState('');
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');
    const [launcherVersion, setLauncherVersion] = useState<string>('');

    useEffect(() => {
        if (isOpen) {
            GetConfig()
                .then((cfg) => {
                    if (cfg) {
                        setLoadedConfig(cfg);
                        setSyncClientSettings(cfg.sync_client_settings ?? true);
                        setServerHost((cfg.server_host || '').trim());
                        const p = cfg.server_port;
                        setServerPort(p != null && p > 0 ? String(p) : '');
                    }
                })
                .catch(() => {});
            GetLauncherVersion()
                .then((v) => setLauncherVersion(v || ''))
                .catch(() => setLauncherVersion(''));
        }
    }, [isOpen]);

    function handleSave() {
        setError('');
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
        const cfg = main.Config.createFrom({
            ...(loadedConfig || {}),
            sync_client_settings: syncClientSettings,
            server_host: host,
            server_port: portNum,
        });
        SaveConfig(cfg)
            .then(() => {
                onSaved?.();
                onClose();
            })
            .catch((e) => setError(e?.message || 'Ошибка сохранения'))
            .finally(() => setSaving(false));
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
                    <div className="modal-dev-warning" role="status">
                        <strong>Только для разработки.</strong> Ниже можно переопределить адрес игрового сервера для автоподключения. Для обычной игры оставьте поля пустыми — используются данные с бэкенда.
                    </div>
                    <label htmlFor="settings-server-host">IP или хост сервера</label>
                    <input
                        id="settings-server-host"
                        type="text"
                        autoComplete="off"
                        placeholder="например 127.0.0.1"
                        value={serverHost}
                        onChange={(e) => setServerHost(e.target.value)}
                    />
                    <label htmlFor="settings-server-port">Порт сервера</label>
                    <input
                        id="settings-server-port"
                        className="modal-input-port"
                        type="text"
                        inputMode="numeric"
                        autoComplete="off"
                        placeholder="например 25565"
                        value={serverPort}
                        onChange={(e) => setServerPort(e.target.value)}
                    />
                    <label className="modal-checkbox">
                        <input
                            type="checkbox"
                            checked={syncClientSettings}
                            onChange={(e) => setSyncClientSettings(e.target.checked)}
                        />
                        <span>Синхронизировать настройки клиента</span>
                    </label>
                    <p className="modal-hint">
                        Если включено, settings-файлы клиента (например options.txt) применяются только при докачке модов с сервера.
                    </p>
                    {error && <p className="modal-error">{error}</p>}
                </div>
                {launcherVersion && (
                    <div className="modal-version-row">
                        <span className="modal-version-label">Версия лаунчера</span>
                        <span className="modal-version-value">{launcherVersion}</span>
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
