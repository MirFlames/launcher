package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Метаданные индекса ассетов.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackAssetIndex(
        String id,
        Long totalSize,
        Long size,
        String url
) {}
