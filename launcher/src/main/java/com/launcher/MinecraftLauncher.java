package com.launcher;

import com.launcher.dto.MinecraftLaunchConfig;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.swing.JFrame;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class MinecraftLauncher {

    private static final Logger log = LoggerFactory.getLogger(MinecraftLauncher.class);

    protected static void startMinecraft(JFrame parentFrame) throws FileNotFoundException, IOException {
        log.info("MinecraftLauncher: startMinecraft");

        File minecraftFolder = MinecraftConfigLoader.getMinecraftFolder();
        log.info("MinecraftLauncher: minecraftFolder: {}", minecraftFolder.getAbsolutePath());

        String javaExePath = JDKManager.ensureJDKInstalled(minecraftFolder, parentFrame);
        if (javaExePath == null) {
            throw new IOException("Не удалось установить или найти JDK");
        }

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

        log.info("MinecraftLauncher: starting process, mainClass={}", mainClass);
        Process proc = new ProcessBuilder(cmd)
                .directory(minecraftFolder)
                .inheritIO()
                .start();
        log.info("MinecraftLauncher: game process started, pid={}", proc.pid());
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
