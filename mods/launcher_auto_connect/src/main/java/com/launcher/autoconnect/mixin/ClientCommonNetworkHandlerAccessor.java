package com.launcher.autoconnect.mixin;

import net.minecraft.client.gui.screen.Screen;
import net.minecraft.client.network.ClientCommonNetworkHandler;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.gen.Accessor;

@Mixin(ClientCommonNetworkHandler.class)
public interface ClientCommonNetworkHandlerAccessor {

    @Accessor("postDisconnectScreen")
    Screen launcherAutoConnect$getPostDisconnectScreen();
}
