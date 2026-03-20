package provisioner

// InstallScripts maps module IDs to their bash install scripts.
// These scripts are designed for Ubuntu/Debian and run as root.
var InstallScripts = map[string]string{

	// ──────────── Web Servers ────────────

	"web-nginx": `
echo "=== Installing Nginx ==="
apt-get update -qq
apt-get install -y nginx

# Enable and start
systemctl enable nginx
systemctl start nginx

# Create default config
cat > /etc/nginx/sites-available/default << 'NGINXCONF'
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    root /var/www/html;
    index index.html index.htm index.php;
    server_name _;

    location / {
        try_files $uri $uri/ =404;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php-fpm.sock;
    }

    location ~ /\.ht {
        deny all;
    }
}
NGINXCONF

# Install PHP-FPM
apt-get install -y php-fpm php-mysql php-curl php-gd php-mbstring php-xml php-zip

# Configure worker processes
CORES=$(nproc)
sed -i "s/worker_processes auto;/worker_processes $CORES;/" /etc/nginx/nginx.conf

# Security headers
cat > /etc/nginx/conf.d/security.conf << 'SECCONF'
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
server_tokens off;
SECCONF

# Test and reload
nginx -t && systemctl reload nginx
echo "Nginx installed successfully"
`,

	"web-apache": `
echo "=== Installing Apache ==="
apt-get update -qq
apt-get install -y apache2

# Enable essential modules
a2enmod rewrite ssl headers expires deflate proxy proxy_http

# Install PHP
apt-get install -y libapache2-mod-php php-mysql php-curl php-gd php-mbstring php-xml php-zip
a2enmod php*

# Security config
cat > /etc/apache2/conf-available/security-extra.conf << 'SECCONF'
ServerTokens Prod
ServerSignature Off
Header always set X-Frame-Options "SAMEORIGIN"
Header always set X-Content-Type-Options "nosniff"
Header always set X-XSS-Protection "1; mode=block"
TraceEnable Off
SECCONF
a2enconf security-extra

# Configure MPM
cat > /etc/apache2/mods-available/mpm_prefork.conf << 'MPMCONF'
<IfModule mpm_prefork_module>
    StartServers           5
    MinSpareServers        5
    MaxSpareServers       10
    MaxRequestWorkers    150
    MaxConnectionsPerChild 3000
</IfModule>
MPMCONF

# Enable and start
systemctl enable apache2
systemctl start apache2
apache2ctl configtest && systemctl reload apache2
echo "Apache installed successfully"
`,

	// ──────────── Databases ────────────

	"database-mysql": `
echo "=== Installing MariaDB ==="
apt-get update -qq
apt-get install -y mariadb-server mariadb-client

systemctl enable mariadb
systemctl start mariadb

# Secure installation
mysql -e "DELETE FROM mysql.user WHERE User='';"
mysql -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');"
mysql -e "DROP DATABASE IF EXISTS test;"
mysql -e "DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';"
mysql -e "FLUSH PRIVILEGES;"

# Enable remote access with bind-address
sed -i 's/bind-address.*=.*/bind-address = 0.0.0.0/' /etc/mysql/mariadb.conf.d/50-server.cnf 2>/dev/null || true

# Performance tuning
cat > /etc/mysql/mariadb.conf.d/99-novapanel.cnf << 'MYSQLCONF'
[mysqld]
innodb_buffer_pool_size = 256M
innodb_log_file_size = 64M
max_connections = 100
query_cache_size = 32M
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 2
MYSQLCONF

systemctl restart mariadb
echo "MariaDB installed successfully"
`,

	"database-postgres": `
echo "=== Installing PostgreSQL ==="
apt-get update -qq
apt-get install -y postgresql postgresql-contrib

systemctl enable postgresql
systemctl start postgresql

# Configure authentication
PG_HBA=$(find /etc/postgresql -name pg_hba.conf 2>/dev/null | head -1)
if [ -n "$PG_HBA" ]; then
    echo "host all all 0.0.0.0/0 md5" >> "$PG_HBA"
fi

# Listen on all interfaces
PG_CONF=$(find /etc/postgresql -name postgresql.conf 2>/dev/null | head -1)
if [ -n "$PG_CONF" ]; then
    sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" "$PG_CONF"
    # Performance tuning
    cat >> "$PG_CONF" << 'PGCONF'
shared_buffers = 256MB
effective_cache_size = 768MB
work_mem = 4MB
maintenance_work_mem = 64MB
max_connections = 100
PGCONF
fi

systemctl restart postgresql
echo "PostgreSQL installed successfully"
`,

	"database-mongo": `
echo "=== Installing MongoDB ==="
apt-get update -qq
apt-get install -y gnupg curl

# Import MongoDB GPG key and add repo
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | gpg --dearmor -o /usr/share/keyrings/mongodb-server-7.0.gpg 2>/dev/null || true
CODENAME=$(lsb_release -cs 2>/dev/null || echo "jammy")
echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu $CODENAME/mongodb-org/7.0 multiverse" > /etc/apt/sources.list.d/mongodb-org-7.0.list

apt-get update -qq
apt-get install -y mongodb-org || apt-get install -y mongodb

systemctl enable mongod 2>/dev/null || systemctl enable mongodb 2>/dev/null
systemctl start mongod 2>/dev/null || systemctl start mongodb 2>/dev/null
echo "MongoDB installed successfully"
`,

	"database-redis": `
echo "=== Installing Redis ==="
apt-get update -qq
apt-get install -y redis-server

# Configure for production
sed -i 's/^supervised no/supervised systemd/' /etc/redis/redis.conf 2>/dev/null || true
sed -i 's/^bind 127.0.0.1/bind 0.0.0.0/' /etc/redis/redis.conf 2>/dev/null || true

# Set max memory policy
cat >> /etc/redis/redis.conf << 'REDISCONF'
maxmemory 256mb
maxmemory-policy allkeys-lru
REDISCONF

systemctl enable redis-server
systemctl restart redis-server
echo "Redis installed successfully"
`,

	// ──────────── Containers ────────────

	"docker": `
echo "=== Installing Docker ==="
apt-get update -qq
apt-get install -y ca-certificates curl gnupg

# Add Docker GPG key
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null || true
chmod a+r /etc/apt/keyrings/docker.gpg

# Add repo
CODENAME=$(lsb_release -cs 2>/dev/null || echo "jammy")
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $CODENAME stable" > /etc/apt/sources.list.d/docker.list

apt-get update -qq
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

systemctl enable docker
systemctl start docker

# Install docker-compose standalone
curl -fsSL "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose 2>/dev/null || true
chmod +x /usr/local/bin/docker-compose 2>/dev/null || true

docker --version
echo "Docker installed successfully"
`,

	"kubernetes": `
echo "=== Installing Kubernetes ==="
apt-get update -qq
apt-get install -y apt-transport-https ca-certificates curl

# Disable swap (required for k8s)
swapoff -a
sed -i '/ swap / s/^/#/' /etc/fstab

# Load required modules
cat > /etc/modules-load.d/k8s.conf << 'EOF'
overlay
br_netfilter
EOF
modprobe overlay 2>/dev/null || true
modprobe br_netfilter 2>/dev/null || true

cat > /etc/sysctl.d/k8s.conf << 'EOF'
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOF
sysctl --system 2>/dev/null || true

# Add Kubernetes apt repo
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg 2>/dev/null || true
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /" > /etc/apt/sources.list.d/kubernetes.list

apt-get update -qq
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl

systemctl enable kubelet
echo "Kubernetes installed successfully"
`,

	// ──────────── System ────────────

	"monitoring": `
echo "=== Installing Monitoring Agent ==="
apt-get update -qq

# Install Node Exporter
NODE_EXPORTER_VERSION="1.7.0"
useradd --no-create-home --shell /bin/false node_exporter 2>/dev/null || true

cd /tmp
curl -fsSLO "https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
tar xzf "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
cp "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter" /usr/local/bin/
chown node_exporter:node_exporter /usr/local/bin/node_exporter

cat > /etc/systemd/system/node_exporter.service << 'EOF'
[Unit]
Description=Node Exporter
After=network.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter --collector.systemd --collector.processes

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable node_exporter
systemctl start node_exporter
rm -rf /tmp/node_exporter-*
echo "Monitoring agent installed successfully"
`,

	"firewall": `
echo "=== Configuring Firewall (UFW) ==="
apt-get update -qq
apt-get install -y ufw

# Default policies
ufw default deny incoming
ufw default allow outgoing

# Allow essential services
ufw allow ssh
ufw allow http
ufw allow https

# Enable (non-interactive)
echo "y" | ufw enable

ufw status verbose
echo "Firewall configured successfully"
`,

	// ──────────── Services ────────────

	"mail": `
echo "=== Installing Mail Server ==="
apt-get update -qq

# Pre-configure postfix
debconf-set-selections <<< "postfix postfix/mailname string $(hostname -f)"
debconf-set-selections <<< "postfix postfix/main_mailer_type string 'Internet Site'"

apt-get install -y postfix dovecot-core dovecot-imapd dovecot-pop3d dovecot-lmtpd

# Basic Postfix config
postconf -e "inet_interfaces = all"
postconf -e "inet_protocols = all"
postconf -e "mydestination = \$myhostname, localhost.\$mydomain, localhost"
postconf -e "smtpd_tls_cert_file = /etc/ssl/certs/ssl-cert-snakeoil.pem"
postconf -e "smtpd_tls_key_file = /etc/ssl/private/ssl-cert-snakeoil.key"
postconf -e "smtpd_tls_security_level = may"
postconf -e "smtp_tls_security_level = may"
postconf -e "smtpd_sasl_auth_enable = yes"
postconf -e "smtpd_sasl_type = dovecot"
postconf -e "smtpd_sasl_path = private/auth"

# Dovecot config
cat > /etc/dovecot/conf.d/10-auth.conf << 'EOF'
auth_mechanisms = plain login
!include auth-system.conf.ext
EOF

cat > /etc/dovecot/conf.d/10-mail.conf << 'EOF'
mail_location = maildir:~/Maildir
EOF

systemctl enable postfix dovecot
systemctl restart postfix dovecot
echo "Mail server installed successfully"
`,

	"dns": `
echo "=== Installing DNS Server ==="
apt-get update -qq
apt-get install -y bind9 bind9utils bind9-doc dnsutils

# Configure as caching/forwarding DNS
cat > /etc/bind/named.conf.options << 'DNSCONF'
options {
    directory "/var/cache/bind";
    recursion yes;
    allow-recursion { any; };
    listen-on { any; };
    listen-on-v6 { any; };
    forwarders {
        8.8.8.8;
        8.8.4.4;
        1.1.1.1;
    };
    dnssec-validation auto;
    allow-transfer { none; };
};
DNSCONF

named-checkconf
systemctl enable named
systemctl restart named
echo "DNS server installed successfully"
`,
}
