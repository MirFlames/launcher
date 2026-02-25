package com.launcher.auth;

import com.mojang.brigadier.arguments.StringArgumentType;
import net.fabricmc.fabric.api.command.v2.CommandRegistrationCallback;
import net.minecraft.command.permission.Permission;
import net.minecraft.command.permission.PermissionLevel;
import net.minecraft.server.command.CommandManager;
import net.minecraft.server.command.ServerCommandSource;
import net.minecraft.text.Text;

import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;

/**
 * Команда /auth invalidate &lt;nickname&gt; — обнуляет сессию игрока (только для OP).
 */
public final class AuthInvalidateCommand {

    private static final HttpClient HTTP_CLIENT = HttpClient.newBuilder()
            .connectTimeout(Duration.ofSeconds(5))
            .build();

    private AuthInvalidateCommand() {}

    public static void register() {
        CommandRegistrationCallback.EVENT.register((dispatcher, registryAccess, environment) -> {
            if (!environment.dedicated) {
                return;
            }
            dispatcher.register(
                    CommandManager.literal("auth")
                            .requires(source -> source.getPermissions().hasPermission(new Permission.Level(PermissionLevel.GAMEMASTERS)))
                            .then(CommandManager.literal("invalidate")
                                    .then(CommandManager.argument("nickname", StringArgumentType.word())
                                            .executes(context -> {
                                                String nickname = StringArgumentType.getString(context, "nickname");
                                                return invalidate(context.getSource(), nickname);
                                            })))
            );
        });
    }

    private static int invalidate(ServerCommandSource source, String nickname) {
        String url = AuthConfig.getApiUrl() + "/api/auth/invalidate"
                + "?nickname=" + URLEncoder.encode(nickname, StandardCharsets.UTF_8);

        try {
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(url))
                    .POST(HttpRequest.BodyPublishers.noBody())
                    .timeout(Duration.ofSeconds(5))
                    .build();

            HttpResponse<String> response = HTTP_CLIENT.send(request, HttpResponse.BodyHandlers.ofString(StandardCharsets.UTF_8));

            if (response.statusCode() == 200) {
                source.sendFeedback(() -> Text.literal("Сессия игрока " + nickname + " обнулена."), true);
                LauncherAuthMod.LOGGER.info("[Auth] Инвалидация по команде: nickname={}", nickname);
                return 1;
            } else {
                source.sendError(Text.literal("Ошибка API: HTTP " + response.statusCode()));
                return 0;
            }
        } catch (Exception e) {
            LauncherAuthMod.LOGGER.error("[Auth] Ошибка invalidate для {}: {}", nickname, e.getMessage());
            source.sendError(Text.literal("Ошибка: " + e.getMessage()));
            return 0;
        }
    }
}
