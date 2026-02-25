package com.launcher;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.JFrame;
import javax.swing.SwingUtilities;

import java.io.File;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Скачивание ассетов Minecraft (текстуры, звуки и т.д.) по индексу.
 * URL: https://resources.download.minecraft.net/{hash[:2]}/{hash}
 * Путь: assets/objects/{hash[:2]}/{hash}
 */
public final class AssetDownloader {

    private static final Logger log = LoggerFactory.getLogger(AssetDownloader.class);
    private static final ObjectMapper MAPPER = new ObjectMapper();
    private static final String ASSETS_BASE_URL = "https://resources.download.minecraft.net/";

    private AssetDownloader() {}

    /**
     * Скачивает отсутствующие ассеты по индексу.
     *
     * @param assetsDir корень папки assets (gameDir/assets)
     * @param indexFile файл индекса (assets/indexes/29.json)
     * @param parentFrame для отображения прогресса (может быть null)
     * @param launchProgress overlay прогресса запуска (если не null — используется вместо отдельного диалога)
     */
    public static void ensureAssets(File assetsDir, File indexFile, JFrame parentFrame) {
        ensureAssets(assetsDir, indexFile, parentFrame, null);
    }

    public static void ensureAssets(File assetsDir, File indexFile, JFrame parentFrame, LaunchProgress launchProgress) {
        if (!indexFile.exists()) {
            log.warn("Asset index not found: {}", indexFile.getAbsolutePath());
            return;
        }

        List<AssetEntry> toDownload = new ArrayList<>();
        try {
            JsonNode root = MAPPER.readTree(indexFile);
            JsonNode objects = root.get("objects");
            if (objects == null || !objects.isObject()) return;

            Iterator<Map.Entry<String, JsonNode>> it = objects.fields();
            while (it.hasNext()) {
                Map.Entry<String, JsonNode> e = it.next();
                JsonNode obj = e.getValue();
                if (obj == null || !obj.has("hash")) continue;

                String hash = obj.get("hash").asText();
                if (hash == null || hash.length() < 2) continue;

                File dest = new File(assetsDir, "objects" + File.separator + hash.substring(0, 2) + File.separator + hash);
                if (!dest.exists()) {
                    toDownload.add(new AssetEntry(hash, obj.has("size") ? obj.get("size").asLong() : 0));
                }
            }
        } catch (Exception e) {
            log.error("Failed to parse asset index: {}", e.getMessage(), e);
            return;
        }

        if (toDownload.isEmpty()) {
            log.info("All assets already present");
            return;
        }

        log.info("Downloading {} missing assets", toDownload.size());

        final boolean useOverlay = launchProgress != null;
        AtomicReference<ProgressBar> progressBarRef = new AtomicReference<>();
        if (!useOverlay && parentFrame != null) {
            SwingUtilities.invokeLater(() -> {
                ProgressBar pb = new ProgressBar(parentFrame, "Скачивание ассетов");
                pb.setVisible(true);
                progressBarRef.set(pb);
            });
            while (progressBarRef.get() == null) {
                try { Thread.sleep(50); } catch (InterruptedException ignored) {}
            }
        } else if (useOverlay) {
            launchProgress.setStage("Загрузка ассетов");
            launchProgress.setIndeterminate(false);
            launchProgress.setProgress(0);
        }

        ProgressBar progressBar = !useOverlay ? progressBarRef.get() : null;
        int total = toDownload.size();
        int[] done = {0};

        try {
            for (AssetEntry entry : toDownload) {
                String url = ASSETS_BASE_URL + entry.hash.substring(0, 2) + "/" + entry.hash;
                File dest = new File(assetsDir, "objects" + File.separator + entry.hash.substring(0, 2) + File.separator + entry.hash);
                dest.getParentFile().mkdirs();

                if (LibraryDownloader.downloadFile(url, dest, entry.size, null)) {
                    done[0]++;
                    int currentDone = done[0];
                    String status = String.format("Ассеты: %d / %d", currentDone, total);
                    SwingUtilities.invokeLater(() -> {
                        if (useOverlay) {
                            launchProgress.setProgress((double) currentDone / total);
                            launchProgress.setStatus(status);
                        } else if (progressBar != null) {
                            progressBar.setProgress((double) currentDone / total);
                            progressBar.setStatus(status);
                        }
                    });
                } else {
                    log.warn("Failed to download asset: {}", entry.hash);
                }
            }
        } finally {
            if (!useOverlay && progressBar != null) {
                SwingUtilities.invokeLater(() -> {
                    progressBar.setVisible(false);
                    progressBar.dispose();
                });
            }
        }

        log.info("Assets download complete: {}/{}", done[0], total);
    }

    private record AssetEntry(String hash, long size) {}
}
