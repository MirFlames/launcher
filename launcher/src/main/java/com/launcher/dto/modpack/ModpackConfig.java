package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Корневая структура modpack.json (формат Mojang/Fabric).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackConfig(
        String id,
        String mainClass,
        Integer minimumLauncherVersion,
        String assets,
        ModpackArguments arguments,
        ModpackLibrary[] libraries,
        ModpackAssetIndex assetIndex,
        ModpackDownloads downloads,
        ModpackJavaVersion javaVersion,
        ModpackLogging logging
) {}
