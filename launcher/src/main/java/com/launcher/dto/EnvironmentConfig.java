package com.launcher.dto;

public record EnvironmentConfig(
        String minecraftFolder,
        String javaExecutable,
        String nativesPath
) {}