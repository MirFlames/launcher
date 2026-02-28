package com.launcher.auth;

import net.fabricmc.loader.api.FabricLoader;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Properties;

/**
 * Конфигурация мода из config/launcher_auth.properties.
 */
public final class AuthConfig {

    private static final String PROPERTIES_FILE = "launcher_auth.properties";
    private static final String KEY_API_URL = "auth_api_url";

    private static String apiUrl;

    private AuthConfig() {}

    static void load() throws IOException {
        Path configFile = FabricLoader.getInstance().getConfigDir().resolve(PROPERTIES_FILE);
        if (!Files.exists(configFile)) {
            throw new IllegalStateException(PROPERTIES_FILE + " не найден. Создайте файл и задайте auth_api_url.");
        }
        Properties props = new Properties();
        try (var reader = Files.newBufferedReader(configFile)) {
            props.load(reader);
        }
        String url = props.getProperty(KEY_API_URL);
        if (url == null || url.isBlank()) {
            throw new IllegalStateException(KEY_API_URL + " не задан в " + PROPERTIES_FILE);
        }
        apiUrl = url.trim().replaceAll("/+$", "");
        LauncherAuthMod.LOGGER.info("Auth API URL: {}", apiUrl);
    }

    public static String getApiUrl() {
        return apiUrl;
    }
}
