package com.launcher.autoconnect.mixin;

import com.launcher.autoconnect.AutoConnectConfig;
import net.minecraft.client.MinecraftClient;
import net.minecraft.client.gui.screen.Screen;
import net.minecraft.client.gui.screen.TitleScreen;
import net.minecraft.client.gui.screen.multiplayer.MultiplayerScreen;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.ModifyVariable;

/**
 * Перехватывает setScreen — если пытаются показать список серверов (MultiplayerScreen),
 * подменяет на главное меню (устраняет моргание: главное меню → список серверов).
 * ConnectScreen НЕ подменяем — он нужен для подключения при нажатии «Играть».
 */
@Mixin(MinecraftClient.class)
public abstract class MinecraftClientMixin {

    @ModifyVariable(
            method = "setScreen",
            at = @At("HEAD"),
            argsOnly = true,
            ordinal = 0
    )
    private Screen launcherAutoConnect$redirectServerListScreen(Screen screen) {
        if (!AutoConnectConfig.isEnabled() || screen == null) return screen;
        // Только MultiplayerScreen — ConnectScreen оставляем для подключения
        if (screen instanceof MultiplayerScreen) {
            return new TitleScreen();
        }
        return screen;
    }
}
