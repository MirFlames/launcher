package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Требования к версии Java.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackJavaVersion(
        String component,
        Integer majorVersion
) {}
