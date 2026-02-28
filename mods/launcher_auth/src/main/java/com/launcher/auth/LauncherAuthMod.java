package com.launcher.auth;

import net.fabricmc.api.DedicatedServerModInitializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Мод аутентификации лаунчера — ограничивает подключение к серверу
 * только для игроков, прошедших аутентификацию через Telegram-бот.
 */
public class LauncherAuthMod implements DedicatedServerModInitializer {

    public static final String MOD_ID = "launcher_auth";
    public static final Logger LOGGER = LoggerFactory.getLogger(MOD_ID);

    @Override
    public void onInitializeServer() {
        try {
            AuthConfig.load();
        } catch (Exception e) {
            throw new RuntimeException("Ошибка загрузки Launcher Auth: " + e.getMessage(), e);
        }
        AuthInvalidateCommand.register();
        LOGGER.info("Launcher Auth загружен. API: {}", AuthConfig.getApiUrl());
    }
}
