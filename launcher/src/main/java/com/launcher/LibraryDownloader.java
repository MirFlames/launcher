package com.launcher;

import com.launcher.dto.modpack.ModpackArtifact;
import com.launcher.dto.modpack.ModpackConfig;
import com.launcher.dto.modpack.ModpackLibrary;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URI;
import java.net.URL;
import java.util.function.Consumer;

/**
 * Скачивание отсутствующих библиотек и client.jar из modpack.json.
 */
public final class LibraryDownloader {

    private static final Logger log = LoggerFactory.getLogger(LibraryDownloader.class);
    private static final int CONNECT_TIMEOUT = 30000;
    private static final int READ_TIMEOUT = 120000;

    private LibraryDownloader() {}

    /**
     * Скачивает файл по URL в указанный путь.
     *
     * @param urlString     URL для скачивания
     * @param destination   целевой файл
     * @param expectedSize  ожидаемый размер (0 если неизвестен)
     * @param progressCallback callback прогресса (0.0–1.0), может быть null
     * @return true при успехе
     */
    public static boolean downloadFile(String urlString, File destination, long expectedSize,
                                       Consumer<Double> progressCallback) {
        try {
            URL url = URI.create(urlString).toURL();
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(CONNECT_TIMEOUT);
            conn.setReadTimeout(READ_TIMEOUT);
            conn.setRequestProperty("User-Agent", "MinecraftLauncher/1.0");

            int responseCode = conn.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                log.error("Ошибка скачивания {}: HTTP {}", urlString, responseCode);
                return false;
            }

            long contentLength = conn.getContentLengthLong();
            if (contentLength <= 0) contentLength = expectedSize;

            destination.getParentFile().mkdirs();

            try (InputStream in = conn.getInputStream();
                 FileOutputStream out = new FileOutputStream(destination)) {
                byte[] buffer = new byte[8192];
                long totalRead = 0;
                int bytesRead;
                boolean sizeKnown = contentLength > 0;
                long lastCallbackMs = 0;
                final long throttleMs = 150;

                while ((bytesRead = in.read(buffer)) != -1) {
                    out.write(buffer, 0, bytesRead);
                    totalRead += bytesRead;

                    if (progressCallback != null && sizeKnown) {
                        long now = System.currentTimeMillis();
                        if (now - lastCallbackMs >= throttleMs || totalRead >= contentLength) {
                            lastCallbackMs = now;
                            double progress = Math.min(1.0, (double) totalRead / contentLength);
                            progressCallback.accept(progress);
                        }
                    }
                }

                if (progressCallback != null && sizeKnown) progressCallback.accept(1.0);
                log.info("Скачано: {} ({} байт)", destination.getName(), totalRead);
                return true;
            }
        } catch (Exception e) {
            log.error("Ошибка скачивания {}: {}", urlString, e.getMessage(), e);
            if (destination.exists()) destination.delete();
            return false;
        }
    }

    /**
     * Скачивает библиотеку, если файл отсутствует.
     */
    public static boolean ensureLibrary(File gameDir, ModpackLibrary lib, Consumer<Double> progressCallback) {
        ModpackArtifact art = lib.artifact();
        if (art == null || art.url() == null || art.url().isBlank()) return false;

        File dest = ModpackConfigLoader.getLibraryFile(gameDir, lib);
        if (dest == null) return false;
        if (dest.exists()) return true;

        log.info("Скачивание библиотеки: {}", lib.name());
        return downloadFile(art.url(), dest, art.size() != null ? art.size() : 0, progressCallback);
    }

    /**
     * Скачивает client.jar, если отсутствует.
     */
    public static boolean ensureClientJar(File gameDir, ModpackConfig modpack, Consumer<Double> progressCallback) {
        var downloads = modpack.downloads();
        if (downloads == null || downloads.client() == null) return false;

        ModpackArtifact client = downloads.client();
        if (client.url() == null || client.url().isBlank()) return false;

        String versionId = modpack.id() != null ? modpack.id() : "modpack";
        File dest = new File(gameDir, "versions" + File.separator + versionId + File.separator + versionId + ".jar");
        if (dest.exists()) return true;

        File altDest = new File(gameDir, "versions" + File.separator + versionId + File.separator + "client.jar");
        if (altDest.exists()) return true;

        log.info("Скачивание client.jar для {}", versionId);
        dest.getParentFile().mkdirs();
        return downloadFile(client.url(), dest, client.size() != null ? client.size() : 0, progressCallback);
    }
}
