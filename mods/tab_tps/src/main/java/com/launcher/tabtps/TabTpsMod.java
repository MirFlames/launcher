package com.launcher.tabtps;

import net.fabricmc.api.DedicatedServerModInitializer;
import net.fabricmc.fabric.api.event.lifecycle.v1.ServerTickEvents;
import net.minecraft.network.packet.s2c.play.PlayerListHeaderS2CPacket;
import net.minecraft.server.MinecraftServer;
import net.minecraft.server.network.ServerPlayerEntity;
import net.minecraft.text.Text;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Серверный мод: отображает в TAB TPS сервера и пинг игрока.
 */
public class TabTpsMod implements DedicatedServerModInitializer {

    public static final String MOD_ID = "tab_tps";
    private static final Logger LOGGER = LoggerFactory.getLogger(MOD_ID);

    /** Обновлять TAB раз в N тиков (20 тиков = 1 секунда). */
    private static final int UPDATE_INTERVAL_TICKS = 20;

    private int tickCounter = 0;

    @Override
    public void onInitializeServer() {
        ServerTickEvents.END_SERVER_TICK.register(this::onServerTick);
        LOGGER.info("Tab TPS загружен — в списке игроков отображаются TPS и пинг.");
    }

    private void onServerTick(MinecraftServer server) {
        tickCounter++;
        if (tickCounter < UPDATE_INTERVAL_TICKS) {
            return;
        }
        tickCounter = 0;

        if (server.getPlayerManager().getCurrentPlayerCount() == 0) {
            return;
        }

        float tps = getTps(server);
        String tpsColor = tps >= 18 ? "a" : (tps >= 15 ? "e" : "c"); // зелёный / жёлтый / красный
        String tpsStr = String.format("§6TPS: §%s%.1f", tpsColor, tps);

        for (ServerPlayerEntity player : server.getPlayerManager().getPlayerList()) {
            int latencyMs = player.networkHandler.getLatency();
            String headerStr = tpsStr + "  §8|  §6Ping: §f" + latencyMs + " ms";
            Text header = Text.literal(headerStr);
            Text footer = Text.literal("");

            player.networkHandler.sendPacket(new PlayerListHeaderS2CPacket(header, footer));
        }
    }

    /**
     * TPS из среднего времени тика сервера (1.21.11).
     * Ограничено сверху 20.0.
     */
    private static float getTps(MinecraftServer server) {
        float avgTickTimeMs = server.getAverageTickTime();
        if (avgTickTimeMs <= 0) {
            return 20.0f;
        }
        float tps = 1000.0f / avgTickTimeMs;
        return Math.min(tps, 20.0f);
    }
}
