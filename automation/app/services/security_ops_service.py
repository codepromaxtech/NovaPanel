import asyncio
import re


_PORT_RE = re.compile(r'^\d{1,5}$')
_IP_CIDR_RE = re.compile(
    r'^((\d{1,3}\.){3}\d{1,3}(/\d{1,2})?)$|^any$'
)
_JAIL_RE = re.compile(r'^[a-zA-Z0-9][a-zA-Z0-9\-]{0,63}$')
_PROTOCOL_RE = re.compile(r'^(tcp|udp)$')
_ACTION_RE = re.compile(r'^(allow|deny|reject)$')


def _validate_port(port: str) -> None:
    if not _PORT_RE.match(port) or not (1 <= int(port) <= 65535):
        raise ValueError(f"Invalid port: {port}")


def _validate_source(source: str) -> None:
    if not _IP_CIDR_RE.match(source):
        raise ValueError(f"Invalid source address: {source}")


def _validate_jail(jail: str) -> None:
    if not _JAIL_RE.match(jail):
        raise ValueError(f"Invalid jail name: {jail}")


class SecurityOpsService:
    """Manages UFW firewall rules and Fail2Ban jails."""

    async def _run(self, args: list) -> str:
        proc = await asyncio.create_subprocess_exec(
            *args, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def add_ufw_rule(self, port: str, action: str = "allow", protocol: str = "tcp", source: str = "any") -> dict:
        _validate_port(port)
        _validate_source(source)
        if not _ACTION_RE.match(action):
            raise ValueError(f"Invalid action: {action}")
        if not _PROTOCOL_RE.match(protocol):
            raise ValueError(f"Invalid protocol: {protocol}")

        if source == "any":
            output = await self._run(["ufw", action, f"{port}/{protocol}"])
        else:
            output = await self._run(
                ["ufw", action, "from", source, "to", "any", "port", port, "proto", protocol]
            )
        return {"status": "success", "message": f"UFW rule added: {action} {port}/{protocol}", "output": output}

    async def delete_ufw_rule(self, port: str, protocol: str = "tcp") -> dict:
        _validate_port(port)
        if not _PROTOCOL_RE.match(protocol):
            raise ValueError(f"Invalid protocol: {protocol}")
        output = await self._run(["ufw", "delete", "allow", f"{port}/{protocol}"])
        return {"status": "success", "message": f"UFW rule deleted: {port}/{protocol}", "output": output}

    async def fail2ban_status(self) -> dict:
        output = await self._run(["fail2ban-client", "status"])
        jails = []
        for line in output.split("\n"):
            if "Jail list:" in line:
                jail_names = line.split(":")[1].strip().split(",")
                jails = [j.strip() for j in jail_names if j.strip()]
        return {"status": "success", "jails": jails, "raw": output}

    async def fail2ban_control(self, jail: str, action: str) -> dict:
        if action not in ("start", "stop", "restart", "status"):
            raise ValueError(f"Invalid action: {action}")
        _validate_jail(jail)
        if action == "status":
            output = await self._run(["fail2ban-client", "status", jail])
        else:
            output = await self._run(["fail2ban-client", action, jail])
        return {"status": "success", "jail": jail, "action": action, "output": output}
