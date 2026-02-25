package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Конфигурация логирования (Log4j2).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackLogging(
        ModpackLoggingClient client
) {}
