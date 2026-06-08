[Unit]
Description=OrvixPanel Daily Update Check Timer
After=network.target

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target