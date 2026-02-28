package com.launcher.autoconnect;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.FileReader;
import java.nio.charset.StandardCharsets;
import java.util.Optional;
import java.util.UUID;

/**
 * Загрузка сессии из configs/launcher-auth.json (тот же файл, что создаёт лаунчер при входе).
 * Используется для подстановки session_uuid при подключении к серверу, если игра была
 * запущена с дефолтным UUID (например, без входа в лаунчере).
 */
public final class LauncherAuthSession {

    private static final Logger LOG = LoggerFactory.getLogger(LauncherAutoConnectMod.MOD_ID);
    private static final Gson GSON = new Gson();
    private static final String AUTH_FILE = "configs/launcher-auth.json";
    private static final String KEY_NICKNAME = "nickname";
    private static final String KEY_SESSION_UUID = "session_uuid";

    /** UUID по умолчанию при отсутствии сессии */
    private static final UUID DEFAULT_UUID = new UUID(0, 0);

    private LauncherAuthSession() {}

    /**
     * Загружает сессию из launcher-auth.json.
     *
     * @return Optional с (nickname, sessionUuid) или empty если файл отсутствует/невалиден
     */
    public static Optional<AuthData> load() {
        try {
            File gameDir = getGameDirectory();
            if (gameDir == null) return Optional.empty();

            File authFile = new File(gameDir, AUTH_FILE);
            if (!authFile.exists() || !authFile.isFile() || authFile.length() == 0) {
                return Optional.empty();
            }

            try (var reader = new FileReader(authFile, StandardCharsets.UTF_8)) {
                JsonObject obj = GSON.fromJson(reader, JsonObject.class);
                if (obj == null) return Optional.empty();

                String nickname = getString(obj, KEY_NICKNAME);
                String sessionUuid = getString(obj, KEY_SESSION_UUID);
                if (nickname == null || nickname.isBlank() || sessionUuid == null || sessionUuid.isBlank()) {
                    return Optional.empty();
                }

                UUID uuid;
                try {
                    uuid = UUID.fromString(sessionUuid.trim());
                } catch (IllegalArgumentException e) {
                    LOG.warn("Невалидный session_uuid в launcher-auth.json: {}", sessionUuid);
                    return Optional.empty();
                }

                if (DEFAULT_UUID.equals(uuid)) {
                    return Optional.empty();
                }

                return Optional.of(new AuthData(nickname.trim(), uuid));
            }
        } catch (Exception e) {
            LOG.debug("Не удалось загрузить launcher-auth.json: {}", e.getMessage());
            return Optional.empty();
        }
    }

    private static String getString(JsonObject obj, String key) {
        if (obj == null || !obj.has(key)) return null;
        var el = obj.get(key);
        if (el == null || el.isJsonNull()) return null;
        return el.getAsString();
    }

    private static File getGameDirectory() {
        try {
            var path = net.fabricmc.loader.api.FabricLoader.getInstance().getGameDir();
            return path.toFile();
        } catch (Throwable e) {
            return null;
        }
    }

    public static boolean isDefaultUuid(UUID uuid) {
        return uuid == null || DEFAULT_UUID.equals(uuid);
    }

    public record AuthData(String nickname, UUID sessionUuid) {}
}
