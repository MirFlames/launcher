package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Условие по features (is_demo_user, has_custom_resolution и т.д.).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackRuleFeatures(
        Boolean is_demo_user,
        Boolean has_custom_resolution,
        Boolean has_quick_plays_support,
        Boolean is_quick_play_singleplayer,
        Boolean is_quick_play_multiplayer,
        Boolean is_quick_play_realms
) {}
