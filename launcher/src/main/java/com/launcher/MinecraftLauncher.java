package com.launcher;

import com.launcher.dto.AuthSession;
import com.launcher.dto.MinecraftLaunchConfig;
import com.launcher.dto.ServerVersion;
import com.launcher.dto.modpack.ModpackConfig;
import com.launcher.dto.modpack.ModpackLibrary;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.JFrame;
import javax.swing.SwingUtilities;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;

public class MinecraftLauncher {

    private static final Logger log = LoggerFactory.getLogger(MinecraftLauncher.class);

    protected static void startMinecraft(JFrame parentFrame) throws FileNotFoundException, IOException {
        log.info("MinecraftLauncher: startMinecraft");

        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        log.info("MinecraftLauncher: minecraftFolder: {}", minecraftFolder.getAbsolutePath());

        // Загрузка модов с сервера (если доступен /api/version)
        String apiBaseUrl = LauncherConfigLoader.getApiBaseUrl();
        ServerVersion version = ServerVersionClient.fetch(apiBaseUrl).orElse(null);
        if (version != null && version.mods() != null && !version.mods().isEmpty()) {
            if (!ModDownloader.ensureMods(minecraftFolder, version.mods(), parentFrame)) {
                throw new IOException("Не удалось загрузить моды с сервера");
            }
        } else if (version == null) {
            log.warn("Сервер недоступен, моды не загружены. Проверьте api_base_url в configs/launcher-config.json");
        }

        String javaExePath = JDKManager.ensureJDKInstalled(minecraftFolder, parentFrame);
        if (javaExePath == null) {
            throw new IOException("Не удалось установить или найти JDK");
        }

        ensureFullscreenEnabled(minecraftFolder);

        if (ModpackConfigLoader.exists(minecraftFolder)) {
            launchFromModpack(minecraftFolder, javaExePath, parentFrame);
        } else {
            launchFromLegacyConfig(minecraftFolder, javaExePath);
        }
    }

    /**
     * Устанавливает fullscreen:true в options.txt, чтобы Minecraft запускался в полноэкранном режиме.
     */
    private static void ensureFullscreenEnabled(File minecraftFolder) {
        File optionsFile = new File(minecraftFolder, "options.txt");
        try {
            List<String> lines;
            if (optionsFile.exists()) {
                lines = new ArrayList<>(Files.readAllLines(optionsFile.toPath(), StandardCharsets.UTF_8));
                boolean found = false;
                for (int i = 0; i < lines.size(); i++) {
                    if (lines.get(i).trim().startsWith("fullscreen:")) {
                        lines.set(i, "fullscreen:true");
                        found = true;
                        break;
                    }
                }
                if (!found) {
                    lines.add("fullscreen:true");
                }
            } else {
                optionsFile.getParentFile().mkdirs();
                lines = List.of("fullscreen:true");
            }
            Files.write(optionsFile.toPath(), lines, StandardCharsets.UTF_8);
            log.info("MinecraftLauncher: fullscreen enabled in options.txt");
        } catch (IOException e) {
            log.warn("Не удалось установить fullscreen в options.txt: {}", e.getMessage());
        }
    }

    /**
     * Запуск по modpack.json (формат Mojang/Fabric).
     */
    private static void launchFromModpack(File minecraftFolder, String javaExePath, JFrame parentFrame) throws IOException {
        ModpackConfig modpack = ModpackConfigLoader.load(minecraftFolder);
        String currentOs = ModpackConfigLoader.getCurrentOs();
        String base = minecraftFolder.getAbsolutePath();
        // Используем / для путей в JVM-аргументах (совместимо с Java/JNI на всех ОС)
        String nativesPath = (base + File.separator + "natives").replace('\\', '/');

        // Скачивание отсутствующих client.jar и библиотек
        ensureModpackFiles(minecraftFolder, modpack, currentOs, parentFrame);

        // Client JAR: versions/{id}/{id}.jar или versions/{id}/client.jar
        String versionId = modpack.id() != null ? modpack.id() : "modpack";
        File clientJar = new File(minecraftFolder, "versions" + File.separator + versionId + File.separator + versionId + ".jar");
        if (!clientJar.exists()) {
            clientJar = new File(minecraftFolder, "versions" + File.separator + versionId + File.separator + "client.jar");
        }
        if (!clientJar.exists()) {
            throw new IOException("Client JAR не найден. Проверьте папку versions/" + versionId + "/ " +
                    "(ожидается " + versionId + ".jar или client.jar). Убедитесь, что модпак полностью установлен.");
        }

        // Classpath: client + libraries (без natives)
        List<String> cp = new ArrayList<>();
        cp.add(clientJar.getAbsolutePath());

        for (ModpackLibrary lib : modpack.libraries()) {
            if (lib == null) continue;
            if (!ModpackConfigLoader.libraryApplies(lib, currentOs)) continue;

            File libFile = ModpackConfigLoader.getLibraryFile(minecraftFolder, lib);
            if (libFile != null && libFile.exists()) {
                cp.add(libFile.getAbsolutePath());
            } else if (!ModpackConfigLoader.isNativeLibrary(lib)) {
                log.warn("Library not found, skipping: {}", libFile != null ? libFile.getAbsolutePath() : lib.name());
            }
        }

        // LWJGL 3 загружает natives из JAR через SharedLibraryLoader при наличии в classpath.
        // Дополнительно извлекаем в natives/ для -Djava.library.path (fallback).
        File nativesDir = new File(nativesPath);
        nativesDir.mkdirs();
        for (ModpackLibrary lib : modpack.libraries()) {
            if (lib == null || !ModpackConfigLoader.libraryApplies(lib, currentOs)) continue;
            if (!ModpackConfigLoader.isNativeLibrary(lib)) continue;

            File libFile = ModpackConfigLoader.getLibraryFile(minecraftFolder, lib);
            if (libFile != null && libFile.exists()) {
                try {
                    NativesExtractor.extract(libFile, nativesDir);
                } catch (IOException e) {
                    log.warn("Failed to extract natives from {}: {}", libFile.getName(), e.getMessage());
                }
            }
        }

        // Скачивание индекса ассетов (assets/indexes/29.json)
        ensureAssetIndex(minecraftFolder, modpack);

        // Скачивание отсутствующих ассетов (текстуры, звуки и т.д.)
        String assetsIndexId = modpack.assets() != null ? modpack.assets() : "29";
        File assetIndexFile = new File(minecraftFolder, "assets" + File.separator + "indexes" + File.separator + assetsIndexId + ".json");
        File assetsDir = new File(minecraftFolder, "assets");
        AssetDownloader.ensureAssets(assetsDir, assetIndexFile, parentFrame);

        String classpath = String.join(File.pathSeparator, cp);
        if (cp.size() <= 1) {
            throw new IOException("Classpath пуст или содержит только client.jar. Проверьте папку libraries.");
        }

        // JVM args из modpack
        List<String> jvmArgs = ModpackConfigLoader.resolveJvmArguments(
                modpack, nativesPath, classpath,
                "custom", Consts.LAUNCHER_VERSION);

        // Game args
        String assetsRoot = base + File.separator + "assets";
        String assetsIndex = modpack.assets() != null ? modpack.assets() : "29";
        AuthSession auth = AuthService.getSession().orElse(null);
        String playerName = auth != null ? auth.nickname() : "Player";
        String sessionUuid = auth != null ? auth.sessionUuid() : null;
        ModpackLaunchContext ctx = ModpackLaunchContext.create(
                base, assetsRoot, assetsIndex, playerName, modpack.id(),
                sessionUuid, sessionUuid);
        List<String> gameArgs = ModpackConfigLoader.resolveGameArguments(modpack, ctx);

        String mainClass = modpack.mainClass();
        if (mainClass == null || mainClass.isBlank()) {
            throw new IOException("В modpack.json не указан mainClass");
        }

        List<String> cmd = new ArrayList<>();
        cmd.add(javaExePath);
        cmd.addAll(jvmArgs);
        cmd.add(mainClass);
        cmd.addAll(gameArgs);

        log.info("MinecraftLauncher: starting process (modpack), mainClass={}", mainClass);
        Process proc = new ProcessBuilder(cmd)
                .directory(minecraftFolder)
                .inheritIO()
                .start();
        log.info("MinecraftLauncher: game process started, pid={}", proc.pid());
    }

    /**
     * Запуск по minecraft-launch-config.json (legacy).
     */
    private static void launchFromLegacyConfig(File minecraftFolder, String javaExePath) throws IOException {
        MinecraftLaunchConfig cfg = MinecraftConfigLoader.load();
        log.info("MinecraftLauncher: javaPath: {}", javaExePath);

        String base = minecraftFolder.getAbsolutePath();
        String baseNorm = base.replace('\\', '/');
        String nativesPath = base + File.separator + "natives";

        // Classpath: libraries + extraJars (относительно minecraftFolder)
        List<String> cp = new ArrayList<>();
        if (cfg.classpath() != null) {
            if (cfg.classpath().libraries() != null) {
                for (var lib : cfg.classpath().libraries()) {
                    if (lib.path() != null && !lib.path().isBlank()) {
                        String p = resolvePath(lib.path(), baseNorm);
                        if (new File(p).exists()) cp.add(p);
                        else log.warn("Library not found, skipping: {}", p);
                    }
                }
            }
            if (cfg.classpath().extraJars() != null) {
                for (String name : cfg.classpath().extraJars()) {
                    if (name == null || name.isBlank()) continue;
                    File f = new File(minecraftFolder, name);
                    if (f.exists()) cp.add(f.getAbsolutePath());
                    else log.warn("Extra jar not found, skipping: {}", f.getAbsolutePath());
                }
            }
        }
        if (cp.isEmpty()) {
            throw new IOException("Classpath пуст: проверьте libraries и extraJars в minecraft-launch-config.json");
        }
        String classpath = String.join(File.pathSeparator, cp);

        // JVM args: подставляем java.library.path и minecraft paths
        List<String> jvmArgs = new ArrayList<>();
        if (cfg.launchArguments() != null && cfg.launchArguments().jvmArguments() != null) {
            for (String a : cfg.launchArguments().jvmArguments()) {
                if (a == null || a.isBlank()) continue;
                if (a.startsWith("-Djava.library.path="))
                    a = "-Djava.library.path=" + nativesPath;
                else
                    a = a.replace("C:/Users/User/AppData/Roaming/.minecraft", baseNorm).replace('/', File.separatorChar);
                jvmArgs.add(a);
            }
        }
        if (jvmArgs.stream().noneMatch(x -> x.startsWith("-Djava.library.path=")))
            jvmArgs.add("-Djava.library.path=" + nativesPath);

        String mainClass = cfg.launchArguments() != null && cfg.launchArguments().mainClass() != null
                ? cfg.launchArguments().mainClass().trim()
                : null;
        if (mainClass == null || mainClass.isEmpty()) {
            throw new IOException("В конфиге не указан mainClass");
        }

        // Game args: --key value (Fabric KnotClient)
        List<String> gameArgs = new ArrayList<>();
        if (cfg.launchArguments() != null && cfg.launchArguments().gameArguments() != null) {
            for (Map.Entry<String, String> e : cfg.launchArguments().gameArguments().entrySet()) {
                if (e.getKey() == null || e.getValue() == null) continue;
                String v = e.getValue()
                        .replace("C:/Users/User/AppData/Roaming/.minecraft", baseNorm)
                        .replace('/', File.separatorChar);
                gameArgs.add("--" + e.getKey());
                gameArgs.add(v);
            }
        }

        List<String> cmd = new ArrayList<>();
        cmd.add(javaExePath);
        cmd.addAll(jvmArgs);
        cmd.add("-cp");
        cmd.add(classpath);
        cmd.add(mainClass);
        cmd.addAll(gameArgs);

        log.info("MinecraftLauncher: starting process (legacy), mainClass={}", mainClass);
        Process proc = new ProcessBuilder(cmd)
                .directory(minecraftFolder)
                .inheritIO()
                .start();
        log.info("MinecraftLauncher: game process started, pid={}", proc.pid());
    }

    /**
     * Скачивает индекс ассетов (assets/indexes/{id}.json) при отсутствии.
     */
    private static void ensureAssetIndex(File minecraftFolder, ModpackConfig modpack) {
        var assetIndex = modpack.assetIndex();
        if (assetIndex == null || assetIndex.url() == null || assetIndex.url().isBlank()) return;

        String id = assetIndex.id() != null ? assetIndex.id() : modpack.assets();
        if (id == null || id.isBlank()) id = "29";

        File indexFile = new File(minecraftFolder, "assets" + File.separator + "indexes" + File.separator + id + ".json");
        if (indexFile.exists()) return;

        log.info("Скачивание индекса ассетов: {}", id);
        indexFile.getParentFile().mkdirs();
        LibraryDownloader.downloadFile(assetIndex.url(), indexFile, assetIndex.size() != null ? assetIndex.size() : 0, null);
    }

    /**
     * Скачивает отсутствующие client.jar и библиотеки.
     */
    private static void ensureModpackFiles(File minecraftFolder, ModpackConfig modpack, String currentOs,
                                           JFrame parentFrame) throws IOException {
        String versionId = modpack.id() != null ? modpack.id() : "modpack";
        File clientJar = new File(minecraftFolder, "versions" + File.separator + versionId + File.separator + versionId + ".jar");
        File clientJarAlt = new File(minecraftFolder, "versions" + File.separator + versionId + File.separator + "client.jar");
        boolean needClientJar = !clientJar.exists() && !clientJarAlt.exists();
        if (needClientJar && (modpack.downloads() == null || modpack.downloads().client() == null
                || modpack.downloads().client().url() == null || modpack.downloads().client().url().isBlank())) {
            needClientJar = false; // нет URL для скачивания
        }

        List<ModpackLibrary> missingLibs = new ArrayList<>();
        for (ModpackLibrary lib : modpack.libraries()) {
            if (lib == null || !ModpackConfigLoader.libraryApplies(lib, currentOs)) continue;
            if (lib.artifact() == null || lib.artifact().url() == null || lib.artifact().url().isBlank()) continue;
            File dest = ModpackConfigLoader.getLibraryFile(minecraftFolder, lib);
            if (dest != null && !dest.exists()) missingLibs.add(lib);
        }

        int total = (needClientJar ? 1 : 0) + missingLibs.size();
        if (total == 0) return;

        AtomicReference<ProgressBar> progressBarRef = new AtomicReference<>();
        SwingUtilities.invokeLater(() -> {
            ProgressBar pb = new ProgressBar(parentFrame, "Скачивание файлов модпака");
            pb.setVisible(true);
            progressBarRef.set(pb);
        });
        while (progressBarRef.get() == null) {
            try { Thread.sleep(50); } catch (InterruptedException ignored) {}
        }
        ProgressBar progressBar = progressBarRef.get();

        try {
            int[] done = {0};
            if (needClientJar) {
                progressBar.setStatus("Скачивание client.jar...");
                if (!LibraryDownloader.ensureClientJar(minecraftFolder, modpack, p ->
                        SwingUtilities.invokeLater(() -> {
                            progressBar.setProgress((done[0] + p) / (double) total);
                            progressBar.setStatus(String.format("Скачивание client.jar... %.0f%%", p * 100));
                        }))) {
                    throw new IOException("Не удалось скачать client.jar");
                }
                done[0]++;
            }

            for (ModpackLibrary lib : missingLibs) {
                String name = lib.name();
                int completedBefore = done[0];
                progressBar.setStatus(String.format("Скачивание %s (%d/%d)...", name, completedBefore + 1, total));
                if (!LibraryDownloader.ensureLibrary(minecraftFolder, lib, p ->
                        SwingUtilities.invokeLater(() ->
                                progressBar.setProgress((completedBefore + p) / (double) total)))) {
                    throw new IOException("Не удалось скачать библиотеку: " + name);
                }
                done[0]++;
                SwingUtilities.invokeLater(() -> progressBar.setProgress((double) done[0] / total));
            }
        } finally {
            SwingUtilities.invokeLater(() -> {
                progressBar.setVisible(false);
                progressBar.dispose();
            });
        }
    }

    private static String resolvePath(String path, String baseNorm) {
        String p = path.replace('\\', '/');
        String prefix = "C:/Users/User/AppData/Roaming/.minecraft";
        if (p.startsWith(prefix)) {
            p = baseNorm + p.substring(prefix.length());
        }
        return p.replace('/', File.separatorChar);
    }
}
