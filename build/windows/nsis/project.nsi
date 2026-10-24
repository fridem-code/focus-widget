Unicode true

####
## Focus Widget installer (docs/TASKS.md T13). Build it with `wails3 task package`,
## which builds the exe, fetches the WebView2 bootstrapper and runs makensis.
##
## By hand, after one `wails3 task package` has generated "wails_tools.nsh":
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\..\bin\focuswidget.exe project.nsi
####

## A per-user widget: installs into %LOCALAPPDATA%\Programs without UAC, like
## its settings (%APPDATA%\FocusWidget) and toast registration (HKCU).
!ifndef WAILS_INSTALL_SCOPE
    !define WAILS_INSTALL_SCOPE "user"
!endif
!define INFO_COPYRIGHT "(c) 2026 Focus Widget"
## The name the toast service registers under (app.New's Options.Name).
!define TOAST_APP_ID "Focus Widget"
!define DATA_DIR "$APPDATA\FocusWidget"

!include "wails_tools.nsh"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI2.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

# The first language is the default; the installer follows the Windows UI language.
!insertmacro MUI_LANGUAGE "Russian"
!insertmacro MUI_LANGUAGE "English"

LangString closingApp ${LANG_RUSSIAN} "Закрываю запущенный Focus Widget"
LangString closingApp ${LANG_ENGLISH} "Closing the running Focus Widget"
LangString askRemoveData ${LANG_RUSSIAN} "Удалить также настройки и журналы Focus Widget?$\r$\n$\r$\n${DATA_DIR}"
LangString askRemoveData ${LANG_ENGLISH} "Also remove Focus Widget's settings and logs?$\r$\n$\r$\n${DATA_DIR}"

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
!else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif
ShowInstDetails show # This will always show the installation details.

# The widget hides to the tray instead of closing, so a polite WM_CLOSE would
# leave it running and its exe locked: end it outright. Settings are saved as
# they change, so nothing is lost.
!macro closeApp
    DetailPrint "$(closingApp)"
    nsExec::Exec 'taskkill /F /IM "${PRODUCT_EXECUTABLE}"'
    Pop $0
    Sleep 500
!macroend

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro closeApp

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    !insertmacro closeApp

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    # Only our own files: $INSTDIR may be a folder the user picked.
    Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    # Toast registration made by the app on every start (docs/spike-results.md §6):
    # the COM activator under the GUID it generated, its AppUserModelId, the
    # notification categories and the toast icon it copied to %TEMP%.
    SetRegView 64
    ReadRegStr $0 HKCU "Software\Classes\AppUserModelId\${TOAST_APP_ID}" "CustomActivator"
    ReadRegStr $1 HKCU "Software\Classes\AppUserModelId\${TOAST_APP_ID}" "IconUri"
    ${If} $0 != ""
        DeleteRegKey HKCU "Software\Classes\CLSID\$0"
    ${EndIf}
    ${If} $1 != ""
        Delete "$1"
    ${EndIf}
    DeleteRegKey HKCU "Software\Classes\AppUserModelId\${TOAST_APP_ID}"
    DeleteRegKey HKCU "Software\${TOAST_APP_ID}\NotificationCategories"
    DeleteRegKey /ifempty HKCU "Software\${TOAST_APP_ID}"

    # Settings and logs stay unless the user says otherwise (kept when silent).
    MessageBox MB_YESNO|MB_ICONQUESTION|MB_DEFBUTTON2 "$(askRemoveData)" /SD IDNO IDNO keepData
        RMDir /r "${DATA_DIR}"
    keepData:

    !insertmacro wails.deleteUninstaller
    RMDir "$INSTDIR"
SectionEnd
