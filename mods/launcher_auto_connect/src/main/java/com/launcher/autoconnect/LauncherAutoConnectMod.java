package com.launcher.autoconnect;

import net.fabricmc.api.ClientModInitializer;
import net.fabricmc.fabric.api.client.screen.v1.ScreenEvents;
import net.minecraft.client.gui.screen.multiplayer.ConnectScreen;
import net.minecraft.client.gui.screen.TitleScreen;
import net.minecraft.client.network.ServerAddress;
import net.minecraft.client.network.ServerInfo;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Клиентский мод для автоматического подключения к серверу при запуске Minecraft через лаунчер.
 * Читает server_host и server_port из configs/launcher-config.json.
 */
public class LauncherAutoConnectMod implements ClientModInitializer {

    public static final String MOD_ID = "launcher_auto_connect";
    private static final Logger LOG = LoggerFactory.getLogger(MOD_ID);

    private static boolean hasAttemptedConnect;

    @Override
    public void onInitializeClient() {
        ScreenEvents.AFTER_INIT.register((client, screen, width, height) -> {
            if (!(screen instanceof TitleScreen)) return;
            if (!AutoConnectConfig.isEnabled()) return;
            if (hasAttemptedConnect) return;
            hasAttemptedConnect = true;

            String host = AutoConnectConfig.getServerHost();
            int port = AutoConnectConfig.getServerPort();
            if (host == null || host.isBlank()) return;

            String addressStr = host + ":" + port;
            LOG.info("Launcher Auto Connect: подключаемся к {}:{}", host, port);

            ServerAddress address = ServerAddress.parse(addressStr);
            ServerInfo serverInfo = new ServerInfo(
                    "Launcher Server",
                    addressStr,
                    ServerInfo.ServerType.OTHER
            );

            ConnectScreen.connect(
                    screen,
                    client,
                    address,
                    serverInfo,
                    false,
                    null
            );
        });
    }
}
