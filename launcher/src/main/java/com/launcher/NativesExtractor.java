package com.launcher;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.*;
import java.nio.file.*;
import java.util.Enumeration;
import java.util.zip.ZipEntry;
import java.util.zip.ZipFile;

/**
 * Извлечение нативных библиотек из JAR-файлов в папку natives.
 */
public final class NativesExtractor {

    private static final Logger log = LoggerFactory.getLogger(NativesExtractor.class);

    private NativesExtractor() {}

    /**
     * Извлекает содержимое JAR (natives-*.jar) в целевую папку.
     * Копирует только .dll, .so, .dylib, .jnilib.
     */
    public static void extract(File jarFile, File targetDir) throws IOException {
        if (!jarFile.exists()) {
            log.warn("Natives jar not found: {}", jarFile.getAbsolutePath());
            return;
        }
        targetDir.mkdirs();

        try (ZipFile zip = new ZipFile(jarFile)) {
            Enumeration<? extends ZipEntry> entries = zip.entries();
            while (entries.hasMoreElements()) {
                ZipEntry entry = entries.nextElement();
                if (entry.isDirectory()) continue;
                String name = entry.getName();
                if (!isNativeFile(name)) continue;
                File outFile = new File(targetDir, new File(name).getName());
                try (InputStream in = zip.getInputStream(entry);
                     OutputStream out = new FileOutputStream(outFile)) {
                    in.transferTo(out);
                }
                log.debug("Extracted native: {}", outFile.getName());
            }
        }
    }

    private static boolean isNativeFile(String name) {
        String lower = name.toLowerCase();
        return lower.endsWith(".dll") || lower.endsWith(".so") || lower.endsWith(".dylib") || lower.endsWith(".jnilib");
    }
}
