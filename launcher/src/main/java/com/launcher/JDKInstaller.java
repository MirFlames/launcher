package com.launcher;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.file.Files;
import java.util.function.Consumer;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;

/**
 * Распаковка и установка JDK из ZIP архива
 */
public class JDKInstaller {
    private static final Logger log = LoggerFactory.getLogger(JDKInstaller.class);
    
    /**
     * Распаковывает JDK ZIP архив в целевую директорию
     * @param zipFile ZIP файл JDK
     * @param targetDir целевая директория (например, {minecraftFolder}/jre_default/jdk-21.0.2)
     * @param progressCallback callback для обновления прогресса (0.0 - 1.0)
     * @return true если успешно
     */
    public static boolean installJDK(File zipFile, File targetDir, Consumer<Double> progressCallback) {
        try {
            log.info("Начало установки JDK из {} в {}", zipFile.getAbsolutePath(), targetDir.getAbsolutePath());
            
            // Создать целевую директорию
            targetDir.mkdirs();
            if (!targetDir.exists() || !targetDir.isDirectory()) {
                log.error("Не удалось создать целевую директорию: {}", targetDir.getAbsolutePath());
                return false;
            }
            
            // Подсчитать общий размер для прогресса
            long totalSize = zipFile.length();
            long extractedSize = 0;
            
            long lastProgressMs = 0;
            final long throttleMs = 100;
            
            try (ZipInputStream zis = new ZipInputStream(new FileInputStream(zipFile))) {
                ZipEntry entry;
                
                while ((entry = zis.getNextEntry()) != null) {
                    String entryName = entry.getName();
                    
                    // Пропустить корневую папку в ZIP (обычно jdk-21.0.2/)
                    // Извлекаем только содержимое
                    String relativePath = entryName;
                    if (entryName.contains("/")) {
                        int firstSlash = entryName.indexOf('/');
                        if (firstSlash < entryName.length() - 1) {
                            relativePath = entryName.substring(firstSlash + 1);
                        } else {
                            continue; // Пропустить корневую директорию
                        }
                    }
                    
                    File targetFile = new File(targetDir, relativePath.replace("/", File.separator));
                    
                    if (entry.isDirectory()) {
                        targetFile.mkdirs();
                    } else {
                        // Создать родительские директории
                        targetFile.getParentFile().mkdirs();
                        
                        // Распаковать файл
                        try (FileOutputStream fos = new FileOutputStream(targetFile)) {
                            byte[] buffer = new byte[8192];
                            int len;
                            while ((len = zis.read(buffer)) > 0) {
                                fos.write(buffer, 0, len);
                                extractedSize += len;
                                
                                if (progressCallback != null && totalSize > 0) {
                                    long now = System.currentTimeMillis();
                                    if (now - lastProgressMs >= throttleMs || extractedSize >= totalSize) {
                                        lastProgressMs = now;
                                        double progress = Math.min(1.0, (double) extractedSize / totalSize);
                                        progressCallback.accept(progress);
                                    }
                                }
                            }
                        }
                    }
                    
                    zis.closeEntry();
                }
            }
            
            log.info("JDK успешно установлен в {}", targetDir.getAbsolutePath());
            return true;
            
        } catch (Exception e) {
            log.error("Ошибка при установке JDK: {}", e.getMessage(), e);
            return false;
        }
    }
    
    /**
     * Проверяет наличие старой версии JDK и не удаляет её (оставляет для ручного удаления)
     */
    public static void checkOldJDKVersions(File minecraftFolder) {
        File jreDefaultDir = new File(minecraftFolder, "jre_default");
        if (!jreDefaultDir.exists()) {
            return;
        }
        
        File[] jdkDirs = jreDefaultDir.listFiles((dir, name) -> name.startsWith("jdk-") && new File(dir, name).isDirectory());
        if (jdkDirs != null && jdkDirs.length > 0) {
            log.info("Найдены старые версии JDK (не удаляются автоматически):");
            for (File jdkDir : jdkDirs) {
                log.info("  - {}", jdkDir.getName());
            }
        }
    }
}
