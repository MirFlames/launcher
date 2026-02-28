package com.launcher.autoconnect;

import net.fabricmc.loader.api.FabricLoader;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Конфигурация автоподключения. IP и порт берутся из параметров запуска Minecraft
 * (--server и --port), если лаунчер их передал. Иначе — пусто/0 (автоподключение отключено).
 */
public final class AutoConnectConfig {

    private static final Logger LOG = LoggerFactory.getLogger(LauncherAutoConnectMod.MOD_ID);

    private AutoConnectConfig() {}

    public static String getServerHost() {
        String fromArgs = parseServerFromLaunchArgs();
        if (fromArgs != null && !fromArgs.isBlank()) {
            return fromArgs.trim();
        }
        return "";
    }

    public static int getServerPort() {
        Integer fromArgs = parsePortFromLaunchArgs();
        if (fromArgs != null && fromArgs > 0) {
            return fromArgs;
        }
        return 0;
    }

    /**
     * Возвращает true, если автоподключение включено (есть хост и порт > 0).
     */
    public static boolean isEnabled() {
        String host = getServerHost();
        int port = getServerPort();
        return host != null && !host.isBlank() && port > 0;
    }

    private static String parseServerFromLaunchArgs() {
        try {
            String[] args = FabricLoader.getInstance().getLaunchArguments(true);
            if (args == null) return null;
            for (int i = 0; i < args.length - 1; i++) {
                if ("--server".equals(args[i])) {
                    return args[i + 1];
                }
            }
        } catch (Throwable e) {
            LOG.warn("Не удалось прочитать --server из аргументов запуска: {}", e.getMessage());
        }
        return null;
    }

    private static Integer parsePortFromLaunchArgs() {
        try {
            String[] args = FabricLoader.getInstance().getLaunchArguments(true);
            if (args == null) return null;
            for (int i = 0; i < args.length - 1; i++) {
                if ("--port".equals(args[i])) {
                    try {
                        return Integer.parseInt(args[i + 1]);
                    } catch (NumberFormatException ignored) {}
                }
            }
        } catch (Throwable e) {
            LOG.warn("Не удалось прочитать --port из аргументов запуска: {}", e.getMessage());
        }
        return null;
    }
}
