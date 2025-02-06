; Inno Setup Script for H4 Flow App Installer

[Setup]
AppName=h4-FlowApp
AppVersion=1.3.3
DefaultDirName={commonpf}\H4 Flow App
DefaultGroupName=H4 Flow App
OutputDir=output
OutputBaseFilename=H4FlowAppInstaller


 ; You can use the idle icon for the setup wizard

; The icon for the installer will be 'idle.ico', but you can also use the active icon or another one later.
Compression=lzma
SolidCompression=yes

[Files]
; Include the main executable and icon files
Source: "h4-flowapp.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "idle.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "active.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
; Set the application icon for the Start Menu and Desktop shortcut
Name: "{group}\h4-FlowApp"; Filename: "{app}\h4-flowapp.exe"; IconFilename: "{app}\idle.ico"
Name: "{userdesktop}\h4-FlowApp"; Filename: "{app}\h4-flowapp.exe"; IconFilename: "{app}\idle.ico"

[Registry]
; Add registry key for the app if needed (e.g., for uninstallation)
Root: HKCU; Subkey: "Software\H4 Flow App"; ValueType: string; ValueName: "Install_Dir"; ValueData: "{app}"

[Run]
; Make sure to run the app after installation
Filename: "{app}\h4-flowapp.exe"; Description: "Launch H4 Flow App"; Flags: nowait postinstall skipifsilent runhidden