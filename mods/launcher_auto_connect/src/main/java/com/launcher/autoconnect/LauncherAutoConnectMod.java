package com.launcher.autoconnect;

import net.fabricmc.api.ClientModInitializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Клиентский мод для подключения к серверу при запуске Minecraft через лаунчер.
 * Заменяет кнопки "Одиночная игра" и "Сетевая игра" на "Играть" с автоподключением.
 * IP и порт берутся из параметров запуска (--server, --port), иначе — из константы.
 */
public class LauncherAutoConnectMod implements ClientModInitializer {

    public static final String MOD_ID = "launcher_auto_connect";
    private static final Logger LOG = LoggerFactory.getLogger(MOD_ID);

    @Override
    public void onInitializeClient() {
        LOG.info("Launcher Auto Connect mod initialized");
    }
}
