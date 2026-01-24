package com.launcher.updater;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.*;
import java.awt.*;
import java.io.File;
import java.nio.file.Files;
import java.nio.file.StandardCopyOption;

/**
 * Замена launcher.exe и запуск обновленного лаунчера
 */
public class UpdateInstaller {
    private static final Logger log = LoggerFactory.getLogger(UpdateInstaller.class);
    
    private final UpdateConfig config;
    
    public UpdateInstaller(UpdateConfig config) {
        this.config = config;
    }
    
    /**
     * Выполняет обновление
     */
    public boolean performUpdate() {
        try {
            // 1. Проверить что лаунчер закрылся
            if (!waitForLauncherToClose()) {
                showError("Лаунчер не закрылся. Закройте его вручную и попробуйте снова.");
                return false;
            }
            
            // 2. Получить URL обновления (если не указан - через API)
            String updateUrl = config.getUpdateUrl();
            String updateHash = null;
            
            if (updateUrl == null || updateUrl.isEmpty()) {
                UpdateChecker.UpdateInfo updateInfo = UpdateChecker.checkUpdate(config.getApiUrl());
                if (updateInfo == null) {
                    showError("Недоступен сервер обновлений лаунчера");
                    return false;
                }
                updateUrl = updateInfo.downloadUrl;
                updateHash = updateInfo.hash;
            }
            
            // 3. Скачать новый exe
            File tempDir = new File(System.getProperty("java.io.tmpdir"), "launcher-update");
            tempDir.mkdirs();
            
            // Проверить доступное место на диске
            File launcherFile = new File(config.getLauncherPath());
            File launcherDir = launcherFile.getParentFile();
            long freeSpace = launcherDir.getFreeSpace();
            if (freeSpace < 100 * 1024 * 1024) { // Минимум 100 МБ
                showError("Недостаточно места на диске");
                return false;
            }
            
            File newLauncher = UpdateDownloader.downloadLauncher(updateUrl, tempDir);
            if (newLauncher == null || !newLauncher.exists()) {
                showError("Ошибка доступа к интернету");
                return false;
            }
            
            // 4. Проверить хеш
            if (!UpdateDownloader.verifyHash(newLauncher, updateHash)) {
                showError("Ошибка: скачанный файл поврежден (неверный хеш)");
                newLauncher.delete();
                return false;
            }
            
            // 5. Создать резервную копию
            File backupFile = new File(config.getLauncherDirectory(), "launcher.exe.backup");
            
            if (backupFile.exists()) {
                log.info("Перезапись существующего бэкапа: {}", backupFile.getAbsolutePath());
                backupFile.delete();
            }
            
            Files.copy(launcherFile.toPath(), backupFile.toPath(), StandardCopyOption.REPLACE_EXISTING);
            log.info("Резервная копия создана: {}", backupFile.getAbsolutePath());
            
            // 6. Заменить launcher.exe
            log.info("Замена launcher.exe...");
            try {
                Files.copy(newLauncher.toPath(), launcherFile.toPath(), StandardCopyOption.REPLACE_EXISTING);
                log.info("launcher.exe заменен успешно");
            } catch (Exception e) {
                log.error("Ошибка при замене launcher.exe: {}", e.getMessage(), e);
                // Проверить доступ к файлу
                if (!launcherFile.canWrite()) {
                    showError("Нет доступа для записи в launcher.exe. Запустите от имени администратора.");
                } else {
                    showError("Ошибка при замене launcher.exe: " + e.getMessage());
                }
                return false;
            }
            
            // 7. Попытаться запустить новый лаунчер
            if (!launchNewLauncher(launcherFile)) {
                // Ошибка запуска - восстановить бэкап
                log.error("Не удалось запустить новый лаунчер, восстанавливаем бэкап");
                Files.copy(backupFile.toPath(), launcherFile.toPath(), StandardCopyOption.REPLACE_EXISTING);
                showError("Ошибка запуска обновленного лаунчера. Восстановлена предыдущая версия.");
                return false;
            }
            
            // 8. Удалить временные файлы
            newLauncher.delete();
            tempDir.delete();
            
            log.info("Обновление завершено успешно");
            return true;
            
        } catch (Exception e) {
            log.error("Ошибка при обновлении: {}", e.getMessage(), e);
            showError("Ошибка обновления: " + e.getMessage());
            return false;
        }
    }
    
    /**
     * Ожидает закрытия лаунчера (до 10 секунд)
     */
    private boolean waitForLauncherToClose() {
        log.info("Ожидание закрытия лаунчера...");
        
        File launcherFile = new File(config.getLauncherPath());
        String launcherName = launcherFile.getName();
        
        for (int i = 0; i < 20; i++) { // 20 попыток по 500мс = 10 секунд
            try {
                boolean isRunning = isProcessRunning(launcherName);
                if (!isRunning) {
                    log.info("Лаунчер закрыт");
                    return true;
                }
                
                Thread.sleep(500);
                log.debug("Лаунчер еще запущен, ожидание... ({}/20)", i + 1);
                
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return false;
            } catch (Exception e) {
                log.warn("Ошибка при проверке процесса: {}", e.getMessage());
            }
        }
        
        log.error("Лаунчер не закрылся за 10 секунд");
        return false;
    }
    
    /**
     * Проверяет запущен ли процесс
     */
    private boolean isProcessRunning(String processName) {
        try {
            return ProcessHandle.allProcesses()
                .anyMatch(ph -> {
                    try {
                        return ph.info().command()
                            .map(cmd -> new File(cmd).getName().equals(processName))
                            .orElse(false);
                    } catch (Exception e) {
                        return false;
                    }
                });
        } catch (Exception e) {
            log.warn("Ошибка при проверке процессов: {}", e.getMessage());
            return true; // В случае ошибки считаем что процесс запущен (безопаснее)
        }
    }
    
    /**
     * Запускает новый launcher.exe
     */
    private boolean launchNewLauncher(File launcherFile) {
        try {
            log.info("Запуск нового launcher.exe: {}", launcherFile.getAbsolutePath());
            
            ProcessBuilder pb = new ProcessBuilder(launcherFile.getAbsolutePath());
            pb.directory(config.getLauncherDirectory());
            Process process = pb.start();
            
            // Подождать 3 секунды и проверить что процесс запустился
            Thread.sleep(3000);
            
            // Проверить что процесс launcher.exe запущен
            String launcherName = launcherFile.getName();
            boolean isRunning = isProcessRunning(launcherName);
            
            if (isRunning) {
                log.info("Новый лаунчер запущен успешно");
                return true;
            } else {
                log.error("Лаунчер не запустился");
                return false;
            }
            
        } catch (Exception e) {
            log.error("Ошибка при запуске нового лаунчера: {}", e.getMessage(), e);
            return false;
        }
    }
    
    /**
     * Показывает ошибку пользователю (headless режим - в консоль)
     */
    private void showError(String message) {
        log.error("ОШИБКА: {}", message);
        // В headless режиме просто логируем
        // Можно добавить GUI диалог если нужно
        try {
            if (!GraphicsEnvironment.isHeadless()) {
                JOptionPane.showMessageDialog(null, message, "Ошибка обновления", JOptionPane.ERROR_MESSAGE);
            }
        } catch (Exception e) {
            // Игнорировать ошибки GUI
        }
    }
}
