package com.launcher.dto;

/**
 * Сессия аутентификации игрока (никнейм + UUID сессии).
 */
public record AuthSession(String nickname, String sessionUuid) {
    public boolean isValid() {
        return nickname != null && !nickname.isBlank()
                && sessionUuid != null && !sessionUuid.isBlank();
    }
}
