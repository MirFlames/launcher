package com.launcher.autoconnect;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.FileReader;
import java.io.IOException;
import java.nio.charset.StandardCharsets;

/**
 * Загрузка конфигурации из configs/launcher-config.json (тот же файл, что использует лаунчер).
 */
public final class AutoConnectConfig {

    private static final Logger LOG = LoggerFactory.getLogger(LauncherAutoConnectMod.MOD_ID);
    private static final Gson GSON = new Gson();

    private static final String CONFIG_FILE = "configs/launcher-config.json";

    private AutoConnectConfig() {}

    public static String getServerHost() {
        JsonObject cfg = load();
        if (cfg == null || !cfg.has("server_host")) return null;
        String host = cfg.get("server_host").getAsString();
        return (host != null && !host.isBlank()) ? host.trim() : null;
    }

    public static int getServerPort() {
        JsonObject cfg = load();
        if (cfg == null || !cfg.has("server_port")) return 25565;
        try {
            int port = cfg.get("server_port").getAsInt();
            return port > 0 ? port : 25565;
        } catch (Exception e) {
            return 25565;
        }
    }

    /**
     * Возвращает true, если в конфиге указан сервер для автоподключения.
     */
    public static boolean isEnabled() {
        String host = getServerHost();
        return host != null && !host.isBlank();
    }

    private static JsonObject load() {
        try {
            File gameDir = getGameDirectory();
            if (gameDir == null) return null;
            File configFile = new File(gameDir, CONFIG_FILE);
            if (!configFile.exists()) return null;
            try (var reader = new FileReader(configFile, StandardCharsets.UTF_8)) {
                return GSON.fromJson(reader, JsonObject.class);
            }
        } catch (IOException e) {
            LOG.warn("Не удалось загрузить конфиг автоподключения: {}", e.getMessage());
            return null;
        }
    }

    private static File getGameDirectory() {
        try {
            var path = net.fabricmc.loader.api.FabricLoader.getInstance().getGameDir();
            return path.toFile();
        } catch (Throwable e) {
            LOG.warn("Не удалось определить gameDir: {}", e.getMessage());
            return null;
        }
    }
}
