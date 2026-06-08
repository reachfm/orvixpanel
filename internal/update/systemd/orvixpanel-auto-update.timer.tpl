[Unit]
Description=OrvixPanel Weekly Automatic Update Timer
After=network.target

[Timer]
OnCalendar=weekly
Persistent=true
# Randomize up to 1 hour to prevent stampede
RandomizedDelaySec=3600

[Install]
WantedBy=timers.target