package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Ссылки на загрузки (client, server и т.д.).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackDownloads(
        ModpackArtifact client,
        ModpackArtifact client_mappings,
        ModpackArtifact server,
        ModpackArtifact server_mappings
) {}
