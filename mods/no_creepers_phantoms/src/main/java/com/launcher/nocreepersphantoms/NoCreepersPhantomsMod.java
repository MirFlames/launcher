package com.launcher.nocreepersphantoms;

import net.fabricmc.api.DedicatedServerModInitializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Серверный мод: отключает спавн криперов и фантомов.
 */
public class NoCreepersPhantomsMod implements DedicatedServerModInitializer {

    public static final String MOD_ID = "no_creepers_phantoms";
    public static final Logger LOGGER = LoggerFactory.getLogger(MOD_ID);

    @Override
    public void onInitializeServer() {
        LOGGER.info("No Creepers Phantoms загружен — спавн криперов и фантомов отключён.");
    }
}
