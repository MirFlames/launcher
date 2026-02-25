package com.launcher;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.launcher.dto.ServerVersion;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.HttpURLConnection;
import java.net.URI;
import java.net.URL;
import java.util.Optional;

/**
 * Клиент для получения манифеста версии (моды, client_files) с сервера.
 */
public final class ServerVersionClient {

    private static final Logger log = LoggerFactory.getLogger(ServerVersionClient.class);
    private static final ObjectMapper MAPPER = new ObjectMapper()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

    private static final int CONNECT_TIMEOUT = 10000;
    private static final int READ_TIMEOUT = 30000;

    private ServerVersionClient() {}

    /**
     * Запрашивает манифест версии с сервера.
     *
     * @param apiBaseUrl базовый URL API (например http://localhost:8080)
     * @return манифест или пусто при ошибке
     */
    public static Optional<ServerVersion> fetch(String apiBaseUrl) {
        String urlString = apiBaseUrl.replaceAll("/$", "") + Consts.API_VERSION;
        try {
            URL url = URI.create(urlString).toURL();
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(CONNECT_TIMEOUT);
            conn.setReadTimeout(READ_TIMEOUT);
            conn.setRequestProperty("Accept", "application/json");

            int responseCode = conn.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                log.error("Ошибка запроса /api/version: HTTP {}", responseCode);
                return Optional.empty();
            }

            ServerVersion version = MAPPER.readValue(conn.getInputStream(), ServerVersion.class);
            log.info("Получен манифест версии: {} модов", version.mods() != null ? version.mods().size() : 0);
            return Optional.of(version);
        } catch (Exception e) {
            log.error("Ошибка при запросе версии с сервера: {}", e.getMessage(), e);
            return Optional.empty();
        }
    }
}
