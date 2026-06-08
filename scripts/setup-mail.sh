#!/bin/bash
# OrvixPanel Mail System Setup Script
# Installs Postfix + Dovecot + OpenDKIM on a VPS

set -e

echo "=========================================="
echo "OrvixPanel Mail System Installer"
echo "=========================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root or with sudo"
    exit 1
fi

# Configuration
MAIL_DOMAIN="${1}"
POSTMASTER_EMAIL="${2:-postmaster@${MAIL_DOMAIN}}"
INSTALL_DIR="/opt/orvixpanel"
CONFIG_DIR="/etc/orvixpanel"

# Validate inputs
if [ -z "$MAIL_DOMAIN" ]; then
    echo "Usage: $0 <domain> [postmaster_email]"
    echo "Example: $0 example.com postmaster@example.com"
    exit 1
fi

echo "Installing mail system for domain: $MAIL_DOMAIN"
echo "Postmaster email: $POSTMASTER_EMAIL"

# Step 1: Update package list
echo ""
echo "=== Updating package list ==="
apt-get update -qq

# Step 2: Install mail packages
echo ""
echo "=== Installing mail packages ==="
DEBIAN_FRONTEND=noninteractive apt-get install -y \
    postfix \
    postfix-pcre \
    dovecot-core \
    dovecot-imapd \
    dovecot-lmtpd \
    dovecot-sqlite \
    opendkim \
    opendkim-tools \
    bsd-mailx

# Step 3: Create directories
echo ""
echo "=== Creating directories ==="
mkdir -p "$CONFIG_DIR/mail"
mkdir -p "/var/spool/postfix/maildir"
mkdir -p "/var/spool/postfix/maildrop"
mkdir -p "/etc/dovecot/private"
mkdir -p "/etc/dovecot/sieve"
mkdir -p "/var/db/opendkim"
chown -R postfix:postfix /var/spool/postfix
chown -R dovecot:dovecot /var/spool/postfix/maildir

# Step 4: Generate Dovecot SSL certificates
echo ""
echo "=== Generating SSL certificates ==="
if [ ! -f "/etc/dovecot/private/dovecot.pem" ]; then
    openssl req -new -x509 -nodes -days 3650 \
        -out /etc/dovecot/dovecot.pem \
        -keyout /etc/dovecot/private/dovecot.pem \
        -subj "/CN=${MAIL_DOMAIN}/O=OrvixPanel"
    chmod 640 /etc/dovecot/private/dovecot.pem
fi

# Step 5: Create OpenDKIM keys
echo ""
echo "=== Creating OpenDKIM keys ==="
if [ ! -f "/var/db/opendkim/default.private" ]; then
    opendkim-genkey -D /var/db/opendkim -d "$MAIL_DOMAIN" -s default
    mv /var/db/opendkim/default.private /var/db/opendkim/default
    chown opendkim:opendkim /var/db/opendkim/default*
fi

# Step 6: Generate configuration files from OrvixPanel
echo ""
echo "=== Generating configuration files ==="
if [ -f "$INSTALL_DIR/orvixpanel" ]; then
    # Generate Postfix config
    $INSTALL_DIR/orvixpanel mail config postfix > "$CONFIG_DIR/mail/main.cf"

    # Generate Dovecot config
    $INSTALL_DIR/orvixpanel mail config dovecot > "$CONFIG_DIR/mail/dovecot.conf"

    # Generate OpenDKIM config
    $INSTALL_DIR/orvixpanel mail config opendkim > "$CONFIG_DIR/mail/opendkim.conf"

    echo "Configuration files generated in $CONFIG_DIR/mail/"
fi

# Step 7: Configure Postfix
echo ""
echo "=== Configuring Postfix ==="
postconf -e "myhostname = $MAIL_DOMAIN"
postconf -e "mydomain = $MAIL_DOMAIN"
postconf -e "myorigin = \$mydomain"
postconf -e "mydestination = \$myhostname, localhost.\$mydomain, localhost, \$mydomain"
postconf -e "home_mailbox = Maildir/"
postconf -e "mail_spool_directory = /var/spool/postfix/maildir"

# Step 8: Configure Dovecot
echo ""
echo "=== Configuring Dovecot ==="
if [ ! -f /etc/dovecot/conf.d/10-orvixpanel.conf ]; then
    cat > /etc/dovecot/conf.d/10-orvixpanel.conf << 'DOVECOT'
# OrvixPanel Mail Configuration
protocols = imap lmtp
mail_location = maildir:/var/spool/postfix/maildir/%u
namespace inbox {
    type = private
    separator = /
    prefix =
    inbox = yes
}
DOVECOT
fi

# Step 9: Configure OpenDKIM
echo ""
echo "=== Configuring OpenDKIM ==="
cat > /etc/opendkim.conf << 'OPENDKIM'
# OpenDKIM Configuration
KeyTable           refile:/var/db/opendkim/key.table
SigningTable       refile:/var/db/opendkim/signing.table
ExternalIgnoreList refile:/var/db/opendkim/trustedHosts
InternalHosts      refile:/var/db/opendkim/trustedHosts
Socket             inet:8891@localhost
UserID             opendkim:opendkim
Canonicalization   simple relaxed
Mode               sv
SyslogSuccess      Yes
LogWhy             Yes
OPENDKIM

# Create key and signing tables
cat > /var/db/opendkim/key.table << KEYTABLE
default._domainkey.$MAIL_DOMAIN $MAIL_DOMAIN:default:/var/db/opendkim/default
KEYTABLE

cat > /var/db/opendkim/signing.table << SIGNING
*@$MAIL_DOMAIN default._domainkey.$MAIL_DOMAIN
SIGNING

cat > /var/db/opendkim/trustedHosts << HOSTS
localhost
127.0.0.1
$MAIL_DOMAIN
*
HOSTS

chown -R opendkim:opendkim /var/db/opendkim

# Step 10: Update Postfix for OpenDKIM
echo ""
echo "=== Integrating OpenDKIM with Postfix ==="
postconf -e "milter_default_action = accept"
postconf -e "mil_macro_maps = hash:/etc/postfix/macro_maps"
postconf -e "smtpd_milters = inet:localhost:8891"
postconf -e "non_smtpd_milters = inet:localhost:8891"

# Step 11: Start services
echo ""
echo "=== Starting services ==="
systemctl enable postfix
systemctl restart postfix
systemctl enable dovecot
systemctl restart dovecot
systemctl enable opendkim
systemctl restart opendkim

# Step 12: Get DKIM DNS record
echo ""
echo "=== DKIM DNS Record ==="
echo "Add the following TXT record to your DNS:"
echo ""
cat /var/db/opendkim/default.txt
echo ""

# Step 13: Create test mailbox directory
echo ""
echo "=== Creating test mailboxes ==="
for user in postmaster info support; do
    mkdir -p "/var/spool/postfix/maildir/$user@$MAIL_DOMAIN"
    chown -R postfix:postfix "/var/spool/postfix/maildir/$user@$MAIL_DOMAIN"
done

echo ""
echo "=========================================="
echo "MAIL SYSTEM INSTALLATION COMPLETE"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Add the DKIM TXT record to your DNS"
echo "2. Add SPF record: v=spf1 +mx -all"
echo "3. Add DMARC record: v=DMARC1; p=none; rua=mailto:dmarc@$MAIL_DOMAIN"
echo "4. Test SMTP: telnet localhost 25"
echo "5. Test IMAP: telnet localhost 143"
echo ""
echo "Configuration files: $CONFIG_DIR/mail/"
echo "Logs: /var/log/mail.log"
echo "=========================================="