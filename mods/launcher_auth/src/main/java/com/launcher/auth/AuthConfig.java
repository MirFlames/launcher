package com.launcher.auth;

import java.io.IOException;
import java.io.InputStream;
import java.util.Properties;

/**
 * Конфигурация мода из launcher_auth.properties.
 */
public final class AuthConfig {

    private static final String PROPERTIES_FILE = "launcher_auth.properties";
    private static final String KEY_API_URL = "auth_api_url";

    private static String apiUrl = "http://localhost:80";

    private AuthConfig() {}

    static void load() {
        try (InputStream is = AuthConfig.class.getClassLoader()
                .getResourceAsStream(PROPERTIES_FILE)) {
            if (is == null) {
                LauncherAuthMod.LOGGER.warn("{} не найден, используется URL по умолчанию: {}", PROPERTIES_FILE, apiUrl);
                return;
            }
            Properties props = new Properties();
            props.load(is);
            String url = props.getProperty(KEY_API_URL);
            if (url != null && !url.isBlank()) {
                apiUrl = url.trim().replaceAll("/+$", "");
            }
            LauncherAuthMod.LOGGER.info("Auth API URL: {}", apiUrl);
        } catch (IOException e) {
            LauncherAuthMod.LOGGER.warn("Не удалось загрузить {}: {}", PROPERTIES_FILE, e.getMessage());
        }
    }

    public static String getApiUrl() {
        return apiUrl;
    }
}
