package com.launcher.dto;

public record MinecraftLaunchConfig(
        EnvironmentConfig environment,
        Classpath classpath,
        LaunchArguments launchArguments
) {}