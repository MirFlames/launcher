package com.launcher.debugoponly;

import net.minecraft.network.RegistryByteBuf;
import net.minecraft.network.codec.PacketCodec;
import net.minecraft.network.codec.PacketCodecs;
import net.minecraft.network.packet.CustomPayload;
import net.minecraft.util.Identifier;

/**
 * Пакет сервер→клиент: разрешена ли отладка (OP).
 */
public record DebugAllowedPayload(boolean allowed) implements CustomPayload {

    public static final Identifier ID = Identifier.of(DebugOpOnlyMod.MOD_ID, "allowed");
    public static final CustomPayload.Id<DebugAllowedPayload> TYPE = new CustomPayload.Id<>(ID);
    public static final PacketCodec<RegistryByteBuf, DebugAllowedPayload> CODEC = PacketCodec.tuple(
            PacketCodecs.BOOLEAN,
            DebugAllowedPayload::allowed,
            DebugAllowedPayload::new
    );

    @Override
    public Id<? extends CustomPayload> getId() {
        return TYPE;
    }
}
