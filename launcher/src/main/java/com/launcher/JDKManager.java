package com.launcher;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.*;
import java.io.File;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.file.Paths;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Управление JDK: проверка наличия, скачивание и установка через Adoptium API
 */
public class JDKManager {
    private static final Logger log = LoggerFactory.getLogger(JDKManager.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();
    
    /**
     * Информация о JDK из backend API
     */
    public static class JDKInfo {
        public String version; // jdk-21.0.2
        public String relative_path; // jre_default\jdk-21.0.2
        public String java_executable; // bin\java.exe
        public boolean mandatory;
    }
    
    /**
     * Проверяет наличие JDK и устанавливает его при необходимости
     * @param minecraftFolder папка Minecraft (где находится launcher.exe)
     * @param parentFrame родительское окно для прогресс-бара
     * @return путь к java.exe или null если установка не удалась
     */
    public static String ensureJDKInstalled(File minecraftFolder, JFrame parentFrame) {
        try {
            // Получить информацию о требуемом JDK из backend
            JDKInfo jdkInfo = fetchJDKInfo();
            if (jdkInfo == null) {
                log.error("Не удалось получить информацию о JDK из backend");
                JOptionPane.showMessageDialog(parentFrame, 
                    "Не удалось получить информацию о JDK с сервера", 
                    "Ошибка", JOptionPane.ERROR_MESSAGE);
                return null;
            }
            
            // Вычислить путь к JDK
            String jdkPath = Paths.get(minecraftFolder.getAbsolutePath(), 
                jdkInfo.relative_path.replace("\\", File.separator)).toString();
            String javaExePath = Paths.get(jdkPath, 
                jdkInfo.java_executable.replace("\\", File.separator)).toString();
            
            File javaExe = new File(javaExePath);
            
            // Проверить наличие JDK
            if (javaExe.exists() && javaExe.isFile()) {
                // Проверить версию
                if (JDKVerifier.verifyJDKVersion(javaExePath, jdkInfo.version)) {
                    log.info("JDK найден и версия совпадает: {}", javaExePath);
                    return javaExePath;
                } else {
                    log.info("JDK найден, но версия отличается. Требуется обновление.");
                }
            }
            
            // JDK отсутствует или версия отличается - требуется установка
            log.info("JDK не найден или версия отличается. Требуется версия: {}", jdkInfo.version);
            log.info("Путь установки: {}", jdkPath);
            
            // Показать прогресс-бар на EDT (invokeAndWait). Вызывающий поток выполняет
            // скачивание/установку — поэтому ensureJDKInstalled должен вызываться НЕ с EDT
            // (например, startMinecraft в фоновом потоке при нажатии «Играть»).
            AtomicReference<ProgressBar> progressBarRef = new AtomicReference<>();
            try {
                SwingUtilities.invokeAndWait(() -> {
                    ProgressBar pb = new ProgressBar(parentFrame, "Установка JDK");
                    pb.setStatus("Подготовка к установке JDK...");
                    pb.setVisible(true);
                    progressBarRef.set(pb);
                });
            } catch (Exception e) {
                log.error("Не удалось показать прогресс-бар: {}", e.getMessage());
                JOptionPane.showMessageDialog(parentFrame, "Ошибка отображения прогресса: " + e.getMessage(), "Ошибка", JOptionPane.ERROR_MESSAGE);
                return null;
            }
            ProgressBar progressBar = progressBarRef.get();

            try {
                File tempDir = new File(System.getProperty("java.io.tmpdir"), "launcher-jdk");
                tempDir.mkdirs();

                updateProgress(progressBar, "Скачивание JDK...", 0.0);
                File zipFile = JDKDownloader.downloadJDK(jdkInfo.version, tempDir, p -> {
                    SwingUtilities.invokeLater(() -> {
                        progressBar.setProgress(p * 0.5);
                        progressBar.setStatus(String.format("Скачивание JDK... %.1f%%", p * 100));
                    });
                });

                if (zipFile == null || !zipFile.exists()) {
                    progressBar.showError("Ошибка скачивания JDK. Проверьте подключение к интернету.");
                    return null;
                }

                updateProgress(progressBar, "Установка JDK...", 0.5);
                File targetDir = new File(jdkPath);
                boolean installed = JDKInstaller.installJDK(zipFile, targetDir, p -> {
                    SwingUtilities.invokeLater(() -> {
                        progressBar.setProgress(0.5 + p * 0.4);
                        progressBar.setStatus(String.format("Установка JDK... %.1f%%", (0.5 + p * 0.4) * 100));
                    });
                });

                if (!installed) {
                    progressBar.showError("Ошибка установки JDK.");
                    return null;
                }

                updateProgress(progressBar, "Проверка JDK...", 0.9);

                if (!JDKVerifier.verifyJDK(javaExePath)) {
                    progressBar.showError("Установленный JDK не работает корректно.");
                    return null;
                }
                if (!JDKVerifier.verifyJDKVersion(javaExePath, jdkInfo.version)) {
                    log.warn("Версия JDK может не совпадать, но JDK работает");
                }

                updateProgress(progressBar, "JDK успешно установлен", 1.0);
                Thread.sleep(500);

                SwingUtilities.invokeAndWait(() -> {
                    progressBar.setVisible(false);
                    progressBar.dispose();
                });

                JDKInstaller.checkOldJDKVersions(minecraftFolder);
                updateConfigWithJDKPath(minecraftFolder, javaExePath);
                log.info("JDK успешно установлен: {}", javaExePath);
                return javaExePath;
            } catch (Exception e) {
                log.error("Ошибка при установке JDK: {}", e.getMessage(), e);
                progressBar.showError("Ошибка установки JDK: " + e.getMessage());
                return null;
            }
            
        } catch (Exception e) {
            log.error("Ошибка при проверке JDK: {}", e.getMessage(), e);
            JOptionPane.showMessageDialog(parentFrame, 
                "Ошибка при проверке JDK: " + e.getMessage(), 
                "Ошибка", JOptionPane.ERROR_MESSAGE);
            return null;
        }
    }
    
    private static void updateProgress(ProgressBar progressBar, String status, double progress) {
        SwingUtilities.invokeLater(() -> {
            progressBar.setStatus(status);
            progressBar.setProgress(progress);
        });
    }

    /**
     * Получает информацию о требуемом JDK из backend API
     */
    private static JDKInfo fetchJDKInfo() {
        try {
            String url = Consts.API_BASE_URL + Consts.API_JDK_INFO;
            log.info("Запрос информации о JDK: {}", url);
            
            HttpURLConnection conn = (HttpURLConnection) new URL(url).openConnection();
            conn.setRequestMethod("GET");
            conn.setConnectTimeout(10000);
            conn.setReadTimeout(10000);
            
            int responseCode = conn.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                log.error("Ошибка получения информации о JDK: HTTP {}", responseCode);
                return null;
            }
            
            JsonNode json = MAPPER.readTree(conn.getInputStream());
            JDKInfo info = new JDKInfo();
            info.version = json.get("version").asText();
            info.relative_path = json.get("relative_path").asText();
            info.java_executable = json.get("java_executable").asText();
            info.mandatory = json.has("mandatory") && json.get("mandatory").asBoolean();
            
            log.info("Получена информация о JDK: версия={}, путь={}", info.version, info.relative_path);
            return info;
            
        } catch (Exception e) {
            log.error("Ошибка при запросе информации о JDK: {}", e.getMessage(), e);
            return null;
        }
    }
    
    /**
     * Получает путь к JDK для указанной папки Minecraft
     */
    public static String getJDKPath(File minecraftFolder, String jdkVersion, String relativePath, String javaExecutable) {
        String jdkPath = Paths.get(minecraftFolder.getAbsolutePath(), 
            relativePath.replace("\\", File.separator)).toString();
        return Paths.get(jdkPath, javaExecutable.replace("\\", File.separator)).toString();
    }
    
    /**
     * Обновляет minecraft-launch-config.json с правильным путем к java.exe
     */
    private static void updateConfigWithJDKPath(File minecraftFolder, String javaExePath) {
        try {
            File configFile = new File(minecraftFolder, "configs/minecraft-launch-config.json");
            if (!configFile.exists()) {
                log.warn("Конфиг не найден, пропускаем обновление пути JDK");
                return;
            }
            
            JsonNode config = MAPPER.readTree(configFile);
            JsonNode environment = config.get("environment");
            if (environment == null) {
                log.warn("Поле 'environment' не найдено в конфиге");
                return;
            }
            
            // Обновить путь к java.exe
            ((ObjectNode) environment).put("javaExecutable", javaExePath);
            
            // Сохранить обновленный конфиг
            MAPPER.writerWithDefaultPrettyPrinter().writeValue(configFile, config);
            log.info("Конфиг обновлен с путем к JDK: {}", javaExePath);
            
        } catch (Exception e) {
            log.warn("Не удалось обновить конфиг с путем JDK: {}", e.getMessage());
            // Не критично, продолжаем работу
        }
    }
}
