package com.launcher.auth.mixin;

import com.launcher.auth.AuthVerifier;
import com.mojang.authlib.GameProfile;
import net.minecraft.network.packet.c2s.login.LoginHelloC2SPacket;
import net.minecraft.server.network.ServerLoginNetworkHandler;
import net.minecraft.text.Text;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Shadow;
import org.spongepowered.asm.mixin.Unique;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

import java.util.UUID;

/**
 * Перехватывает отправку пакета успешного входа и проверяет аутентификацию.
 * Использует profileId из LoginHelloC2SPacket (session_uuid от клиента), т.к. в offline-режиме
 * сервер может подменять UUID в GameProfile на offline-UUID.
 */
@Mixin(ServerLoginNetworkHandler.class)
public abstract class ServerLoginNetworkHandlerMixin {

    @Shadow
    String profileName;

    /** profileId, отправленный клиентом в LoginHelloC2SPacket (session_uuid из лаунчера) */
    @Unique
    private UUID launcherAuth$clientProfileId;

    @Inject(method = "onHello", at = @At("TAIL"))
    private void launcherAuth$captureClientProfileId(LoginHelloC2SPacket packet, CallbackInfo ci) {
        launcherAuth$clientProfileId = packet.profileId();
    }

    @Inject(method = "sendSuccessPacket", at = @At("HEAD"), cancellable = true)
    private void launcherAuth$verifyBeforeAccept(GameProfile profile, CallbackInfo ci) {
        if (profile == null) {
            return;
        }

        String nickname = profileName != null ? profileName : "";
        // Используем profileId из пакета клиента (session_uuid), а не из GameProfile
        String sessionUuid = (launcherAuth$clientProfileId != null)
                ? launcherAuth$clientProfileId.toString()
                : (((GameProfileAccessor) (Object) profile).launcherAuth$getId() != null
                        ? ((GameProfileAccessor) (Object) profile).launcherAuth$getId().toString()
                        : null);

        if (!AuthVerifier.verify(nickname, sessionUuid)) {
            Text reason = Text.literal("§cДоступ запрещён.\n\nВойдите через лаунчер: нажмите «Войти», пройдите аутентификацию в Telegram-боте.");
            ((ServerLoginNetworkHandler) (Object) this).disconnect(reason);
            ci.cancel();
        }
    }
}
