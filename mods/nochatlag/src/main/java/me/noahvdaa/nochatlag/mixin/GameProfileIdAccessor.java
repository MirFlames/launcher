package me.noahvdaa.nochatlag.mixin;

import com.mojang.authlib.GameProfile;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.gen.Accessor;

import java.util.UUID;

@Mixin(GameProfile.class)
public interface GameProfileIdAccessor {

    @Accessor("id")
    UUID nochatlag$getId();
}
