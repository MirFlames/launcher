import './ProgressOverlay.css';

interface ProgressOverlayProps {
    visible: boolean;
    title: string;
    description?: string;
}

export function ProgressOverlay({visible, title, description}: ProgressOverlayProps) {
    if (!visible) return null;

    return (
        <div className="progress-overlay">
            <div className="progress-overlay-backdrop" />
            <div className="progress-overlay-content">
                <div className="progress-overlay-spinner">
                    <svg viewBox="0 0 50 50" className="spinner-svg">
                        <circle cx="25" cy="25" r="20" fill="none" strokeWidth="4" stroke="currentColor" strokeDasharray="31.4 94.2" strokeLinecap="round" />
                    </svg>
                </div>
                <p className="progress-overlay-title">{title}</p>
                {description && <p className="progress-overlay-desc">{description}</p>}
            </div>
        </div>
    );
}
