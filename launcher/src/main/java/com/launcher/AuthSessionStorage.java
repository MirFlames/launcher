package com.launcher;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.JavaType;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.launcher.dto.AuthSession;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.util.Map;
import java.util.Optional;

/**
 * Хранение сессии аутентификации в launcher-auth.json.
 * Использует Map для сериализации — совместимо с GraalVM native image.
 */
public final class AuthSessionStorage {

    private static final Logger log = LoggerFactory.getLogger(AuthSessionStorage.class);
    private static final ObjectMapper MAPPER = new ObjectMapper()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

    private static final JavaType MAP_TYPE = MAPPER.getTypeFactory()
            .constructMapType(Map.class, String.class, String.class);

    private static final String AUTH_FILE_NAME = "configs/launcher-auth.json";
    private static final String KEY_NICKNAME = "nickname";
    private static final String KEY_SESSION_UUID = "session_uuid";

    private AuthSessionStorage() {}

    /**
     * Загружает сессию из файла.
     */
    public static Optional<AuthSession> load() {
        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        File authFile = new File(minecraftFolder, AUTH_FILE_NAME);
        if (!authFile.exists() || !authFile.isFile() || authFile.length() == 0) {
            return Optional.empty();
        }
        try {
            @SuppressWarnings("unchecked")
            Map<String, String> data = MAPPER.readValue(authFile, MAP_TYPE);
            if (data == null) {
                return Optional.empty();
            }
            String nickname = data.get(KEY_NICKNAME);
            String sessionUuid = data.get(KEY_SESSION_UUID);
            if (nickname == null || sessionUuid == null) {
                return Optional.empty();
            }
            AuthSession session = new AuthSession(nickname.trim(), sessionUuid.trim());
            return session.isValid() ? Optional.of(session) : Optional.empty();
        } catch (IOException e) {
            log.warn("Не удалось загрузить сессию из {}: {}", authFile.getAbsolutePath(), e.getMessage());
            return Optional.empty();
        }
    }

    /**
     * Сохраняет сессию в файл.
     */
    public static void save(AuthSession session) throws IOException {
        if (session == null || !session.isValid()) {
            throw new IllegalArgumentException("Сессия должна быть валидной");
        }
        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        File configsDir = new File(minecraftFolder, "configs");
        configsDir.mkdirs();
        File authFile = new File(configsDir, "launcher-auth.json");
        Map<String, String> data = Map.of(
                KEY_NICKNAME, session.nickname(),
                KEY_SESSION_UUID, session.sessionUuid());
        MAPPER.writerWithDefaultPrettyPrinter().writeValue(authFile, data);
        log.info("Сессия сохранена в {}", authFile.getAbsolutePath());
    }

    /**
     * Удаляет файл сессии.
     */
    public static void delete() {
        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        File authFile = new File(minecraftFolder, AUTH_FILE_NAME);
        if (authFile.exists() && authFile.delete()) {
            log.info("Сессия удалена: {}", authFile.getAbsolutePath());
        }
    }
}
