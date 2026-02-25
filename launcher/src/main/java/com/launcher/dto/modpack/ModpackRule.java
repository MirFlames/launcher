package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Правило применения аргумента (по ОС или features).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackRule(
        String action,
        ModpackRuleOs os,
        ModpackRuleFeatures features
) {}
