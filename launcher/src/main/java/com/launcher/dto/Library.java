package com.launcher.dto;

public record Library(
        String groupId,
        String artifactId,
        String version,
        String classifier,
        String path
) {}