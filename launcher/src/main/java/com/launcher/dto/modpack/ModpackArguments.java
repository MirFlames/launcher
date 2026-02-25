package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * JVM и игровые аргументы из modpack.json.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackArguments(
        ModpackArgumentEntry[] jvm,
        ModpackArgumentEntry[] game,
        String[] default_user_jvm
) {}
