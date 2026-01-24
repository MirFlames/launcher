package com.launcher;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.launcher.dto.MinecraftLaunchConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.util.Optional;

public final class MinecraftConfigLoader {

    private static final Logger log = LoggerFactory.getLogger(MinecraftConfigLoader.class);
    private static final ObjectMapper MAPPER = new ObjectMapper()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, true);

    private static final String CONFIG_FILE_NAME = "configs/minecraft-launch-config.json";

    private MinecraftConfigLoader() {}

    /**
     * Определяет папку Minecraft по расположению исполняемого файла launcher.exe.
     * Папка Minecraft = директория, где находится launcher.exe.
     */
    public static File getMinecraftFolder() {
        try {
            Optional<String> cmd = ProcessHandle.current().info().command();
            if (cmd.isPresent()) {
                File exe = new File(cmd.get());
                File dir = exe.getParentFile();
                if (dir != null && dir.isDirectory()) {
                    return dir;
                }
            }
        } catch (Throwable ignored) {
            // ProcessHandle недоступен (редкие JVM / окружения)
        }
        // Fallback: используем текущую рабочую директорию
        return new File(System.getProperty("user.dir", "."));
    }

    public static MinecraftLaunchConfig load() throws IOException {
        File minecraftFolder = getMinecraftFolder();
        File configFile = new File(minecraftFolder, CONFIG_FILE_NAME);
        
        log.info("MinecraftConfigLoader: loading config from {}", configFile.getAbsolutePath());
        
        if (!configFile.exists()) {
            log.error("MinecraftConfigLoader: config file not found at {}", configFile.getAbsolutePath());
            throw new IllegalStateException("Config file not found: " + configFile.getAbsolutePath());
        }
        
        if (!configFile.isFile()) {
            log.error("MinecraftConfigLoader: path is not a file: {}", configFile.getAbsolutePath());
            throw new IllegalStateException("Config path is not a file: " + configFile.getAbsolutePath());
        }
    
        try {
            MinecraftLaunchConfig cfg = MAPPER.readValue(configFile, MinecraftLaunchConfig.class);
            log.info("MinecraftConfigLoader: loaded successfully from {}", configFile.getAbsolutePath());
            return cfg;
        } catch (IOException e) {
            log.error("MinecraftConfigLoader: error reading config: {}", e.getMessage(), e);
            throw new IllegalStateException("Error reading config: " + e.getMessage(), e);
        }
    }
}
