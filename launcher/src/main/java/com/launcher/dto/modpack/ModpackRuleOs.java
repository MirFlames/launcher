package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Условие по ОС: name = "windows" | "linux" | "osx".
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackRuleOs(
        String name,
        String arch
) {}
