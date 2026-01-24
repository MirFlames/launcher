package com.launcher.updater;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.util.Arrays;

/**
 * Главный класс updater - точка входа
 */
public class UpdaterMain {
    private static final Logger log = LoggerFactory.getLogger(UpdaterMain.class);
    
    public static void main(String[] args) {
        // Настройка логирования
        File logFile = new File(System.getProperty("java.io.tmpdir"), "updater.log");
        System.setProperty("LOG_FILE", logFile.getAbsolutePath());
        
        log.info("Updater запущен с аргументами: {}", Arrays.toString(args));
        
        try {
            // Парсинг аргументов
            UpdateConfig config = UpdateConfig.parseArgs(args);
            if (config == null) {
                log.error("Неверные аргументы командной строки");
                System.exit(1);
                return;
            }
            
            log.info("Конфигурация updater:");
            log.info("  Launcher path: {}", config.getLauncherPath());
            log.info("  Update URL: {}", config.getUpdateUrl());
            log.info("  API URL: {}", config.getApiUrl());
            
            // Выполнить обновление
            UpdateInstaller installer = new UpdateInstaller(config);
            boolean success = installer.performUpdate();
            
            if (success) {
                log.info("Обновление завершено успешно");
                System.exit(0);
            } else {
                log.error("Обновление завершилось с ошибкой");
                System.exit(1);
            }
            
        } catch (Exception e) {
            log.error("Критическая ошибка updater: {}", e.getMessage(), e);
            System.exit(1);
        }
    }
}
