package com.launcher.autoconnect.mixin;

import com.launcher.autoconnect.AutoConnectConfig;
import net.minecraft.client.gui.screen.Screen;
import net.minecraft.client.gui.screen.TitleScreen;
import net.minecraft.client.gui.screen.multiplayer.ConnectScreen;
import net.minecraft.client.gui.widget.ButtonWidget;
import net.minecraft.client.network.ServerAddress;
import net.minecraft.client.network.ServerInfo;
import net.minecraft.text.Text;
import net.minecraft.text.TranslatableTextContent;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;

/**
 * Заменяет кнопки "Одиночная игра", "Сетевая игра" и "Realms" на одну кнопку "Играть"
 * при включённом автоподключении. Подключает к серверу из параметров запуска
 * (--server, --port) или захардкоженной константы.
 */
@Mixin(TitleScreen.class)
public abstract class TitleScreenMixin extends Screen {

    protected TitleScreenMixin(Text title) {
        super(title);
    }

    @Inject(method = "addNormalWidgets", at = @At("RETURN"))
    private void launcherAutoConnect$replaceButtons(int y, int spacingY, CallbackInfoReturnable<Integer> cir) {
        if (!AutoConnectConfig.isEnabled()) return;

        List<ButtonWidget> toRemove = new ArrayList<>();
        for (var child : this.children()) {
            if (child instanceof ButtonWidget button && isSingleplayerOrMultiplayerOrRealms(button)) {
                toRemove.add(button);
            }
        }

        for (var button : toRemove) {
            this.remove(button);
        }

        if (!toRemove.isEmpty()) {
            int buttonY = y;
            int buttonWidth = 200;
            int buttonHeight = 20;
            int buttonX = this.width / 2 - buttonWidth / 2;

            this.addDrawableChild(
                    ButtonWidget.builder(Text.translatable("launcher_auto_connect.menu.play"), btn -> connectToServer())
                            .dimensions(buttonX, buttonY, buttonWidth, buttonHeight)
                            .build()
            );
        }
    }

    private static boolean isSingleplayerOrMultiplayerOrRealms(ButtonWidget button) {
        var content = button.getMessage().getContent();
        if (content instanceof TranslatableTextContent ttc) {
            String key = ttc.getKey();
            return "menu.singleplayer".equals(key)
                    || "menu.multiplayer".equals(key)
                    || "menu.realms".equals(key)
                    || "menu.online".equals(key);  // Minecraft Realms (1.21+)
        }
        return false;
    }

    private static final long CONNECT_COOLDOWN_MS = 2000;
    private static long lastConnectAttempt = 0;

    private void connectToServer() {
        long now = System.currentTimeMillis();
        if (now - lastConnectAttempt < CONNECT_COOLDOWN_MS) return;
        lastConnectAttempt = now;

        try {
            String host = AutoConnectConfig.getServerHost();
            int port = AutoConnectConfig.getServerPort();
            if (host == null || host.isBlank()) return;

            String addressStr = host + ":" + port;
            ServerAddress address = ServerAddress.parse(addressStr);
            ServerInfo serverInfo = new ServerInfo("Launcher Server", addressStr, ServerInfo.ServerType.OTHER);

            ConnectScreen.connect(
                    (TitleScreen) (Object) this,
                    this.client,
                    address,
                    serverInfo,
                    false,
                    null
            );
        } catch (Throwable t) {
            LoggerFactory.getLogger("launcher_auto_connect").error("Ошибка при подключении к серверу", t);
        }
    }
}
