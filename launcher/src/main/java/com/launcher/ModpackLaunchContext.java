package com.launcher;

import com.launcher.dto.modpack.ModpackRuleFeatures;

import java.util.Map;

/**
 * Контекст запуска для подстановки плейсхолдеров в аргументы игры.
 */
public record ModpackLaunchContext(
        String gameDirectory,
        String assetsRoot,
        String assetsIndexName,
        String authPlayerName,
        String versionName,
        String authUuid,
        String authAccessToken,
        String clientId,
        String authXuid,
        String versionType,
        String resolutionWidth,
        String resolutionHeight,
        String quickPlayPath,
        String quickPlaySingleplayer,
        String quickPlayMultiplayer,
        String quickPlayRealms,
        boolean isDemoUser,
        boolean hasCustomResolution,
        boolean hasQuickPlaysSupport,
        boolean isQuickPlaySingleplayer,
        boolean isQuickPlayMultiplayer,
        boolean isQuickPlayRealms
) {

    public String substitute(String value) {
        if (value == null) return null;
        return value
                .replace("${game_directory}", gameDirectory)
                .replace("${assets_root}", assetsRoot)
                .replace("${assets_index_name}", assetsIndexName)
                .replace("${auth_player_name}", authPlayerName)
                .replace("${version_name}", versionName)
                .replace("${auth_uuid}", authUuid)
                .replace("${auth_access_token}", authAccessToken)
                .replace("${clientid}", clientId)
                .replace("${auth_xuid}", authXuid)
                .replace("${version_type}", versionType)
                .replace("${resolution_width}", resolutionWidth != null ? resolutionWidth : "")
                .replace("${resolution_height}", resolutionHeight != null ? resolutionHeight : "")
                .replace("${quickPlayPath}", quickPlayPath != null ? quickPlayPath : "")
                .replace("${quickPlaySingleplayer}", quickPlaySingleplayer != null ? quickPlaySingleplayer : "")
                .replace("${quickPlayMultiplayer}", quickPlayMultiplayer != null ? quickPlayMultiplayer : "")
                .replace("${quickPlayRealms}", quickPlayRealms != null ? quickPlayRealms : "");
    }

    public boolean featureMatches(ModpackRuleFeatures features) {
        if (features == null) return false;
        if (features.is_demo_user() != null && features.is_demo_user() != isDemoUser) return false;
        if (features.has_custom_resolution() != null && features.has_custom_resolution() != hasCustomResolution) return false;
        if (features.has_quick_plays_support() != null && features.has_quick_plays_support() != hasQuickPlaysSupport) return false;
        if (features.is_quick_play_singleplayer() != null && features.is_quick_play_singleplayer() != isQuickPlaySingleplayer) return false;
        if (features.is_quick_play_multiplayer() != null && features.is_quick_play_multiplayer() != isQuickPlayMultiplayer) return false;
        if (features.is_quick_play_realms() != null && features.is_quick_play_realms() != isQuickPlayRealms) return false;
        return true;
    }

    public static ModpackLaunchContext create(String gameDir, String assetsRoot, String assetsIndexName,
                                              String playerName, String versionId) {
        String base = gameDir.replace('\\', '/');
        String assets = assetsRoot != null ? assetsRoot : base + "/assets";
        return new ModpackLaunchContext(
                base,
                assets,
                assetsIndexName != null ? assetsIndexName : "29",
                playerName != null ? playerName : "Player",
                versionId != null ? versionId : "modpack",
                "00000000-0000-0000-0000-000000000000",
                "0",
                "0",
                "0",
                "fabric",
                null,
                null,
                null,
                null,
                null,
                null,
                false,
                false,
                false,
                false,
                false,
                false
        );
    }
}
