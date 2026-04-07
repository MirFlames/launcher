; Inno Setup 6 — установщик лаунчера.
;   .\generate-wizard-images.ps1
;   ISCC.exe launcher.iss /DAppVersion=1.2.3
#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif

#define MyAppName "OpenSource Minecraft Launcher"
#define MyAppPublisher "Dmitry Miroshnikov"
#define MyAppExeName "launcher.exe"

[Setup]
AppId={{8B3C9F2A-1D4E-4F6A-9C2B-7E8D5A1F0B3C}}
AppName={#MyAppName}
AppVersion={#AppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={sd}\Games\OpenSourceMinecraftLauncher
DisableProgramGroupPage=no
DefaultGroupName={#MyAppName}
OutputDir=..\..\..\
OutputBaseFilename=Launcher-Setup-{#AppVersion}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\{#MyAppExeName}
SetupLogging=yes
WizardImageFile=assets\wizard-large.bmp
WizardSmallImageFile=assets\wizard-small.bmp

[Languages]
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[Tasks]
Name: "desktopicon"; Description: "Создать ярлык на рабочем столе"; GroupDescription: "Дополнительно:"
Name: "launchapp"; Description: "Запустить Minecraft"; GroupDescription: "Дополнительно:"

[Files]
Source: "..\build\bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{commondesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "Запустить лаунчер после установки"; Flags: nowait postinstall skipifsilent; Tasks: launchapp
