package com.launcher;

import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;

/**
 * Загрузка конфигурации лаунчера из configs/launcher-config.json.
 * Позволяет разделять dev/prod окружения (api_base_url).
 */
public final class LauncherConfigLoader {

    private static final Logger log = LoggerFactory.getLogger(LauncherConfigLoader.class);
    private static final ObjectMapper MAPPER = new ObjectMapper()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

    private static final String CONFIG_FILE = "configs/launcher-config.json";

    private LauncherConfigLoader() {}

    /**
     * Возвращает базовый URL API. Читает из launcher-config.json, иначе — Consts.API_BASE_URL.
     */
    public static String getApiBaseUrl() {
        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        File configFile = new File(minecraftFolder, CONFIG_FILE);
        if (!configFile.exists()) {
            return Consts.API_BASE_URL;
        }
        try {
            LauncherConfig cfg = MAPPER.readValue(configFile, LauncherConfig.class);
            if (cfg != null && cfg.apiBaseUrl() != null && !cfg.apiBaseUrl().isBlank()) {
                return cfg.apiBaseUrl().replaceAll("/$", "");
            }
        } catch (IOException e) {
            log.warn("Не удалось прочитать {}: {}", configFile.getAbsolutePath(), e.getMessage());
        }
        return Consts.API_BASE_URL;
    }

    private record LauncherConfig(
            @JsonProperty("api_base_url") String apiBaseUrl,
            @JsonProperty("server_host") String serverHost,
            @JsonProperty("server_port") Integer serverPort) {}

    /**
     * Возвращает хост сервера для автоподключения. По умолчанию localhost.
     */
    public static String getServerHost() {
        LauncherConfig cfg = loadConfig();
        return (cfg != null && cfg.serverHost() != null && !cfg.serverHost().isBlank())
                ? cfg.serverHost().trim() : "localhost";
    }

    /**
     * Возвращает порт сервера для автоподключения. По умолчанию 25565.
     */
    public static int getServerPort() {
        LauncherConfig cfg = loadConfig();
        return (cfg != null && cfg.serverPort() != null && cfg.serverPort() > 0)
                ? cfg.serverPort() : 25565;
    }

    private static LauncherConfig loadConfig() {
        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        File configFile = new File(minecraftFolder, CONFIG_FILE);
        if (!configFile.exists()) return null;
        try {
            return MAPPER.readValue(configFile, LauncherConfig.class);
        } catch (IOException e) {
            log.warn("Не удалось прочитать {}: {}", configFile.getAbsolutePath(), e.getMessage());
            return null;
        }
    }
}
