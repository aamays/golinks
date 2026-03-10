#!/bin/bash
set -euo pipefail

echo "Building golinks..."
go build -o golinks .

echo "Installing binary to /usr/local/bin/golinks..."
sudo cp golinks /usr/local/bin/golinks

echo "Setting up /etc/hosts entry..."
if ! grep -q '127.0.0.1.*\bgo\b' /etc/hosts; then
    echo '127.0.0.1 go' | sudo tee -a /etc/hosts > /dev/null
    echo "Added '127.0.0.1 go' to /etc/hosts"
else
    echo "/etc/hosts already has 'go' entry"
fi

echo "Installing LaunchDaemon..."
sudo tee /Library/LaunchDaemons/com.golinks.server.plist > /dev/null <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.golinks.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/golinks</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/golinks.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/golinks.err</string>
</dict>
</plist>
PLIST

echo "Loading LaunchDaemon..."
sudo launchctl bootout system/com.golinks.server 2>/dev/null || true
sudo launchctl bootstrap system /Library/LaunchDaemons/com.golinks.server.plist

echo ""
echo "Done! golinks is running. Try: open http://go/"
