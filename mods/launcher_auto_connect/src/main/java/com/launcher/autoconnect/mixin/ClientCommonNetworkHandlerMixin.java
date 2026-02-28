package com.launcher.autoconnect.mixin;

import com.launcher.autoconnect.AutoConnectConfig;
import net.minecraft.client.gui.screen.Screen;
import net.minecraft.client.gui.screen.TitleScreen;
import net.minecraft.client.network.ClientCommonNetworkHandler;
import net.minecraft.network.DisconnectionInfo;
import net.minecraft.text.Text;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

/**
 * Перехватывает создание экрана отключения и подменяет parent на TitleScreen,
 * когда игрок подключался через автоподключение (кнопка «Играть»).
 * DisconnectedScreen создаётся через createDisconnectedScreen, а не через конструктор.
 */
@Mixin(ClientCommonNetworkHandler.class)
public abstract class ClientCommonNetworkHandlerMixin {

    @Inject(method = "createDisconnectedScreen", at = @At("HEAD"), cancellable = true)
    private void launcherAutoConnect$redirectToTitleScreen(DisconnectionInfo info, CallbackInfoReturnable<Screen> cir) {
        if (!AutoConnectConfig.isEnabled()) return;

        // Всегда возвращать в главное меню при отключении (игнорируем parent)
        cir.setReturnValue(new net.minecraft.client.gui.screen.DisconnectedScreen(
                new TitleScreen(),
                Text.translatable("disconnect.lost"),
                info
        ));
        cir.cancel();
    }
}
