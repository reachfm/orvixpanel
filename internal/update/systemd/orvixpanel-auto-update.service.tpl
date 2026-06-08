[Unit]
Description=OrvixPanel Automatic Update Service
After=network.target

[Service]
Type=oneshot
ExecStart={{.BinaryPath}} update --channel stable --skip-backup
User=root
StandardOutput=journal
StandardError=journal
# Auto-rollback on failure (handled by binary)
Environment=ORVIX_AUTO_UPDATE=1

[Install]
WantedBy=multi-user.target