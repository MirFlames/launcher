package com.launcher.auth.mixin;

import com.mojang.authlib.GameProfile;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.gen.Accessor;

import java.util.UUID;

@Mixin(GameProfile.class)
public interface GameProfileAccessor {

    @Accessor("id")
    UUID launcherAuth$getId();
}
