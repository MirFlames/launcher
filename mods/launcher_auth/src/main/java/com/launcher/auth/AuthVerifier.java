package com.launcher.auth;

import java.io.IOException;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;

/**
 * Проверка сессии через backend API.
 */
public final class AuthVerifier {

    private static final HttpClient HTTP_CLIENT = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(5))
            .build();

    private AuthVerifier() {}

    /**
     * Проверяет, валидна ли сессия для данного никнейма.
     *
     * @param nickname   никнейм игрока
     * @param sessionUuid UUID сессии (из лаунчера)
     * @return true если сессия валидна
     */
    public static boolean verify(String nickname, String sessionUuid) {
        if (nickname == null || nickname.isBlank() || sessionUuid == null || sessionUuid.isBlank()) {
            LauncherAuthMod.LOGGER.warn("[Auth] Пустой nickname или session_uuid");
            return false;
        }

        String url = AuthConfig.getApiUrl() + "/api/auth/verify"
                + "?nickname=" + URLEncoder.encode(nickname, StandardCharsets.UTF_8)
                + "&session_uuid=" + URLEncoder.encode(sessionUuid, StandardCharsets.UTF_8);

        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(url))
                    .GET()
                    .timeout(Duration.ofSeconds(5))
                    .build();

            HttpResponse<String> response = HTTP_CLIENT.send(request, HttpResponse.BodyHandlers.ofString(StandardCharsets.UTF_8));

            if (response.statusCode() != 200) {
                LauncherAuthMod.LOGGER.warn("[Auth] Verify HTTP {} для {}: {}", response.statusCode(), nickname, response.body());
                return false;
            }

            String body = response.body();
            boolean valid = body != null && (body.contains("\"valid\":true") || body.contains("\"valid\": true"));
            LauncherAuthMod.LOGGER.info("[Auth] Verify {} для {}: {}", valid ? "OK" : "FAIL", nickname, valid);
            return valid;

        } catch (IOException e) {
            LauncherAuthMod.LOGGER.error("[Auth] Ошибка запроса verify для {}: {}", nickname, e.getMessage());
            return false;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            LauncherAuthMod.LOGGER.error("[Auth] Verify прерван для {}", nickname);
            return false;
        }
    }
}
