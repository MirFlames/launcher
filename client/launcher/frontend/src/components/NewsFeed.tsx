import {useState, useEffect} from 'react';
import {GetNewsFeed} from '../../wailsjs/go/main/App';
import type {main} from '../../wailsjs/go/models';
import './NewsFeed.css';

export function NewsFeed() {
    const [response, setResponse] = useState<main.NewsFeedResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    function load() {
        setLoading(true);
        setError(null);
        GetNewsFeed()
            .then((data) => {
                setResponse(data || null);
            })
            .catch((e) => {
                setError(e?.message || 'Не удалось загрузить новости');
                setResponse(null);
            })
            .finally(() => setLoading(false));
    }

    useEffect(() => {
        load();
        const id = setInterval(load, 5 * 60 * 1000); // обновление каждые 5 мин
        return () => clearInterval(id);
    }, []);

    if (loading && !response) {
        return (
            <div className="news-feed">
                <h3 className="news-feed-title">Новость дня</h3>
                <p className="news-feed-placeholder">Загрузка…</p>
            </div>
        );
    }

    if (error) {
        return (
            <div className="news-feed">
                <h3 className="news-feed-title">Новость дня</h3>
                <p className="news-feed-error">{error}</p>
            </div>
        );
    }

    // Не авторизован — показываем призыв войти
    if (response && !response.authenticated) {
        return (
            <div className="news-feed">
                <h3 className="news-feed-title">Новость дня</h3>
                <p className="news-feed-placeholder">{response.message || 'Войдите для просмотра новостей'}</p>
            </div>
        );
    }

    // Авторизован, но нет новости
    if (response && response.authenticated && !response.news) {
        return (
            <div className="news-feed">
                <h3 className="news-feed-title">Новость дня</h3>
                <p className="news-feed-placeholder">{response.message || 'В канале пока нет новостей'}</p>
            </div>
        );
    }

    // Авторизован, есть новость
    const news = response!.news!;
    const text = news.text || '';
    const truncated = text.length > 200 ? text.slice(0, 200) + '…' : text;

    return (
        <div className="news-feed">
            <h3 className="news-feed-title">Новость дня</h3>
            <div className="news-feed-item">
                {news.published && (
                    <span className="news-feed-date">{news.published}</span>
                )}
                <p className="news-feed-desc">{truncated}</p>
                {news.link && (
                    <a
                        href={news.link}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="news-feed-link"
                    >
                        Читать в канале →
                    </a>
                )}
            </div>
        </div>
    );
}
