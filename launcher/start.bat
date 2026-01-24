@echo off
setlocal

set MCFOLDER=%APPDATA%\.minecraft_family
set JAVA=%MCFOLDER%\jre_default\jdk-21.0.2\bin\javaw.exe

set CLASSPATH=^
%MCFOLDER%\libraries\org\ow2\asm\asm\9.9\asm-9.9.jar;^
%MCFOLDER%\libraries\org\ow2\asm\asm-analysis\9.9\asm-analysis-9.9.jar;^
%MCFOLDER%\libraries\org\ow2\asm\asm-commons\9.9\asm-commons-9.9.jar;^
%MCFOLDER%\libraries\org\ow2\asm\asm-tree\9.9\asm-tree-9.9.jar;^
%MCFOLDER%\libraries\org\ow2\asm\asm-util\9.9\asm-util-9.9.jar;^
%MCFOLDER%\libraries\net\fabricmc\sponge-mixin\0.17.0+mixin.0.8.7\sponge-mixin-0.17.0+mixin.0.8.7.jar;^
%MCFOLDER%\libraries\net\fabricmc\intermediary\1.21.11\intermediary-1.21.11.jar;^
%MCFOLDER%\libraries\net\fabricmc\fabric-loader\0.18.4\fabric-loader-0.18.4.jar;^
%MCFOLDER%\libraries\at\yawk\lz4\lz4-java\1.8.1\lz4-java-1.8.1.jar;^
%MCFOLDER%\libraries\com\azure\azure-json\1.4.0\azure-json-1.4.0.jar;^
%MCFOLDER%\libraries\com\github\oshi\oshi-core\6.9.0\oshi-core-6.9.0.jar;^
%MCFOLDER%\libraries\com\google\code\gson\gson\2.13.2\gson-2.13.2.jar;^
%MCFOLDER%\libraries\com\google\guava\failureaccess\1.0.3\failureaccess-1.0.3.jar;^
%MCFOLDER%\libraries\com\google\guava\guava\33.5.0-jre\guava-33.5.0-jre.jar;^
%MCFOLDER%\libraries\com\ibm\icu\icu4j\77.1\icu4j-77.1.jar;^
%MCFOLDER%\libraries\com\microsoft\azure\msal4j\1.23.1\msal4j-1.23.1.jar;^
%MCFOLDER%\libraries\com\mojang\authlib\7.0.61\authlib-7.0.61.jar;^
%MCFOLDER%\libraries\com\mojang\blocklist\1.0.10\blocklist-1.0.10.jar;^
%MCFOLDER%\libraries\com\mojang\brigadier\1.3.10\brigadier-1.3.10.jar;^
%MCFOLDER%\libraries\com\mojang\datafixerupper\9.0.19\datafixerupper-9.0.19.jar;^
%MCFOLDER%\libraries\com\mojang\jtracy\1.0.37\jtracy-1.0.37.jar;^
%MCFOLDER%\libraries\com\mojang\jtracy\1.0.37\jtracy-1.0.37-natives-windows.jar;^
%MCFOLDER%\libraries\com\mojang\logging\1.6.11\logging-1.6.11.jar;^
%MCFOLDER%\libraries\com\mojang\patchy\2.2.10\patchy-2.2.10.jar;^
%MCFOLDER%\libraries\com\mojang\text2speech\1.18.11\text2speech-1.18.11.jar;^
%MCFOLDER%\libraries\commons-codec\commons-codec\1.19.0\commons-codec-1.19.0.jar;^
%MCFOLDER%\libraries\commons-io\commons-io\2.20.0\commons-io-2.20.0.jar;^
%MCFOLDER%\libraries\io\netty\netty-buffer\4.2.7.Final\netty-buffer-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-codec-base\4.2.7.Final\netty-codec-base-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-codec-compression\4.2.7.Final\netty-codec-compression-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-codec-http\4.2.7.Final\netty-codec-http-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-common\4.2.7.Final\netty-common-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-handler\4.2.7.Final\netty-handler-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-resolver\4.2.7.Final\netty-resolver-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-transport\4.2.7.Final\netty-transport-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-transport-classes-epoll\4.2.7.Final\netty-transport-classes-epoll-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-transport-native-unix-common\4.2.7.Final\netty-transport-native-unix-common-4.2.7.Final.jar;^
%MCFOLDER%\libraries\io\netty\netty-transport-classes-kqueue\4.2.7.Final\netty-transport-classes-kqueue-4.2.7.Final.jar;^
%MCFOLDER%\libraries\it\unimi\dsi\fastutil\8.5.18\fastutil-8.5.18.jar;^
%MCFOLDER%\libraries\net\java\dev\jna\jna\5.17.0\jna-5.17.0.jar;^
%MCFOLDER%\libraries\net\java\dev\jna\jna-platform\5.17.0\jna-platform-5.17.0.jar;^
%MCFOLDER%\libraries\net\sf\jopt-simple\jopt-simple\5.0.4\jopt-simple-5.0.4.jar;^
%MCFOLDER%\libraries\org\apache\commons\commons-compress\1.28.0\commons-compress-1.28.0.jar;^
%MCFOLDER%\libraries\org\apache\commons\commons-lang3\3.19.0\commons-lang3-3.19.0.jar;^
%MCFOLDER%\libraries\org\apache\logging\log4j\log4j-api\2.25.2\log4j-api-2.25.2.jar;^
%MCFOLDER%\libraries\org\apache\logging\log4j\log4j-core\2.25.2\log4j-core-2.25.2.jar;^
%MCFOLDER%\libraries\org\apache\logging\log4j\log4j-slf4j2-impl\2.25.2\log4j-slf4j2-impl-2.25.2.jar;^
%MCFOLDER%\libraries\org\joml\joml\1.10.8\joml-1.10.8.jar;^
%MCFOLDER%\libraries\org\jspecify\jspecify\1.0.0\jspecify-1.0.0.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl\3.3.3\lwjgl-3.3.3.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl-glfw\3.3.3\lwjgl-glfw-3.3.3.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl-opengl\3.3.3\lwjgl-opengl-3.3.3.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl-stb\3.3.3\lwjgl-stb-3.3.3.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl-freetype\3.3.3\lwjgl-freetype-3.3.3.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl-jemalloc\3.3.3\lwjgl-jemalloc-3.3.3.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl-openal\3.3.3\lwjgl-openal-3.3.3.jar;^
%MCFOLDER%\libraries\org\lwjgl\lwjgl-tinyfd\3.3.3\lwjgl-tinyfd-3.3.3.jar;^
%MCFOLDER%\modpack.jar;^
%MCFOLDER%\fabric-loader.jar;^
%MCFOLDER%\libraries\org\slf4j\slf4j-api\2.0.17\slf4j-api-2.0.17.jar

start "" "%JAVA%" ^
 -Djava.library.path="%MCFOLDER%\natives" ^
 -cp "%CLASSPATH%" ^
 net.fabricmc.loader.impl.launch.knot.KnotClient ^
 --assetsDir "%MCFOLDER%\assets" ^
 --gameDir "%MCFOLDER%" ^
 --assetIndex 29 ^
 --version 1.21.11 ^
 --username OfflinePlayer

exit
