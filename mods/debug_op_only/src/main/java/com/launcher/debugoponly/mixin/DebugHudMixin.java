package com.launcher.debugoponly.mixin;

import com.launcher.debugoponly.DebugOpOnlyMod;
import net.minecraft.client.gui.hud.DebugHud;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

/**
 * Блокирует отладочный прицел (с координатами) для обычных игроков.
 */
@Mixin(DebugHud.class)
public class DebugHudMixin {

    @Inject(method = "renderDebugCrosshair", at = @At("HEAD"), cancellable = true)
    private void debugOpOnly$blockDebugCrosshair(CallbackInfo ci) {
        if (DebugOpOnlyMod.shouldBlockDebug()) {
            ci.cancel();
        }
    }
}
