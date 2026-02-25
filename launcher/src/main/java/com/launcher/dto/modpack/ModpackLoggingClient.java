package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Клиентская конфигурация логирования.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackLoggingClient(
        String argument,
        ModpackLoggingFile file,
        String type
) {}
