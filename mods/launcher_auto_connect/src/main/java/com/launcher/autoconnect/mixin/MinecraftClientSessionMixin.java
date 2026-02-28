package com.launcher.autoconnect.mixin;

import com.launcher.autoconnect.LauncherAuthSession;
import net.minecraft.client.MinecraftClient;
import net.minecraft.client.session.Session;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

/**
 * Подставляет сессию из launcher-auth.json, если текущая сессия имеет дефолтный UUID
 * (00000000-0000-0000-0000-000000000000). Это исправляет ошибку "Недействительный сеанс"
 * при подключении к серверу, когда игра была запущена без корректной сессии (например,
 * или без входа в лаунчере).
 */
@Mixin(MinecraftClient.class)
public abstract class MinecraftClientSessionMixin {

    @Inject(method = "getSession", at = @At("RETURN"), cancellable = true)
    private void launcherAutoConnect$replaceSessionIfDefault(CallbackInfoReturnable<Session> cir) {
        Session current = cir.getReturnValue();
        if (current == null) return;

        if (!LauncherAuthSession.isDefaultUuid(current.getUuidOrNull())) {
            return; // Уже есть валидная сессия
        }

        LauncherAuthSession.load().ifPresent(auth -> {
            Session replacement = new Session(
                    auth.nickname(),
                    auth.sessionUuid(),
                    current.getAccessToken(),
                    current.getXuid(),
                    current.getClientId()
            );
            cir.setReturnValue(replacement);
        });
    }
}
