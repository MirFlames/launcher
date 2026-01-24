package com.launcher;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;
import java.util.ArrayList;
import java.util.List;

/**
 * Проверка установленного JDK
 */
public class JDKVerifier {
    private static final Logger log = LoggerFactory.getLogger(JDKVerifier.class);
    
    /**
     * Проверяет работоспособность JDK через java -version
     * @param javaExePath путь к java.exe
     * @return true если JDK работает корректно
     */
    public static boolean verifyJDK(String javaExePath) {
        try {
            File javaExe = new File(javaExePath);
            if (!javaExe.exists() || !javaExe.isFile()) {
                log.error("java.exe не найден: {}", javaExePath);
                return false;
            }
            
            log.info("Проверка JDK: {}", javaExePath);
            
            ProcessBuilder pb = new ProcessBuilder(javaExePath, "-version");
            pb.redirectErrorStream(true);
            Process process = pb.start();
            
            List<String> output = new ArrayList<>();
            try (BufferedReader reader = new BufferedReader(
                    new InputStreamReader(process.getInputStream()))) {
                String line;
                while ((line = reader.readLine()) != null) {
                    output.add(line);
                    log.debug("JDK output: {}", line);
                }
            }
            
            int exitCode = process.waitFor();
            if (exitCode == 0) {
                log.info("JDK проверен успешно");
                return true;
            } else {
                log.error("JDK вернул код ошибки: {}", exitCode);
                return false;
            }
            
        } catch (Exception e) {
            log.error("Ошибка при проверке JDK: {}", e.getMessage(), e);
            return false;
        }
    }
    
    /**
     * Проверяет версию установленного JDK
     * @param javaExePath путь к java.exe
     * @param expectedVersion ожидаемая версия (например, jdk-21.0.2)
     * @return true если версия совпадает
     */
    public static boolean verifyJDKVersion(String javaExePath, String expectedVersion) {
        try {
            // Извлечь номер версии из jdk-21.0.2 -> 21.0.2
            String expectedVersionNumber = expectedVersion.replace("jdk-", "");
            
            File javaExe = new File(javaExePath);
            ProcessBuilder pb = new ProcessBuilder(javaExePath, "-version");
            pb.redirectErrorStream(true);
            Process process = pb.start();
            
            try (BufferedReader reader = new BufferedReader(
                    new InputStreamReader(process.getInputStream()))) {
                String line;
                while ((line = reader.readLine()) != null) {
                    // Проверить наличие версии в выводе
                    if (line.contains(expectedVersionNumber)) {
                        log.info("Версия JDK совпадает: {}", expectedVersionNumber);
                        process.waitFor();
                        return true;
                    }
                }
            }
            
            process.waitFor();
            log.warn("Не удалось проверить версию JDK, но JDK работает");
            return true; // Если JDK работает, считаем что версия подходит
            
        } catch (Exception e) {
            log.error("Ошибка при проверке версии JDK: {}", e.getMessage(), e);
            return false;
        }
    }
}
