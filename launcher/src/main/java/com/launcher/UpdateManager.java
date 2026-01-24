package com.launcher;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.*;
import java.io.File;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Optional;

/**
 * Управление обновлениями лаунчера
 */
public class UpdateManager {
    private static final Logger log = LoggerFactory.getLogger(UpdateManager.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();
    
    /**
     * Информация о версии лаунчера с сервера
     */
    public static class LauncherVersionInfo {
        public String version;
        public String download_url;
        public String hash;
        public long size;
        public String release_notes;
        public boolean mandatory;
    }
    
    /**
     * Проверяет наличие обновлений лаунчера
     * @return информация об обновлении или null если обновлений нет
     */
    public static LauncherVersionInfo checkForUpdates() {
        try {
            String url = Consts.API_BASE_URL + Consts.API_LAUNCHER_VERSION;
            log.info("Проверка обновлений лаунчера: {}", url);
            
            HttpURLConnection conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(10000);
            
            int responseCode = conn.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                log.error("Ошибка проверки обновлений: HTTP {}", responseCode);
                return null;
            }
            
            JsonNode json = MAPPER.readTree(conn.getInputStream());
            LauncherVersionInfo info = new LauncherVersionInfo();
            info.version = json.get("version").asText();
            info.download_url = json.get("download_url").asText();
            info.hash = json.has("hash") ? json.get("hash").asText() : "";
            info.size = json.has("size") ? json.get("size").asLong() : 0;
            info.release_notes = json.has("release_notes") ? json.get("release_notes").asText() : "";
            info.mandatory = json.has("mandatory") && json.get("mandatory").asBoolean();
            
            // Сравнить версии
            String currentVersion = Consts.LAUNCHER_VERSION;
            if (!currentVersion.equals(info.version)) {
                log.info("Найдено обновление: {} -> {}", currentVersion, info.version);
                return info;
            }
            
            log.info("Лаунчер актуален: {}", currentVersion);
            return null;
            
        } catch (Exception e) {
            log.error("Ошибка при проверке обновлений: {}", e.getMessage(), e);
            return null;
        }
    }
    
    /**
     * Запускает процесс обновления через updater.exe
     * @param updateInfo информация об обновлении
     * @return true если updater запущен успешно
     */
    public static boolean startUpdate(LauncherVersionInfo updateInfo) {
        try {
            // Определить папку Minecraft (где находится launcher.exe)
            File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
            File launcherExe = getLauncherExeFile();
            File updaterExe = new File(minecraftFolder, "updater.exe");
            
            if (!updaterExe.exists() || !updaterExe.isFile()) {
                log.error("updater.exe не найден: {}", updaterExe.getAbsolutePath());
                // Фатальная ошибка - показываем и закрываем лаунчер
                JOptionPane.showMessageDialog(null, 
                    "Отсутствует агент автообновления. Переустановите лаунчер.", 
                    "Фатальная ошибка", JOptionPane.ERROR_MESSAGE);
                System.exit(1);
                return false;
            }
            
            log.info("Запуск updater: {}", updaterExe.getAbsolutePath());
            log.info("Путь к launcher.exe: {}", launcherExe.getAbsolutePath());
            log.info("URL обновления: {}", updateInfo.download_url);
            
            // Запустить updater с параметрами
            ProcessBuilder pb = new ProcessBuilder(
                updaterExe.getAbsolutePath(),
                "--launcher-path", launcherExe.getAbsolutePath(),
                "--update-url", updateInfo.download_url
            );
            pb.directory(minecraftFolder);
            pb.start();
            
            log.info("Updater запущен, закрываем лаунчер");
            return true;
            
        } catch (Exception e) {
            log.error("Ошибка при запуске updater: {}", e.getMessage(), e);
            JOptionPane.showMessageDialog(null, 
                "Ошибка при запуске обновления: " + e.getMessage(), 
                "Ошибка", JOptionPane.ERROR_MESSAGE);
            return false;
        }
    }
    
    /**
     * Получает файл launcher.exe
     */
    private static File getLauncherExeFile() {
        try {
            Optional<String> cmd = ProcessHandle.current().info().command();
            if (cmd.isPresent()) {
                return new File(cmd.get());
            }
        } catch (Throwable ignored) {
        }
        // Fallback
        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        return new File(minecraftFolder, "launcher.exe");
    }
    
    /**
     * Проверяет, не запущен ли уже другой экземпляр лаунчера
     * @return true если другой экземпляр уже запущен
     */
    public static boolean isAnotherInstanceRunning() {
        try {
            String currentProcessName = getLauncherExeFile().getName();
            long currentPid = ProcessHandle.current().pid();
            
            return ProcessHandle.allProcesses()
                .filter(ph -> {
                    try {
                        Optional<String> cmd = ph.info().command();
                        if (cmd.isPresent()) {
                            String processName = new File(cmd.get()).getName();
                            return processName.equals(currentProcessName) && ph.pid() != currentPid;
                        }
                    } catch (Exception e) {
                        // Игнорировать ошибки
                    }
                    return false;
                })
                .findAny()
                .isPresent();
                
        } catch (Exception e) {
            log.warn("Не удалось проверить другие экземпляры: {}", e.getMessage());
            return false;
        }
    }
}
