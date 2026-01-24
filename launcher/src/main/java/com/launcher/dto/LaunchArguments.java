package com.launcher.dto;

import java.util.List;
import java.util.Map;

public record LaunchArguments(
        String mainClass,
        List<String> jvmArguments,
        Map<String, String> gameArguments
) {}
