import asyncio
import shlex
from pathlib import Path


class GitOpsService:
    """Git clone, pull, and build operations."""

    async def _run_exec(self, args: list, cwd: str = None) -> str:
        proc = await asyncio.create_subprocess_exec(
            *args, cwd=cwd,
            stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def clone(self, repo_url: str, target_dir: str, branch: str = "main") -> dict:
        Path(target_dir).mkdir(parents=True, exist_ok=True)
        output = await self._run_exec(
            ["git", "clone", "--branch", branch, "--depth", "1", repo_url, target_dir]
        )
        return {"status": "success", "message": f"Cloned {repo_url} → {target_dir}", "output": output}

    async def pull(self, target_dir: str, branch: str = "main") -> dict:
        output = await self._run_exec(["git", "pull", "origin", branch], cwd=target_dir)
        commit = await self._run_exec(["git", "rev-parse", "--short", "HEAD"], cwd=target_dir)
        return {"status": "success", "commit": commit.strip(), "output": output}

    async def build(self, target_dir: str, build_command: str = "npm run build") -> dict:
        # shlex.split prevents shell metacharacter injection while supporting
        # commands with arguments (e.g. "npm run build", "composer install")
        args = shlex.split(build_command)
        output = await self._run_exec(args, cwd=target_dir)
        return {"status": "success", "message": f"Build completed in {target_dir}", "output": output}
