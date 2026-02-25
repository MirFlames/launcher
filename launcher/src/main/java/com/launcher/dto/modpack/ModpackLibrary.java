package com.launcher.dto.modpack;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Библиотека из modpack.json.
 * name: "groupId:artifactId:version[:classifier]"
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public record ModpackLibrary(
        String name,
        ModpackRule[] rules,
        ModpackArtifact artifact,
        ModpackArtifact[] downloads
) {}
