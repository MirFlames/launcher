package me.noahvdaa.nochatlag.mixin;

import com.mojang.authlib.GameProfile;
import net.minecraft.client.MinecraftClient;
import net.minecraft.client.network.message.MessageHandler;
import net.minecraft.network.message.MessageType;
import net.minecraft.network.message.SignedMessage;
import net.minecraft.text.Text;
import org.spongepowered.asm.mixin.Final;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Shadow;
import org.spongepowered.asm.mixin.Unique;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

import java.time.Instant;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Moves the blocklist check (shouldBlockMessages) off the render thread to a background thread,
 * fixing lag spikes when the chat is rendered and Mojang's blocked users API is queried (Mojira WEB-5587).
 */
@Mixin(MessageHandler.class)
public abstract class MessageHandlerMixin {

    @Shadow
    @Final
    private MinecraftClient client;

    @Unique
    private static final ExecutorService nochatlag$executor = Executors.newSingleThreadExecutor(r -> {
        Thread t = new Thread(r, "NoChatLag-blocklist");
        t.setDaemon(true);
        return t;
    });

    @Inject(method = "processChatMessageInternal", at = @At("HEAD"), cancellable = true)
    private void nochatlag$processChatMessageAsync(
            MessageType.Parameters params,
            SignedMessage message,
            Text decorated,
            GameProfile sender,
            boolean onlyShowSecureChat,
            Instant receptionTimestamp,
            CallbackInfoReturnable<Boolean> cir
    ) {
        // Resolve sender UUID on this thread (extractSender reads Text)
        UUID toCheck = (client.options != null && client.options.getHideMatchedNames().getValue())
                ? nochatlag$extractSender(decorated)
                : ((GameProfileIdAccessor) (Object) sender).nochatlag$getId();
        // Run blocklist check on background thread so it doesn't freeze the game
        nochatlag$executor.execute(() -> {
            boolean blocked = client.shouldBlockMessages(toCheck);
            if (!blocked) {
                client.execute(() -> nochatlag$invokeProcessChatMessageInternal(
                        params, message, decorated, sender, onlyShowSecureChat, receptionTimestamp
                ));
            }
        });
        cir.setReturnValue(false);
        cir.cancel();
    }

    @Unique
    private UUID nochatlag$extractSender(Text text) {
        return ((MessageHandlerAccessor) this).nochatlag$extractSender(text);
    }

    @Unique
    private boolean nochatlag$invokeProcessChatMessageInternal(
            MessageType.Parameters params,
            SignedMessage message,
            Text decorated,
            GameProfile sender,
            boolean onlyShowSecureChat,
            Instant receptionTimestamp
    ) {
        return ((MessageHandlerAccessor) this).nochatlag$processChatMessageInternal(
                params, message, decorated, sender, onlyShowSecureChat, receptionTimestamp
        );
    }
}
