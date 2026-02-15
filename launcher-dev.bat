@echo off
setlocal enabledelayedexpansion

echo ========================================
echo   Noein Video Editor - Dev Mode
echo ========================================
echo.
echo Auto-rebuilds on code changes.
echo Press Ctrl+C to stop.
echo.

:: Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"
cd /d "%SCRIPT_DIR%"

:: Check for FFmpeg
set "FFMPEG_FOUND=0"

if exist "build\bin\ffmpeg.exe" (
    echo [OK] FFmpeg found ^(bundled^)
    set "FFMPEG_FOUND=1"
    goto :LaunchDev
)

where ffmpeg >nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] FFmpeg found (system PATH)
    set "FFMPEG_FOUND=1"
    goto :LaunchDev
)

echo [WARNING] FFmpeg not found! Video processing will not work.
echo Install from https://ffmpeg.org or run launcher.bat to auto-download.
echo.

:LaunchDev
echo [LAUNCHING] Starting Noein in dev mode...
echo.
wails dev -v 2
pause
exit /b 0
