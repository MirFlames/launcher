package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Элемент аргументов (JVM или game) с опциональными правилами.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackArgumentEntry(
        String[] values,
        ModpackRule[] rules
) {}
