import './UpdateModal.css';

export interface LauncherUpdateInfo {
    version: string;
    mandatory: boolean;
    changelog: string;
    download_url: string;
    size: number;
    sha256: string;
    current_version: string;
}

interface UpdateModalProps {
    info: LauncherUpdateInfo;
    visible: boolean;
    inProgress: boolean;
    error: string;
    onUpdate: () => void;
    onSkip: () => void;
}

export function UpdateModal({info, visible, inProgress, error, onUpdate, onSkip}: UpdateModalProps) {
    if (!visible) return null;

    const isMandatory = !!info.mandatory;

    const handleOverlayClick = () => {
        if (!isMandatory) {
            onSkip();
        }
    };

    return (
        <div className="update-modal-overlay" onClick={handleOverlayClick}>
            <div className="update-modal" onClick={(e) => e.stopPropagation()}>
                <div className="update-modal-header">
                    <h2>Доступно обновление лаунчера</h2>
                    {!isMandatory && (
                        <button
                            className="update-modal-close"
                            onClick={onSkip}
                            aria-label="Закрыть окно обновления"
                        >
                            ×
                        </button>
                    )}
                </div>
                <div className="update-modal-body">
                    <p className="update-modal-version">
                        Текущая версия: <strong>{info.current_version}</strong><br />
                        Новая версия: <strong>{info.version}</strong>
                    </p>
                    {info.size > 0 && (
                        <p className="update-modal-size">
                            Размер загрузки: ~{Math.round(info.size / (1024 * 1024))} МБ
                        </p>
                    )}
                    <div className="update-modal-changelog">
                        <h3>Что нового</h3>
                        <pre>{info.changelog || 'Описание изменений отсутствует.'}</pre>
                    </div>
                    {error && <p className="update-modal-error">{error}</p>}
                    {inProgress && !error && (
                        <p className="update-modal-progress">Загрузка обновления…</p>
                    )}
                </div>
                <div className="update-modal-footer">
                    {!isMandatory && (
                        <button
                            className="btn btn-secondary"
                            onClick={onSkip}
                            disabled={inProgress}
                        >
                            В другой раз
                        </button>
                    )}
                    <button
                        className="btn btn-primary"
                        onClick={onUpdate}
                        disabled={inProgress}
                    >
                        {inProgress ? 'Обновление…' : 'Обновить'}
                    </button>
                </div>
            </div>
        </div>
    );
}

