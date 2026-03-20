"""SSL certificate management service using Let's Encrypt."""

import asyncio
import os
from datetime import datetime
from pathlib import Path

from app.schemas.schemas import SSLRequest, SSLResponse

LETSENCRYPT_LIVE = "/etc/letsencrypt/live"


class SSLService:
    """Manages SSL certificates via Let's Encrypt / certbot."""

    async def issue_certificate(self, req: SSLRequest) -> SSLResponse:
        """Issue a new SSL certificate using certbot."""
        cmd = [
            "certbot", "certonly",
            "--non-interactive",
            "--agree-tos",
            "--email", req.email,
            "--webroot",
            "--webroot-path", req.webroot,
            "-d", req.domain,
            "-d", f"www.{req.domain}",
        ]

        proc = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()

        if proc.returncode != 0:
            return SSLResponse(
                domain=req.domain,
                status="failed",
                message=f"Certificate issuance failed: {stderr.decode()}",
            )

        cert_path = os.path.join(LETSENCRYPT_LIVE, req.domain, "fullchain.pem")
        key_path = os.path.join(LETSENCRYPT_LIVE, req.domain, "privkey.pem")

        return SSLResponse(
            domain=req.domain,
            status="issued",
            certificate_path=cert_path,
            private_key_path=key_path,
            message=f"SSL certificate issued for {req.domain}",
        )

    async def renew_certificate(self, domain: str) -> SSLResponse:
        """Renew an existing SSL certificate."""
        proc = await asyncio.create_subprocess_exec(
            "certbot", "renew",
            "--cert-name", domain,
            "--non-interactive",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()

        if proc.returncode != 0:
            return SSLResponse(
                domain=domain,
                status="renewal_failed",
                message=f"Renewal failed: {stderr.decode()}",
            )

        return SSLResponse(
            domain=domain,
            status="renewed",
            message=f"SSL certificate renewed for {domain}",
        )

    async def get_certificate_status(self, domain: str) -> SSLResponse:
        """Check certificate status and expiration."""
        cert_dir = os.path.join(LETSENCRYPT_LIVE, domain)
        if not os.path.exists(cert_dir):
            raise FileNotFoundError(f"No certificate found for {domain}")

        # Use openssl to check expiry
        cert_path = os.path.join(cert_dir, "fullchain.pem")
        proc = await asyncio.create_subprocess_exec(
            "openssl", "x509", "-enddate", "-noout", "-in", cert_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await proc.communicate()

        expires_str = stdout.decode().strip().replace("notAfter=", "")

        return SSLResponse(
            domain=domain,
            status="active",
            certificate_path=cert_path,
            private_key_path=os.path.join(cert_dir, "privkey.pem"),
            expires_at=expires_str,
            message=f"Certificate active, expires: {expires_str}",
        )

    async def revoke_certificate(self, domain: str) -> str:
        """Revoke an SSL certificate."""
        proc = await asyncio.create_subprocess_exec(
            "certbot", "revoke",
            "--cert-name", domain,
            "--non-interactive",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        _, stderr = await proc.communicate()

        if proc.returncode != 0:
            raise Exception(f"Revocation failed: {stderr.decode()}")

        return f"Certificate for {domain} revoked successfully"
