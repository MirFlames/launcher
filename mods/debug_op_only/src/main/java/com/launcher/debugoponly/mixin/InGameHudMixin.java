package com.launcher.debugoponly.mixin;

import com.launcher.debugoponly.DebugOpOnlyMod;
import net.minecraft.client.MinecraftClient;
import net.minecraft.client.gui.DrawContext;
import net.minecraft.client.gui.hud.InGameHud;
import net.minecraft.client.render.RenderTickCounter;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Shadow;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

/**
 * Блокирует отображение отладочной информации (F3) для обычных игроков.
 * При блокировке и нажатой F3 рисует стандартный прицел вместо отладочного.
 */
@Mixin(InGameHud.class)
public abstract class InGameHudMixin {

    @Shadow
    private MinecraftClient client;

    @Shadow
    protected abstract void renderCrosshair(DrawContext context, RenderTickCounter tickCounter);

    @Inject(method = "renderDebugHud", at = @At("HEAD"), cancellable = true)
    private void debugOpOnly$blockDebugForNormalPlayers(DrawContext context, CallbackInfo ci) {
        if (DebugOpOnlyMod.shouldBlockDebug()) {
            ci.cancel();
        }
    }

    @Inject(method = "render", at = @At("TAIL"))
    private void debugOpOnly$renderStandardCrosshairWhenBlocked(DrawContext context, RenderTickCounter tickCounter, CallbackInfo ci) {
        if (client != null && client.options != null && client.options.debugOverlayKey.isPressed()
                && DebugOpOnlyMod.shouldBlockDebug()) {
            renderCrosshair(context, tickCounter);
        }
    }
}
