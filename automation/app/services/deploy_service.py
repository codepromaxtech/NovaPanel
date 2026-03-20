"""Application deployment service."""

import asyncio
import os
from datetime import datetime

from app.schemas.schemas import DeployRequest, DeployResponse

DEPLOY_BASE_DIR = os.getenv("DEPLOY_BASE_DIR", "/var/www")


class DeployService:
    """Handles application deployment via git, upload, or container."""

    async def deploy(self, req: DeployRequest) -> DeployResponse:
        """Deploy an application based on the deploy method."""
        deploy_log = []
        deploy_dir = os.path.join(DEPLOY_BASE_DIR, req.domain)

        try:
            if req.deploy_method.value == "git":
                result = await self._deploy_git(req, deploy_dir, deploy_log)
            elif req.deploy_method.value == "container":
                result = await self._deploy_container(req, deploy_log)
            else:
                result = await self._deploy_upload(req, deploy_dir, deploy_log)

            return DeployResponse(
                app_name=req.app_name,
                domain=req.domain,
                status="success",
                deploy_log="\n".join(deploy_log),
                message=f"Application {req.app_name} deployed successfully",
            )
        except Exception as e:
            deploy_log.append(f"ERROR: {str(e)}")
            return DeployResponse(
                app_name=req.app_name,
                domain=req.domain,
                status="failed",
                deploy_log="\n".join(deploy_log),
                message=f"Deployment failed: {str(e)}",
            )

    async def _deploy_git(self, req: DeployRequest, deploy_dir: str, log: list):
        """Deploy from a git repository."""
        if not req.git_repo:
            raise ValueError("git_repo is required for git deployments")

        log.append(f"[{datetime.now().isoformat()}] Starting git deployment")
        log.append(f"  Repository: {req.git_repo}")
        log.append(f"  Branch: {req.git_branch}")

        if os.path.exists(os.path.join(deploy_dir, ".git")):
            # Pull latest changes
            log.append("  Pulling latest changes...")
            proc = await asyncio.create_subprocess_exec(
                "git", "pull", "origin", req.git_branch,
                cwd=deploy_dir,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, stderr = await proc.communicate()
            log.append(f"  {stdout.decode().strip()}")
        else:
            # Clone repository
            log.append("  Cloning repository...")
            os.makedirs(deploy_dir, exist_ok=True)
            proc = await asyncio.create_subprocess_exec(
                "git", "clone", "-b", req.git_branch, req.git_repo, deploy_dir,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, stderr = await proc.communicate()
            if proc.returncode != 0:
                raise Exception(f"Git clone failed: {stderr.decode()}")
            log.append("  Clone successful")

        # Write env vars if provided
        if req.env_vars:
            env_file = os.path.join(deploy_dir, ".env")
            with open(env_file, "w") as f:
                for key, value in req.env_vars.items():
                    f.write(f"{key}={value}\n")
            log.append("  Environment variables written to .env")

        # Auto-detect and run install commands
        if os.path.exists(os.path.join(deploy_dir, "package.json")):
            log.append("  Detected Node.js project — running npm install...")
            await self._run_cmd(["npm", "install", "--production"], deploy_dir, log)
        elif os.path.exists(os.path.join(deploy_dir, "requirements.txt")):
            log.append("  Detected Python project — installing dependencies...")
            await self._run_cmd(["pip", "install", "-r", "requirements.txt"], deploy_dir, log)
        elif os.path.exists(os.path.join(deploy_dir, "composer.json")):
            log.append("  Detected PHP project — running composer install...")
            await self._run_cmd(["composer", "install", "--no-dev"], deploy_dir, log)

        log.append(f"[{datetime.now().isoformat()}] Git deployment complete")

    async def _deploy_container(self, req: DeployRequest, log: list):
        """Deploy using Docker container."""
        log.append(f"[{datetime.now().isoformat()}] Starting container deployment")
        container_name = f"novapanel-{req.app_name}"

        # Stop existing container
        log.append(f"  Stopping existing container {container_name}...")
        await self._run_cmd(["docker", "stop", container_name], "/tmp", log, ignore_errors=True)
        await self._run_cmd(["docker", "rm", container_name], "/tmp", log, ignore_errors=True)

        # Build env args
        env_args = []
        for key, value in req.env_vars.items():
            env_args.extend(["-e", f"{key}={value}"])

        # Run container
        cmd = [
            "docker", "run", "-d",
            "--name", container_name,
            "--restart", "unless-stopped",
            *env_args,
            req.git_repo or req.app_name,  # image name
        ]
        log.append(f"  Starting container...")
        await self._run_cmd(cmd, "/tmp", log)

        log.append(f"[{datetime.now().isoformat()}] Container deployment complete")

    async def _deploy_upload(self, req: DeployRequest, deploy_dir: str, log: list):
        """Handle file upload deployment (prepare directory)."""
        log.append(f"[{datetime.now().isoformat()}] Preparing upload deployment directory")
        os.makedirs(os.path.join(deploy_dir, "public"), exist_ok=True)
        log.append(f"  Directory ready at: {deploy_dir}")
        log.append(f"  Upload files to {deploy_dir}/public/")
        log.append(f"[{datetime.now().isoformat()}] Upload deployment directory prepared")

    async def _run_cmd(self, cmd: list, cwd: str, log: list, ignore_errors: bool = False):
        """Run a command and capture output."""
        proc = await asyncio.create_subprocess_exec(
            *cmd,
            cwd=cwd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()

        if stdout.decode().strip():
            log.append(f"    {stdout.decode().strip()}")

        if proc.returncode != 0 and not ignore_errors:
            error_msg = stderr.decode().strip()
            log.append(f"    ERROR: {error_msg}")
            raise Exception(error_msg)

    async def get_status(self, app_name: str) -> dict:
        """Get deployment status for an application."""
        deploy_dir = os.path.join(DEPLOY_BASE_DIR, app_name)
        if not os.path.exists(deploy_dir):
            raise Exception(f"Application {app_name} not found")

        return {
            "app_name": app_name,
            "deploy_dir": deploy_dir,
            "status": "active",
            "last_modified": datetime.fromtimestamp(
                os.path.getmtime(deploy_dir)
            ).isoformat(),
        }

    async def rollback(self, app_name: str) -> dict:
        """Rollback to previous deployment (git reset)."""
        deploy_dir = os.path.join(DEPLOY_BASE_DIR, app_name)
        if not os.path.exists(os.path.join(deploy_dir, ".git")):
            raise Exception("Rollback only supported for git deployments")

        proc = await asyncio.create_subprocess_exec(
            "git", "reset", "--hard", "HEAD~1",
            cwd=deploy_dir,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate()

        return {
            "app_name": app_name,
            "status": "rolled_back",
            "message": f"Rolled back to previous version: {stdout.decode().strip()}",
        }
