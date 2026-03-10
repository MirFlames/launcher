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
        setSaving(true);
        const cfg = main.Config.createFrom({
            ...(loadedConfig || {}),
            sync_client_settings: syncClientSettings,
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
