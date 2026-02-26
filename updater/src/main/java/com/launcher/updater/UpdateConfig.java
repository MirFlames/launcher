package com.launcher.updater;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.util.ArrayList;
import java.util.List;

/**
 * Конфигурация updater из аргументов командной строки
 */
public class UpdateConfig {
    private static final Logger log = LoggerFactory.getLogger(UpdateConfig.class);
    
    private String launcherPath;
    private String updateUrl;
    private String apiUrl;
    
    public UpdateConfig(String launcherPath, String updateUrl, String apiUrl) {
        this.launcherPath = launcherPath;
        this.updateUrl = updateUrl;
        this.apiUrl = apiUrl;
    }
    
    /**
     * Парсит аргументы командной строки
     */
    public static UpdateConfig parseArgs(String[] args) {
        String launcherPath = null;
        String updateUrl = null;
        String apiUrl = "http://62.182.138.124:80";
        
        for (int i = 0; i < args.length; i++) {
            if ("--launcher-path".equals(args[i]) && i + 1 < args.length) {
                launcherPath = args[++i];
            } else if ("--update-url".equals(args[i]) && i + 1 < args.length) {
                updateUrl = args[++i];
            } else if ("--api-url".equals(args[i]) && i + 1 < args.length) {
                apiUrl = args[++i];
            }
        }
        
        if (launcherPath == null) {
            log.error("Не указан --launcher-path");
            return null;
        }
        
        // Если updateUrl не указан, можно запросить через API
        // Но для простоты требуем его
        
        return new UpdateConfig(launcherPath, updateUrl, apiUrl);
    }
    
    public String getLauncherPath() {
        return launcherPath;
    }
    
    public String getUpdateUrl() {
        return updateUrl;
    }
    
    public String getApiUrl() {
        return apiUrl;
    }
    
    /**
     * Получает директорию где находится launcher.exe
     */
    public File getLauncherDirectory() {
        File launcherFile = new File(launcherPath);
        return launcherFile.getParentFile();
    }
}
