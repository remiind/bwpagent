@echo off
setlocal
cd /d "%~dp0"
set GOOS=windows
set GOARCH=amd64
go build -trimpath -ldflags="-H windowsgui -s -w" -o bwpageant.exe .
if errorlevel 1 (echo Build fehlgeschlagen & pause & exit /b 1)
echo Fertig: bwpageant.exe
pause
