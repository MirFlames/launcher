package com.launcher.dto;

import java.util.List;

public record Classpath(
        List<Library> libraries,
        List<String> extraJars
) {}