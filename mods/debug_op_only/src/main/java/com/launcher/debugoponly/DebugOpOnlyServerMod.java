package com.launcher.debugoponly;

import io.netty.buffer.Unpooled;
import net.fabricmc.api.DedicatedServerModInitializer;
import net.fabricmc.fabric.api.networking.v1.PayloadTypeRegistry;
import net.fabricmc.fabric.api.networking.v1.ServerLoginConnectionEvents;
import net.fabricmc.fabric.api.networking.v1.ServerLoginNetworking;
import net.fabricmc.fabric.api.networking.v1.ServerPlayConnectionEvents;
import net.fabricmc.fabric.api.networking.v1.ServerPlayNetworking;
import net.minecraft.command.permission.Permission;
import net.minecraft.command.permission.PermissionLevel;
import net.minecraft.network.PacketByteBuf;
import net.minecraft.text.Text;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Серверная часть: требует мод debug_op_only на клиенте.
 * Только OP видят отладочную информацию (F3).
 */
public class DebugOpOnlyServerMod implements DedicatedServerModInitializer {

    private static final Logger LOG = LoggerFactory.getLogger(DebugOpOnlyMod.MOD_ID);

    @Override
    public void onInitializeServer() {
        PayloadTypeRegistry.playS2C().register(DebugAllowedPayload.TYPE, DebugAllowedPayload.CODEC);

        ServerLoginNetworking.registerGlobalReceiver(DebugOpOnlyMod.CHECK_CHANNEL, (server, handler, understood, buf, synchronizer, responseSender) -> {
            if (!understood) {
                Text reason = Text.literal("§cДля входа на сервер необходим мод Debug OP Only.\n\nУстановите мод debug_op_only и перезапустите игру.");
                handler.disconnect(reason);
                LOG.info("Игрок {} отключён: отсутствует мод debug_op_only", handler.getConnectionInfo());
                return;
            }
        });

        ServerLoginConnectionEvents.QUERY_START.register((handler, server, sender, synchronizer) -> {
            sender.sendPacket(DebugOpOnlyMod.CHECK_CHANNEL, new PacketByteBuf(Unpooled.buffer(0)));
        });

        ServerPlayConnectionEvents.JOIN.register((handler, sender, server) -> {
            boolean allowed = handler.getPlayer().getPermissions().hasPermission(new Permission.Level(PermissionLevel.GAMEMASTERS));
            if (ServerPlayNetworking.canSend(handler, DebugAllowedPayload.TYPE)) {
                ServerPlayNetworking.send(handler.getPlayer(), new DebugAllowedPayload(allowed));
            }
        });

        LOG.info("Debug OP Only (сервер): отладка только для OP");
    }
}
