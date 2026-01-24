package com.launcher;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.*;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.function.Consumer;

/**
 * Скачивание JDK с Oracle (прямая ссылка на ZIP)
 */
public class JDKDownloader {
    private static final Logger log = LoggerFactory.getLogger(JDKDownloader.class);
    
    private static final String ORACLE_JDK_BASE = "https://download.oracle.com";
    
    /**
     * Скачивает JDK по прямой ссылке на ZIP.
     * @param jdkVersion версия в формате jdk-21 или jdk-21.0.2 (для URL используется мажор, напр. 21)
     * @param tempDir временная директория для скачивания
     * @param progressCallback callback для обновления прогресса (0.0 - 1.0)
     * @return путь к скачанному ZIP файлу или null при ошибке
     */
    public static File downloadJDK(String jdkVersion, File tempDir, Consumer<Double> progressCallback) {
        try {
            String versionNumber = jdkVersion.replace("jdk-", "").trim();
            // Мажорная версия для /java/21/ (21.0.2 -> 21)
            String major = versionNumber.contains(".") ? versionNumber.split("\\.")[0] : versionNumber;
            log.info("Скачивание JDK {} (мажор {})", versionNumber, major);
            
            // Прямая ссылка на ZIP: Oracle отдаёт файл, не JSON
            String downloadUrl = String.format("%s/java/%s/latest/jdk-%s_windows-x64_bin.zip",
                ORACLE_JDK_BASE, major, major);
            
            log.info("URL: {}", downloadUrl);
            tempDir.mkdirs();
            File zipFile = new File(tempDir, "jdk-" + versionNumber + ".zip");
            
            return downloadFile(downloadUrl, zipFile, 0, progressCallback);
            
        } catch (Exception e) {
            log.error("Ошибка при скачивании JDK: {}", e.getMessage(), e);
            return null;
        }
    }
    
    /**
     * Скачивает файл с прогресс-баром
     */
    private static File downloadFile(String urlString, File destination, long expectedSize, Consumer<Double> progressCallback) {
        try {
            URL url = new URL(urlString);
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(30000);
            conn.setReadTimeout(60000);
            
            // Проверить доступность
            int responseCode = conn.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                log.error("Ошибка скачивания: HTTP {}", responseCode);
                return null;
            }
            
            long contentLength = conn.getContentLengthLong();
            if (contentLength <= 0) {
                contentLength = expectedSize;
            }
            
            log.info("Начало скачивания: {} байт", contentLength);
            
            boolean sizeKnown = contentLength > 0;
            long lastCallbackMs = 0;
            long lastReported64k = 0;
            final long throttleMs = 100;
            final long chunk64k = 64 * 1024;
            
            try (InputStream in = conn.getInputStream();
                 FileOutputStream out = new FileOutputStream(destination)) {
                
                byte[] buffer = new byte[8192];
                long totalRead = 0;
                int bytesRead;
                
                while ((bytesRead = in.read(buffer)) != -1) {
                    out.write(buffer, 0, bytesRead);
                    totalRead += bytesRead;
                    
                    if (progressCallback == null) continue;
                    if (sizeKnown) {
                        long now = System.currentTimeMillis();
                        if (now - lastCallbackMs >= throttleMs || totalRead >= contentLength) {
                            lastCallbackMs = now;
                            double progress = Math.min(1.0, (double) totalRead / contentLength);
                            progressCallback.accept(progress);
                        }
                    } else {
                        long n64 = totalRead / chunk64k;
                        if (n64 > lastReported64k) {
                            lastReported64k = n64;
                            progressCallback.accept(0.0);
                        }
                    }
                }
                
                if (progressCallback != null && sizeKnown)
                    progressCallback.accept(1.0);
                
                log.info("Скачивание завершено: {} байт", totalRead);
                return destination;
            }
            
        } catch (Exception e) {
            log.error("Ошибка при скачивании файла: {}", e.getMessage(), e);
            return null;
        }
    }
}
