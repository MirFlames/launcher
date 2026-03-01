package com.launcher.nocreepersphantoms.mixin;

import net.minecraft.server.world.ServerWorld;
import net.minecraft.world.spawner.PhantomSpawner;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

/**
 * Блокирует спавн фантомов через PhantomSpawner (отдельная система от SpawnRestriction).
 * Фантомы спавнятся на 3-й день без сна — этот код обходит SpawnRestriction.canSpawn.
 */
@Mixin(PhantomSpawner.class)
public class PhantomSpawnerMixin {

    @Inject(method = "spawn", at = @At("HEAD"), cancellable = true)
    private void noCreepersPhantoms$preventPhantomSpawn(ServerWorld world, boolean spawnMonsters, CallbackInfo ci) {
        ci.cancel();
    }
}
