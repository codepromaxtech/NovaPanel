import asyncio


class SecurityOpsService:
    """Manages UFW firewall rules and Fail2Ban jails."""

    async def _run(self, cmd: str) -> str:
        proc = await asyncio.create_subprocess_shell(
            cmd, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def add_ufw_rule(self, port: str, action: str = "allow", protocol: str = "tcp", source: str = "any") -> dict:
        if source == "any":
            cmd = f"ufw {action} {port}/{protocol}"
        else:
            cmd = f"ufw {action} from {source} to any port {port} proto {protocol}"
        output = await self._run(cmd)
        return {"status": "success", "message": f"UFW rule added: {action} {port}/{protocol}", "output": output}

    async def delete_ufw_rule(self, port: str, protocol: str = "tcp") -> dict:
        output = await self._run(f"ufw delete allow {port}/{protocol}")
        return {"status": "success", "message": f"UFW rule deleted: {port}/{protocol}", "output": output}

    async def fail2ban_status(self) -> dict:
        output = await self._run("fail2ban-client status")
        jails = []
        for line in output.split("\n"):
            if "Jail list:" in line:
                jail_names = line.split(":")[1].strip().split(",")
                jails = [j.strip() for j in jail_names if j.strip()]
        return {"status": "success", "jails": jails, "raw": output}

    async def fail2ban_control(self, jail: str, action: str) -> dict:
        if action not in ("start", "stop", "restart", "status"):
            raise ValueError(f"Invalid action: {action}")
        if action == "status":
            output = await self._run(f"fail2ban-client status {jail}")
        else:
            output = await self._run(f"fail2ban-client {action} {jail}")
        return {"status": "success", "jail": jail, "action": action, "output": output}
