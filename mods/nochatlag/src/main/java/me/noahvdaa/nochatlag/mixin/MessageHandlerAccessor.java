package me.noahvdaa.nochatlag.mixin;

import com.mojang.authlib.GameProfile;
import net.minecraft.client.network.message.MessageHandler;
import net.minecraft.network.message.MessageType;
import net.minecraft.network.message.SignedMessage;
import net.minecraft.text.Text;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.gen.Invoker;

import java.time.Instant;
import java.util.UUID;

@Mixin(MessageHandler.class)
public interface MessageHandlerAccessor {

    @Invoker("processChatMessageInternal")
    boolean nochatlag$processChatMessageInternal(
            MessageType.Parameters params,
            SignedMessage message,
            Text decorated,
            GameProfile sender,
            boolean onlyShowSecureChat,
            Instant receptionTimestamp
    );

    @Invoker("extractSender")
    UUID nochatlag$extractSender(Text text);
}
