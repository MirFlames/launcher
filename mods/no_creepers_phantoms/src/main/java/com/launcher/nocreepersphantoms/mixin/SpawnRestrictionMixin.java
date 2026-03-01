package com.launcher.nocreepersphantoms.mixin;

import net.minecraft.entity.EntityType;
import net.minecraft.entity.SpawnReason;
import net.minecraft.entity.SpawnRestriction;
import net.minecraft.util.math.BlockPos;
import net.minecraft.util.math.random.Random;
import net.minecraft.world.ServerWorldAccess;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

/**
 * Отменяет спавн криперов и фантомов на сервере.
 */
@Mixin(SpawnRestriction.class)
public abstract class SpawnRestrictionMixin {

    @Inject(method = "canSpawn", at = @At("HEAD"), cancellable = true)
    private static void noCreepersPhantoms$preventSpawn(
            EntityType<?> type,
            ServerWorldAccess world,
            SpawnReason spawnReason,
            BlockPos pos,
            Random random,
            CallbackInfoReturnable<Boolean> cir
    ) {
        if (type == EntityType.CREEPER || type == EntityType.PHANTOM) {
            cir.setReturnValue(false);
        }
    }
}
