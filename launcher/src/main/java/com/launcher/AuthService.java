package com.launcher;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.launcher.dto.AuthSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.awt.Desktop;
import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URI;
import java.net.URL;
import java.util.Optional;
import java.util.function.Consumer;

/**
 * Сервис аутентификации через Telegram-бот с polling.
 * Использует кэш сессии и проверяет её валидность на backend при старте.
 */
public final class AuthService {

    private static final Logger log = LoggerFactory.getLogger(AuthService.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();

    private static final int POLL_INTERVAL_MS = 2000;
    private static final int TIMEOUT_MS = 5 * 60 * 1000; // 5 минут
    private static final int VERIFY_TIMEOUT_MS = 5000;

    /** Кэш сессии (null = ещё не загружен) */
    private static volatile Optional<AuthSession> cachedSession = null;

    private AuthService() {}

    private static void ensureCacheLoaded() {
        if (cachedSession == null) {
            cachedSession = AuthSessionStorage.load();
        }
    }

    /**
     * Проверяет, аутентифицирован ли игрок.
     */
    public static boolean isAuthenticated() {
        ensureCacheLoaded();
        return cachedSession.isPresent();
    }

    /**
     * Возвращает текущую сессию (если есть).
     */
    public static Optional<AuthSession> getSession() {
        ensureCacheLoaded();
        return cachedSession;
    }

    /**
     * Обновляет кэш сессии: загружает из файла, проверяет на backend.
     * При valid=false удаляет сессию. При ошибке сети — оставляет (оптимистично).
     *
     * @param onComplete вызывается в EDT по завершении (для repaint кнопки)
     */
    public static void refreshSession(Runnable onComplete) {
        new Thread(() -> {
            Optional<AuthSession> loaded = AuthSessionStorage.load();
            if (loaded.isEmpty()) {
                cachedSession = Optional.empty();
                invokeOnEdt(onComplete);
                return;
            }
            AuthSession session = loaded.get();
            Boolean valid = callAuthVerify(session.nickname(), session.sessionUuid());
            if (Boolean.FALSE.equals(valid)) {
                AuthSessionStorage.delete();
                cachedSession = Optional.empty();
                log.info("Сессия обнулена сервером, требуется повторный вход");
            } else {
                cachedSession = Optional.of(session);
            }
            invokeOnEdt(onComplete);
        }, "auth-refresh").start();
    }

    private static void invokeOnEdt(Runnable r) {
        if (r != null) {
            javax.swing.SwingUtilities.invokeLater(r);
        }
    }

    /**
     * Запускает процесс входа: init → открытие бота → polling.
     *
     * @param onSuccess вызывается при успешной аутентификации (в EDT)
     * @param onError   вызывается при ошибке с сообщением (в EDT)
     */
    public static void startLogin(Runnable onSuccess, Consumer<String> onError) {
        new Thread(() -> {
            try {
                AuthInitResponse initResp = callAuthInit();
                if (initResp == null) {
                    invokeOnError(onError, "Не удалось получить код для входа. Проверьте подключение к интернету.");
                    return;
                }

                openBrowser(initResp.botUrl);

                AuthSession session = pollUntilAuthenticated(initResp.code, onError);
                if (session != null) {
                    AuthSessionStorage.save(session);
                    cachedSession = Optional.of(session);
                    javax.swing.SwingUtilities.invokeLater(onSuccess);
                }
            } catch (Exception e) {
                log.error("Ошибка входа: {}", e.getMessage(), e);
                invokeOnError(onError, "Ошибка входа: " + e.getMessage());
            }
        }, "auth-login").start();
    }

    /**
     * Выход — удаление сохранённой сессии.
     */
    public static void logout() {
        AuthSessionStorage.delete();
        cachedSession = Optional.empty();
    }

    private static void invokeOnError(Consumer<String> onError, String message) {
        if (onError != null) {
            javax.swing.SwingUtilities.invokeLater(() -> onError.accept(message));
        }
    }

    private static AuthInitResponse callAuthInit() {
        try {
            String urlStr = Consts.API_BASE_URL + Consts.API_AUTH_INIT;
            HttpURLConnection conn = (HttpURLConnection) new URL(urlStr).openConnection();
            conn.setRequestMethod("POST");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(10000);

            int code = conn.getResponseCode();
            if (code != HttpURLConnection.HTTP_OK) {
                log.warn("Auth init: HTTP {}", code);
                return null;
            }

            JsonNode json = MAPPER.readTree(conn.getInputStream());
            return new AuthInitResponse(
                    json.has("code") ? json.get("code").asText() : null,
                    json.has("bot_url") ? json.get("bot_url").asText() : null);
        } catch (Exception e) {
            log.error("Auth init failed: {}", e.getMessage());
            return null;
        }
    }

    private static void openBrowser(String url) {
        try {
            if (Desktop.isDesktopSupported() && Desktop.getDesktop().isSupported(Desktop.Action.BROWSE)) {
                Desktop.getDesktop().browse(URI.create(url));
            } else {
                log.warn("Desktop.browse не поддерживается");
            }
        } catch (IOException e) {
            log.warn("Не удалось открыть браузер: {}", e.getMessage());
        }
    }

    private static AuthSession pollUntilAuthenticated(String authCode, Consumer<String> onError) {
        long deadline = System.currentTimeMillis() + TIMEOUT_MS;
        while (System.currentTimeMillis() < deadline) {
            try {
                AuthCheckResponse resp = callAuthCheck(authCode);
                if (resp == null) {
                    Thread.sleep(POLL_INTERVAL_MS);
                    continue;
                }
                if ("authenticated".equals(resp.status) && resp.nickname != null && resp.sessionUuid != null) {
                    return new AuthSession(resp.nickname, resp.sessionUuid);
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                invokeOnError(onError, "Вход отменён.");
                return null;
            } catch (Exception e) {
                log.debug("Auth check: {}", e.getMessage());
            }
            try {
                Thread.sleep(POLL_INTERVAL_MS);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return null;
            }
        }
        invokeOnError(onError, "Время ожидания входа истекло. Попробуйте снова.");
        return null;
    }

    /**
     * Проверяет сессию на backend. Возвращает true если валидна, false если нет, null при ошибке сети.
     */
    private static Boolean callAuthVerify(String nickname, String sessionUuid) {
        try {
            String urlStr = Consts.API_BASE_URL + Consts.API_AUTH_VERIFY
                    + "?nickname=" + java.net.URLEncoder.encode(nickname, "UTF-8")
                    + "&session_uuid=" + java.net.URLEncoder.encode(sessionUuid, "UTF-8");
            HttpURLConnection conn = (HttpURLConnection) new URL(urlStr).openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(VERIFY_TIMEOUT_MS);
            conn.setReadTimeout(VERIFY_TIMEOUT_MS);

            int rc = conn.getResponseCode();
            if (rc != HttpURLConnection.HTTP_OK) {
                log.warn("Auth verify: HTTP {}", rc);
                return null;
            }

            JsonNode json = MAPPER.readTree(conn.getInputStream());
            return json.has("valid") && json.get("valid").asBoolean();
        } catch (Exception e) {
            log.debug("Auth verify failed: {}", e.getMessage());
            return null;
        }
    }

    private static AuthCheckResponse callAuthCheck(String code) {
        try {
            String urlStr = Consts.API_BASE_URL + Consts.API_AUTH_CHECK + "?code=" + java.net.URLEncoder.encode(code, "UTF-8");
            HttpURLConnection conn = (HttpURLConnection) new URL(urlStr).openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(5000);
            conn.setReadTimeout(5000);

            int rc = conn.getResponseCode();
            if (rc != HttpURLConnection.HTTP_OK) {
                return null;
            }

            JsonNode json = MAPPER.readTree(conn.getInputStream());
            String status = json.has("status") ? json.get("status").asText() : "pending";
            String nickname = json.has("nickname") ? json.get("nickname").asText() : null;
            String sessionUuid = json.has("session_uuid") ? json.get("session_uuid").asText() : null;
            return new AuthCheckResponse(status, nickname, sessionUuid);
        } catch (Exception e) {
            return null;
        }
    }

    private static class AuthInitResponse {
        final String code;
        final String botUrl;

        AuthInitResponse(String code, String botUrl) {
            this.code = code;
            this.botUrl = botUrl;
        }
    }

    private static class AuthCheckResponse {
        final String status;
        final String nickname;
        final String sessionUuid;

        AuthCheckResponse(String status, String nickname, String sessionUuid) {
            this.status = status;
            this.nickname = nickname;
            this.sessionUuid = sessionUuid;
        }
    }
}
