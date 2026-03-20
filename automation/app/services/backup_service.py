import asyncio
import os
import glob
from datetime import datetime
from pathlib import Path


class BackupOpsService:
    """Manages backup creation (tar/rsync) and S3 uploads."""

    BACKUP_DIR = "/var/backups/novapanel"

    async def _run(self, cmd: str) -> str:
        proc = await asyncio.create_subprocess_shell(
            cmd, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Backup command failed: {stderr.decode()}")
        return stdout.decode()

    async def create_backup(
        self, source_path: str, backup_type: str = "full",
        storage: str = "local", s3_destination: str = None
    ) -> dict:
        Path(self.BACKUP_DIR).mkdir(parents=True, exist_ok=True)
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        archive_name = f"backup_{timestamp}.tar.gz"
        archive_path = os.path.join(self.BACKUP_DIR, archive_name)

        if backup_type == "full":
            await self._run(f"tar -czf {archive_path} -C / {source_path.lstrip('/')}")
        elif backup_type == "incremental":
            snapshot_file = os.path.join(self.BACKUP_DIR, "snapshot.snar")
            await self._run(
                f"tar -czf {archive_path} --listed-incremental={snapshot_file} -C / {source_path.lstrip('/')}"
            )

        size_mb = os.path.getsize(archive_path) / (1024 * 1024) if os.path.exists(archive_path) else 0

        result = {
            "status": "success",
            "archive": archive_path,
            "size_mb": round(size_mb, 2),
            "type": backup_type,
            "storage": storage,
        }

        if storage == "s3" and s3_destination:
            await self._run(f"aws s3 cp {archive_path} {s3_destination}/{archive_name}")
            result["s3_path"] = f"{s3_destination}/{archive_name}"

        return result

    async def restore(self, archive_path: str, restore_to: str) -> dict:
        Path(restore_to).mkdir(parents=True, exist_ok=True)
        await self._run(f"tar -xzf {archive_path} -C {restore_to}")
        return {"status": "success", "message": f"Restored {archive_path} to {restore_to}"}

    async def list_backups(self, storage: str = "local") -> dict:
        archives = sorted(glob.glob(os.path.join(self.BACKUP_DIR, "*.tar.gz")), reverse=True)
        backups = []
        for a in archives[:50]:
            stat = os.stat(a)
            backups.append({
                "path": a,
                "size_mb": round(stat.st_size / (1024 * 1024), 2),
                "created_at": datetime.fromtimestamp(stat.st_ctime).isoformat(),
            })
        return {"backups": backups, "total": len(backups)}
