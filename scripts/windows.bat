cd ..
git pull

IF EXIST "%PROGRAMFILES(X86)%" (GOTO 64BIT) ELSE (GOTO 32BIT)
:64BIT
go build -o webp-server-windows-amd64.exe webp-server.go
GOTO END

:32BIT
echo 32-bit...
go build -o webp-server-windows-i386.exe webp-server.go
GOTO END

pause
