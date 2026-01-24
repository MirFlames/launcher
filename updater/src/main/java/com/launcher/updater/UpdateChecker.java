package com.launcher.updater;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.HttpURLConnection;
import java.net.URL;

/**
 * Проверка версий через API (если updateUrl не указан)
 */
public class UpdateChecker {
    private static final Logger log = LoggerFactory.getLogger(UpdateChecker.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();
    
    /**
     * Получает информацию об обновлении через API
     */
    public static UpdateInfo checkUpdate(String apiUrl) {
        try {
            String url = apiUrl + "/api/launcher/version";
            log.info("Запрос информации об обновлении: {}", url);
            
            HttpURLConnection conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(10000);
            
            int responseCode = conn.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                log.error("Ошибка запроса к API: HTTP {}", responseCode);
                return null;
            }
            
            JsonNode json = MAPPER.readTree(conn.getInputStream());
            UpdateInfo info = new UpdateInfo();
            info.downloadUrl = json.get("download_url").asText();
            info.hash = json.has("hash") ? json.get("hash").asText() : "";
            info.size = json.has("size") ? json.get("size").asLong() : 0;
            
            return info;
            
        } catch (Exception e) {
            log.error("Ошибка при проверке обновления: {}", e.getMessage(), e);
            return null;
        }
    }
    
    public static class UpdateInfo {
        public String downloadUrl;
        public String hash;
        public long size;
    }
}
