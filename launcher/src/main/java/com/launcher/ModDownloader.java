package com.launcher;

import com.launcher.dto.ServerVersion;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.JFrame;
import javax.swing.SwingUtilities;

import java.io.File;
import java.io.InputStream;
import java.nio.file.Files;
import java.security.MessageDigest;
import java.util.List;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Скачивание модов с сервера в папку mods/ с проверкой SHA-256.
 */
public final class ModDownloader {

    private static final Logger log = LoggerFactory.getLogger(ModDownloader.class);

    private ModDownloader() {}

    /**
     * Скачивает отсутствующие или устаревшие моды в minecraftFolder/mods/.
     *
     * @param minecraftFolder папка Minecraft
     * @param mods            список модов из /api/version
     * @param parentFrame     родительское окно для ProgressBar (может быть null)
     * @param launchProgress  overlay прогресса запуска (если не null — используется вместо отдельного диалога)
     * @return true если все моды успешно загружены
     */
    public static boolean ensureMods(File minecraftFolder, List<ServerVersion.ServerFile> mods,
                                      JFrame parentFrame) {
        return ensureMods(minecraftFolder, mods, parentFrame, null);
    }

    public static boolean ensureMods(File minecraftFolder, List<ServerVersion.ServerFile> mods,
                                      JFrame parentFrame, LaunchProgress launchProgress) {
        if (mods == null || mods.isEmpty()) {
            log.info("Список модов пуст, пропуск загрузки");
            return true;
        }

        File modsDir = new File(minecraftFolder, "mods");
        modsDir.mkdirs();

        List<ServerVersion.ServerFile> toDownload = mods.stream()
                .filter(m -> needsDownload(modsDir, m))
                .toList();

        if (toDownload.isEmpty()) {
            log.info("Все моды актуальны ({} шт.)", mods.size());
            return true;
        }

        final boolean useOverlay = launchProgress != null;
        AtomicReference<ProgressBar> progressBarRef = new AtomicReference<>();
        if (!useOverlay && parentFrame != null) {
            SwingUtilities.invokeLater(() -> {
                ProgressBar pb = new ProgressBar(parentFrame, "Скачивание модов");
                pb.setVisible(true);
                progressBarRef.set(pb);
            });
            while (progressBarRef.get() == null) {
                try {
                    Thread.sleep(50);
                } catch (InterruptedException ignored) {
                    Thread.currentThread().interrupt();
                    return false;
                }
            }
        } else if (useOverlay) {
            launchProgress.setIndeterminate(false);
            launchProgress.setProgress(0);
        }

        ProgressBar progressBar = !useOverlay ? progressBarRef.get() : null;
        int total = toDownload.size();
        int[] done = {0};

        try {
            for (ServerVersion.ServerFile mod : toDownload) {
                File dest = new File(modsDir, mod.name());
                String status = String.format("Скачивание %s (%d/%d)...", mod.name(), done[0] + 1, total);
                if (useOverlay) {
                    launchProgress.setStatus(status);
                } else if (progressBar != null) {
                    SwingUtilities.invokeLater(() -> progressBar.setStatus(status));
                }

                if (!LibraryDownloader.downloadFile(mod.url(), dest, 0, p -> {
                    double overall = (done[0] + p) / (double) total;
                    SwingUtilities.invokeLater(() -> {
                        if (useOverlay) launchProgress.setProgress(overall);
                        else if (progressBar != null) progressBar.setProgress(overall);
                    });
                })) {
                    return false;
                }

                if (mod.hash() != null && !mod.hash().isBlank()) {
                    if (!verifySha256(dest, mod.hash())) {
                        log.error("Хеш мода не совпадает: {}", mod.name());
                        if (dest.exists()) dest.delete();
                        return false;
                    }
                }

                done[0]++;
                final int d = done[0];
                SwingUtilities.invokeLater(() -> {
                    if (useOverlay) launchProgress.setProgress((double) d / total);
                    else if (progressBar != null) progressBar.setProgress((double) d / total);
                });
            }
            return true;
        } finally {
            if (!useOverlay && progressBar != null) {
                SwingUtilities.invokeLater(() -> {
                    progressBar.setVisible(false);
                    progressBar.dispose();
                });
            }
        }
    }

    private static boolean needsDownload(File modsDir, ServerVersion.ServerFile mod) {
        File file = new File(modsDir, mod.name());
        if (!file.exists()) return true;
        if (mod.hash() == null || mod.hash().isBlank()) return false;
        return !verifySha256(file, mod.hash());
    }

    private static boolean verifySha256(File file, String expectedHash) {
        if (expectedHash == null || expectedHash.isBlank()) return true;
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            try (InputStream in = Files.newInputStream(file.toPath())) {
                byte[] buffer = new byte[8192];
                int bytesRead;
                while ((bytesRead = in.read(buffer)) != -1) {
                    digest.update(buffer, 0, bytesRead);
                }
            }
            StringBuilder sb = new StringBuilder();
            for (byte b : digest.digest()) {
                sb.append(String.format("%02x", b));
            }
            String actual = sb.toString();
            String expected = expectedHash.replace("sha256:", "").toLowerCase();
            return actual.equalsIgnoreCase(expected);
        } catch (Exception e) {
            log.warn("Ошибка проверки хеша {}: {}", file.getName(), e.getMessage());
            return false;
        }
    }
}
