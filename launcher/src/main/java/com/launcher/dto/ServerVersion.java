package com.launcher.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

/**
 * Ответ API GET /api/version — манифест версии с модами и клиентскими файлами.
 */
public record ServerVersion(
        @JsonProperty("minecraft_version") String minecraftVersion,
        @JsonProperty("mods_hash") String modsHash,
        @JsonProperty("client_files") List<ServerFile> clientFiles,
        @JsonProperty("mods") List<ServerFile> mods
) {
    /**
     * Файл из манифеста (мод или client_file).
     */
    public record ServerFile(String name, String url, String hash) {}
}
