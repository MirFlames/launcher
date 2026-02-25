package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Файл конфигурации логирования.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackLoggingFile(
        String id,
        String sha1,
        Long size,
        String url
) {}
