package com.launcher;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.launcher.dto.modpack.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

/**
 * Загрузчик конфигурации modpack.json (формат Mojang/Fabric).
 */
public final class ModpackConfigLoader {

    private static final Logger log = LoggerFactory.getLogger(ModpackConfigLoader.class);
    private static final ObjectMapper MAPPER = new ObjectMapper()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);

    private static final String MODPACK_FILE_NAME = "configs/modpack.json";

    private ModpackConfigLoader() {}

    public static ModpackConfig load(File minecraftFolder) throws IOException {
        File configFile = new File(minecraftFolder, MODPACK_FILE_NAME);
        log.info("ModpackConfigLoader: loading modpack from {}", configFile.getAbsolutePath());

        if (!configFile.exists()) {
            throw new IOException("Modpack config not found: " + configFile.getAbsolutePath());
        }

        ModpackConfig cfg = MAPPER.readValue(configFile, ModpackConfig.class);
        log.info("ModpackConfigLoader: loaded modpack id={}", cfg.id());
        return cfg;
    }

    /**
     * Проверяет, существует ли modpack.json в папке Minecraft.
     */
    public static boolean exists(File minecraftFolder) {
        return new File(minecraftFolder, MODPACK_FILE_NAME).exists();
    }

    /**
     * Определяет текущую ОС для правил: "windows", "linux", "osx".
     */
    public static String getCurrentOs() {
        String os = System.getProperty("os.name", "").toLowerCase();
        if (os.contains("win")) return "windows";
        if (os.contains("mac") || os.contains("darwin")) return "osx";
        return "linux";
    }

    /**
     * Проверяет, применяется ли правило к текущей ОС.
     */
    public static boolean ruleMatchesOs(ModpackRule rule, String currentOs) {
        if (rule == null) return true;
        ModpackRuleOs os = rule.os();
        if (os == null || os.name() == null || os.name().isEmpty()) return true;
        return os.name().equalsIgnoreCase(currentOs);
    }

    /**
     * Проверяет, применяется ли entry с правилами к текущей ОС.
     */
    public static boolean argumentEntryApplies(ModpackArgumentEntry entry, String currentOs) {
        if (entry == null || entry.values() == null) return false;
        ModpackRule[] rules = entry.rules();
        if (rules == null || rules.length == 0) return true;
        for (ModpackRule r : rules) {
            boolean osMatch = ruleMatchesOs(r, currentOs);
            if ("allow".equalsIgnoreCase(r.action()) && osMatch) return true;
            if ("disallow".equalsIgnoreCase(r.action()) && osMatch) return false;
        }
        return false;
    }

    /**
     * Собирает JVM аргументы с учётом правил ОС и подстановкой плейсхолдеров.
     */
    public static List<String> resolveJvmArguments(ModpackConfig modpack, String nativesDir, String classpath,
                                                  String launcherName, String launcherVersion) {
        List<String> result = new ArrayList<>();
        String currentOs = getCurrentOs();
        ModpackArguments args = modpack.arguments();
        if (args == null || args.jvm() == null) return result;

        for (ModpackArgumentEntry entry : args.jvm()) {
            if (!argumentEntryApplies(entry, currentOs)) continue;
            if (entry.values() == null) continue;
            for (String v : entry.values()) {
                if (v == null || v.isBlank()) continue;
                String resolved = v
                        .replace("${natives_directory}", nativesDir)
                        .replace("${classpath}", classpath)
                        .replace("${launcher_name}", launcherName)
                        .replace("${launcher_version}", launcherVersion);
                result.add(resolved);
            }
        }
        return result;
    }

    /**
     * Собирает игровые аргументы с учётом правил и подстановкой плейсхолдеров.
     * features: is_demo_user, has_custom_resolution и т.д. — по умолчанию false.
     */
    public static List<String> resolveGameArguments(ModpackConfig modpack, ModpackLaunchContext ctx) {
        List<String> result = new ArrayList<>();
        String currentOs = getCurrentOs();
        ModpackArguments args = modpack.arguments();
        if (args == null || args.game() == null) return result;

        for (ModpackArgumentEntry entry : args.game()) {
            if (!argumentEntryAppliesForGame(entry, currentOs, ctx)) continue;
            if (entry.values() == null) continue;
            for (String v : entry.values()) {
                if (v == null || v.isBlank()) continue;
                String resolved = ctx.substitute(v);
                result.add(resolved);
            }
        }
        return result;
    }

    private static boolean argumentEntryAppliesForGame(ModpackArgumentEntry entry, String currentOs,
                                                      ModpackLaunchContext ctx) {
        ModpackRule[] rules = entry.rules();
        if (rules == null || rules.length == 0) return true;
        for (ModpackRule r : rules) {
            if (r.os() != null && r.os().name() != null && !r.os().name().isEmpty()) {
                if ("allow".equalsIgnoreCase(r.action()) && r.os().name().equalsIgnoreCase(currentOs))
                    return true;
                if ("disallow".equalsIgnoreCase(r.action()) && r.os().name().equalsIgnoreCase(currentOs))
                    return false;
            }
            if (r.features() != null) {
                boolean featureMatch = ctx.featureMatches(r.features());
                if ("allow".equalsIgnoreCase(r.action()) && featureMatch) return true;
                if ("disallow".equalsIgnoreCase(r.action()) && featureMatch) return false;
            }
        }
        return false;
    }

    /**
     * Проверяет, нужна ли библиотека для текущей ОС.
     */
    public static boolean libraryApplies(ModpackLibrary lib, String currentOs) {
        ModpackRule[] rules = lib.rules();
        if (rules == null || rules.length == 0) return true;
        for (ModpackRule r : rules) {
            boolean osMatch = ruleMatchesOs(r, currentOs);
            if ("allow".equalsIgnoreCase(r.action()) && osMatch) return true;
            if ("disallow".equalsIgnoreCase(r.action()) && osMatch) return false;
        }
        return false;
    }

    /**
     * Проверяет, является ли библиотека native (имеет classifier natives-*).
     */
    public static boolean isNativeLibrary(ModpackLibrary lib) {
        if (lib.name() == null) return false;
        return lib.name().contains(":natives-");
    }

    /**
     * Возвращает полный путь к файлу библиотеки.
     */
    public static File getLibraryFile(File gameDir, ModpackLibrary lib) {
        ModpackArtifact art = lib.artifact();
        if (art == null || art.path() == null) return null;
        String path = art.path();
        if (!path.startsWith("libraries/")) path = "libraries/" + path;
        return new File(gameDir, path.replace("/", File.separator));
    }
}
