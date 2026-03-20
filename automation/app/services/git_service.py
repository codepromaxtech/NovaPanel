import asyncio
from pathlib import Path


class GitOpsService:
    """Git clone, pull, and build operations."""

    async def _run(self, cmd: str, cwd: str = None) -> str:
        proc = await asyncio.create_subprocess_shell(
            cmd, cwd=cwd,
            stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def clone(self, repo_url: str, target_dir: str, branch: str = "main") -> dict:
        Path(target_dir).mkdir(parents=True, exist_ok=True)
        output = await self._run(f"git clone --branch {branch} --depth 1 {repo_url} {target_dir}")
        return {"status": "success", "message": f"Cloned {repo_url} → {target_dir}", "output": output}

    async def pull(self, target_dir: str, branch: str = "main") -> dict:
        output = await self._run(f"git pull origin {branch}", cwd=target_dir)
        commit = await self._run("git rev-parse --short HEAD", cwd=target_dir)
        return {"status": "success", "commit": commit.strip(), "output": output}

    async def build(self, target_dir: str, build_command: str = "npm run build") -> dict:
        output = await self._run(build_command, cwd=target_dir)
        return {"status": "success", "message": f"Build completed in {target_dir}", "output": output}
