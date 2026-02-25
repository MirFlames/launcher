package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Артефакт библиотеки (path, url, sha1, size).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackArtifact(
        String path,
        String url,
        String sha1,
        Long size
) {}
