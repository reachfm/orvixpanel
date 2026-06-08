[Unit]
Description=OrvixPanel Update Check Service
After=network.target

[Service]
Type=oneshot
ExecStart={{.BinaryPath}} update --check
User=root
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target