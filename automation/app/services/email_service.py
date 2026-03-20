import asyncio
import os
from pathlib import Path


class EmailOpsService:
    """Manages Postfix virtual mailbox and Dovecot user database."""

    VMAIL_DIR = "/var/mail/vhosts"
    VIRTUAL_MAILBOX_MAP = "/etc/postfix/virtual_mailbox_users"
    VIRTUAL_ALIAS_MAP = "/etc/postfix/virtual_aliases"

    async def _run(self, cmd: str) -> str:
        proc = await asyncio.create_subprocess_shell(
            cmd, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def create_mailbox(self, address: str, password: str, quota_mb: int = 1024) -> dict:
        user, domain = address.split("@")

        # Create maildir
        maildir = Path(self.VMAIL_DIR) / domain / user
        maildir.mkdir(parents=True, exist_ok=True)

        # Generate Dovecot password hash
        result = await self._run(f"doveadm pw -s SHA512-CRYPT -p '{password}'")
        password_hash = result.strip()

        # Add to Postfix virtual mailbox map
        map_file = Path(self.VIRTUAL_MAILBOX_MAP)
        map_file.parent.mkdir(parents=True, exist_ok=True)
        with open(map_file, "a") as f:
            f.write(f"{address}    {domain}/{user}/\n")

        # Rebuild postmap
        await self._run(f"postmap {self.VIRTUAL_MAILBOX_MAP}")

        return {
            "status": "success",
            "message": f"Mailbox '{address}' created",
            "maildir": str(maildir),
            "quota_mb": quota_mb,
        }

    async def delete_mailbox(self, address: str) -> dict:
        # Remove from virtual mailbox map
        map_file = Path(self.VIRTUAL_MAILBOX_MAP)
        if map_file.exists():
            lines = map_file.read_text().splitlines()
            lines = [l for l in lines if not l.startswith(address)]
            map_file.write_text("\n".join(lines) + "\n")
            await self._run(f"postmap {self.VIRTUAL_MAILBOX_MAP}")

        return {"status": "success", "message": f"Mailbox '{address}' deleted"}

    async def create_forwarder(self, source: str, destination: str) -> dict:
        alias_file = Path(self.VIRTUAL_ALIAS_MAP)
        alias_file.parent.mkdir(parents=True, exist_ok=True)
        with open(alias_file, "a") as f:
            f.write(f"{source}    {destination}\n")

        await self._run(f"postmap {self.VIRTUAL_ALIAS_MAP}")
        return {"status": "success", "message": f"Forwarder {source} → {destination} created"}
