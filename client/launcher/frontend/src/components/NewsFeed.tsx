import {useState, useEffect} from 'react';
import {GetNewsFeed} from '../../wailsjs/go/main/App';
import './NewsFeed.css';

interface NewsItem {
    title: string;
    link: string;
    description: string;
    published: string;
}

export function NewsFeed() {
    const [items, setItems] = useState<NewsItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    function load() {
        setLoading(true);
        setError(null);
        GetNewsFeed()
            .then((data) => {
                setItems(data || []);
            })
            .catch((e) => {
                setError(e?.message || 'Не удалось загрузить новости');
                setItems([]);
            })
            .finally(() => setLoading(false));
    }

    useEffect(() => {
        load();
        const id = setInterval(load, 5 * 60 * 1000); // обновление каждые 5 мин
        return () => clearInterval(id);
    }, []);

    if (loading && items.length === 0) {
        return (
            <div className="news-feed">
                <h3 className="news-feed-title">Новости</h3>
                <p className="news-feed-placeholder">Загрузка…</p>
            </div>
        );
    }

    if (error && items.length === 0) {
        return (
            <div className="news-feed">
                <h3 className="news-feed-title">Новости</h3>
                <p className="news-feed-error">{error}</p>
                <p className="news-feed-hint">Укажите Telegram-канал в настройках</p>
            </div>
        );
    }

    if (items.length === 0) {
        return (
            <div className="news-feed">
                <h3 className="news-feed-title">Новости</h3>
                <p className="news-feed-placeholder">Нет новостей. Добавьте канал в настройках</p>
            </div>
        );
    }

    return (
        <div className="news-feed">
            <h3 className="news-feed-title">Новости</h3>
            <ul className="news-feed-list">
                {items.map((item, i) => (
                    <li key={i} className="news-feed-item">
                        <a
                            href={item.link}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="news-feed-link"
                        >
                            {item.title}
                        </a>
                        {item.published && (
                            <span className="news-feed-date">{item.published}</span>
                        )}
                        {item.description && (() => {
                            const text = item.description.replace(/<[^>]+>/g, '');
                            return (
                                <p className="news-feed-desc">
                                    {text.slice(0, 120)}{text.length > 120 ? '…' : ''}
                                </p>
                            );
                        })()}
                    </li>
                ))}
            </ul>
        </div>
    );
}
