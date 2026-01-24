package com.launcher.updater;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.MessageDigest;

/**
 * Скачивание нового launcher.exe
 */
public class UpdateDownloader {
    private static final Logger log = LoggerFactory.getLogger(UpdateDownloader.class);
    
    /**
     * Скачивает новый launcher.exe
     * @param downloadUrl URL для скачивания
     * @param tempDir временная директория
     * @return путь к скачанному файлу или null при ошибке
     */
    public static File downloadLauncher(String downloadUrl, File tempDir) {
        try {
            tempDir.mkdirs();
            File tempFile = new File(tempDir, "launcher-new.exe");
            
            log.info("Скачивание launcher.exe из: {}", downloadUrl);
            
            URL url = new URL(downloadUrl);
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(30000);
            conn.setReadTimeout(60000);
            
            int responseCode = conn.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                log.error("Ошибка скачивания: HTTP {}", responseCode);
                return null;
            }
            
            long contentLength = conn.getContentLengthLong();
            log.info("Размер файла: {} байт", contentLength);
            
            try (InputStream in = conn.getInputStream();
                 FileOutputStream out = new FileOutputStream(tempFile)) {
                
                byte[] buffer = new byte[8192];
                long totalRead = 0;
                int bytesRead;
                
                while ((bytesRead = in.read(buffer)) != -1) {
                    out.write(buffer, 0, bytesRead);
                    totalRead += bytesRead;
                    
                    if (contentLength > 0 && totalRead % (1024 * 1024) == 0) {
                        double progress = (double) totalRead / contentLength;
                        log.debug("Прогресс скачивания: {:.1f}%", progress * 100);
                    }
                }
                
                log.info("Скачивание завершено: {} байт", totalRead);
            }
            
            return tempFile;
            
        } catch (Exception e) {
            log.error("Ошибка при скачивании launcher.exe: {}", e.getMessage(), e);
            return null;
        }
    }
    
    /**
     * Проверяет SHA-256 хеш файла
     */
    public static boolean verifyHash(File file, String expectedHash) {
        if (expectedHash == null || expectedHash.isEmpty()) {
            log.warn("Хеш не указан, пропускаем проверку");
            return true;
        }
        
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            try (InputStream in = Files.newInputStream(file.toPath())) {
                byte[] buffer = new byte[8192];
                int bytesRead;
                while ((bytesRead = in.read(buffer)) != -1) {
                    digest.update(buffer, 0, bytesRead);
                }
            }
            
            byte[] hashBytes = digest.digest();
            StringBuilder sb = new StringBuilder();
            for (byte b : hashBytes) {
                sb.append(String.format("%02x", b));
            }
            String actualHash = sb.toString();
            
            // Убрать префикс sha256: если есть
            String cleanExpectedHash = expectedHash.replace("sha256:", "").toLowerCase();
            
            boolean matches = actualHash.equalsIgnoreCase(cleanExpectedHash);
            if (matches) {
                log.info("Хеш файла проверен успешно");
            } else {
                log.error("Хеш файла не совпадает. Ожидался: {}, получен: {}", cleanExpectedHash, actualHash);
            }
            
            return matches;
            
        } catch (Exception e) {
            log.error("Ошибка при проверке хеша: {}", e.getMessage(), e);
            return false;
        }
    }
}
