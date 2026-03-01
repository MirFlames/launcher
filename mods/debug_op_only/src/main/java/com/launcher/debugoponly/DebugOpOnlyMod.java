package com.launcher.debugoponly;

import io.netty.buffer.Unpooled;
import net.fabricmc.api.ClientModInitializer;
import net.fabricmc.fabric.api.client.networking.v1.ClientLoginNetworking;
import net.fabricmc.fabric.api.client.networking.v1.ClientPlayConnectionEvents;
import net.fabricmc.fabric.api.client.networking.v1.ClientPlayNetworking;
import net.fabricmc.fabric.api.networking.v1.PayloadTypeRegistry;
import net.minecraft.network.PacketByteBuf;
import net.minecraft.util.Identifier;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.concurrent.CompletableFuture;

/**
 * Клиентский мод: отладочная информация (F3) только для игроков с OP.
 * Обычные игроки не видят отладку при нажатии F3.
 */
public class DebugOpOnlyMod implements ClientModInitializer {

    public static final String MOD_ID = "debug_op_only";
    private static final Logger LOG = LoggerFactory.getLogger(MOD_ID);

    public static final Identifier CHECK_CHANNEL = Identifier.of(MOD_ID, "check");

    /** true = показывать отладку (игрок с OP на сервере) */
    public static volatile boolean debugAllowed = false;
    /** true = получили пакет от сервера (ограничения действуют) */
    public static volatile boolean onRestrictedServer = false;

    @Override
    public void onInitializeClient() {
        PayloadTypeRegistry.playS2C().register(DebugAllowedPayload.TYPE, DebugAllowedPayload.CODEC);

        ClientLoginNetworking.registerGlobalReceiver(CHECK_CHANNEL, (client, handler, buf, callbacksConsumer) ->
                CompletableFuture.completedFuture(new PacketByteBuf(Unpooled.buffer(0))));

        ClientPlayNetworking.registerGlobalReceiver(DebugAllowedPayload.TYPE, (payload, context) -> {
            onRestrictedServer = true;
            debugAllowed = payload.allowed();
        });

        ClientPlayConnectionEvents.DISCONNECT.register((handler, client) -> {
            onRestrictedServer = false;
            debugAllowed = false;
        });

        LOG.info("Debug OP Only mod initialized");
    }

    /** true = нужно скрыть отладку (на сервере с модом и без OP) */
    public static boolean shouldBlockDebug() {
        return onRestrictedServer && !debugAllowed;
    }
}
