"""Nginx virtual host management service."""

import os
import asyncio
from pathlib import Path
from jinja2 import Template

from app.schemas.schemas import NginxVhostCreate, NginxVhostResponse

# Nginx configuration paths
NGINX_SITES_AVAILABLE = os.getenv("NGINX_SITES_AVAILABLE", "/etc/nginx/sites-available")
NGINX_SITES_ENABLED = os.getenv("NGINX_SITES_ENABLED", "/etc/nginx/sites-enabled")

# Default vhost template
VHOST_TEMPLATE = """\
{% if is_load_balancer and target_ips %}
upstream backend_pool_{{ domain | replace('.', '_') }} {
    least_conn;
{% for ip in target_ips %}
    server {{ ip }};
{% endfor %}
}
{% endif %}

server {
    listen 80;
    listen [::]:80;
    server_name {{ domain }} www.{{ domain }};

    root {{ document_root }}/{{ domain }}/public;
    index index.php index.html index.htm;

    access_log /var/log/nginx/{{ domain }}.access.log;
    error_log /var/log/nginx/{{ domain }}.error.log;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    {% if is_load_balancer and target_ips %}
    location / {
        proxy_pass http://backend_pool_{{ domain | replace('.', '_') }};
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
    {% elif proxy_pass %}
    location / {
        proxy_pass {{ proxy_pass }};
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
    {% else %}
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    {% endif %}

    {% if php_version and not is_load_balancer and not proxy_pass %}

    # Deny access to hidden files
    location ~ /\\. {
        deny all;
    }

    # Cache static assets
    location ~* \\.(jpg|jpeg|png|gif|ico|css|js|woff2|woff|ttf|svg)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}

{% if ssl_enabled %}
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name {{ domain }} www.{{ domain }};

    ssl_certificate /etc/letsencrypt/live/{{ domain }}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/{{ domain }}/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    root {{ document_root }}/{{ domain }}/public;
    index index.php index.html index.htm;

    access_log /var/log/nginx/{{ domain }}.ssl.access.log;
    error_log /var/log/nginx/{{ domain }}.ssl.error.log;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;

    {% if is_load_balancer and target_ips %}
    location / {
        proxy_pass http://backend_pool_{{ domain | replace('.', '_') }};
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
    {% elif proxy_pass %}
    location / {
        proxy_pass {{ proxy_pass }};
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
    {% else %}
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
    {% endif %}

    {% if php_version and not is_load_balancer and not proxy_pass %}

    location ~ /\\. {
        deny all;
    }

    location ~* \\.(jpg|jpeg|png|gif|ico|css|js|woff2|woff|ttf|svg)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
{% endif %}
"""


class NginxService:
    """Manages Nginx virtual host configurations."""

    def __init__(self):
        self.template = Template(VHOST_TEMPLATE)

    async def create_vhost(self, req: NginxVhostCreate) -> NginxVhostResponse:
        """Create a new Nginx vhost configuration file."""
        config_content = self.template.render(
            domain=req.domain,
            document_root=req.document_root,
            php_version=req.php_version,
            ssl_enabled=req.ssl_enabled,
            proxy_pass=req.proxy_pass,
            is_load_balancer=req.is_load_balancer,
            target_ips=req.target_ips,
        )

        config_path = os.path.join(NGINX_SITES_AVAILABLE, req.domain)

        # Create document root
        doc_root = os.path.join(req.document_root, req.domain, "public")
        os.makedirs(doc_root, exist_ok=True)

        # Write a default index page
        index_path = os.path.join(doc_root, "index.html")
        if not os.path.exists(index_path):
            with open(index_path, "w") as f:
                f.write(f"<html><body><h1>Welcome to {req.domain}</h1><p>Powered by NovaPanel</p></body></html>\n")

        # Write config file
        with open(config_path, "w") as f:
            f.write(config_content)

        # Enable site (symlink)
        enabled_path = os.path.join(NGINX_SITES_ENABLED, req.domain)
        if not os.path.exists(enabled_path):
            os.symlink(config_path, enabled_path)

        # Test and reload
        await self.test_config()
        await self.reload()

        return NginxVhostResponse(
            domain=req.domain,
            config_path=config_path,
            status="active",
            message=f"Vhost for {req.domain} created and enabled",
        )

    async def delete_vhost(self, domain: str) -> NginxVhostResponse:
        """Remove an Nginx vhost configuration."""
        config_path = os.path.join(NGINX_SITES_AVAILABLE, domain)
        enabled_path = os.path.join(NGINX_SITES_ENABLED, domain)

        if not os.path.exists(config_path):
            raise FileNotFoundError(f"Vhost config not found: {config_path}")

        if os.path.exists(enabled_path):
            os.unlink(enabled_path)

        os.unlink(config_path)
        await self.reload()

        return NginxVhostResponse(
            domain=domain,
            config_path=config_path,
            status="deleted",
            message=f"Vhost for {domain} deleted",
        )

    async def get_vhost_config(self, domain: str) -> str:
        """Read the vhost configuration for a domain."""
        config_path = os.path.join(NGINX_SITES_AVAILABLE, domain)
        if not os.path.exists(config_path):
            raise FileNotFoundError(f"Vhost config not found: {config_path}")

        with open(config_path, "r") as f:
            return f.read()

    async def test_config(self) -> str:
        """Test Nginx configuration syntax."""
        proc = await asyncio.create_subprocess_exec(
            "nginx", "-t",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        _, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Nginx config test failed: {stderr.decode()}")
        return "Configuration test passed"

    async def reload(self) -> str:
        """Reload Nginx configuration."""
        proc = await asyncio.create_subprocess_exec(
            "nginx", "-s", "reload",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        _, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Nginx reload failed: {stderr.decode()}")
        return "Nginx reloaded successfully"
