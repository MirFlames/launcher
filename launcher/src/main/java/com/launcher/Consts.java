package com.launcher;

public class Consts {
    // Версия лаунчера (жестко закодирована, обновляется при сборке нового exe)
    public static String LAUNCHER_VERSION = "1.0.3";
    
    // API URL для проверки обновлений
    // TODO: Переход на хранение в конфиг-файле для разделения dev/prod окружений
    // Конфиг должен находиться в {minecraftFolder}/configs/launcher-config.json
    public static String API_BASE_URL = "http://localhost:80";
    
    // Endpoints
    public static String API_LAUNCHER_VERSION = "/api/launcher/version";
    public static String API_JDK_INFO = "/api/jdk/info";
    public static String API_AUTH_INIT = "/api/auth/init";
    public static String API_AUTH_CHECK = "/api/auth/check";
    public static String API_AUTH_COMPLETE = "/api/auth/complete";
    public static String API_AUTH_VERIFY = "/api/auth/verify";
}
