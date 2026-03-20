import asyncio
import secrets
import string


class DatabaseOpsService:
    """Manages MySQL and PostgreSQL database operations via CLI."""

    async def _run(self, cmd: str) -> str:
        proc = await asyncio.create_subprocess_shell(
            cmd, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def create_database(self, name: str, engine: str = "mysql", charset: str = "utf8mb4") -> dict:
        if engine in ("mysql", "mariadb"):
            await self._run(
                f"mysql -u root -e \"CREATE DATABASE IF NOT EXISTS \\`{name}\\` CHARACTER SET {charset};\""
            )
        elif engine == "postgresql":
            await self._run(
                f"sudo -u postgres psql -c \"CREATE DATABASE \\\"{name}\\\" ENCODING 'UTF8';\""
            )
        else:
            raise ValueError(f"Unsupported engine: {engine}")

        return {"status": "success", "message": f"Database '{name}' created ({engine})"}

    async def create_user(self, db_name: str, username: str, password: str = None) -> dict:
        if not password:
            password = ''.join(secrets.choice(string.ascii_letters + string.digits) for _ in range(16))

        await self._run(
            f"mysql -u root -e \"CREATE USER IF NOT EXISTS '{username}'@'localhost' IDENTIFIED BY '{password}';\""
        )
        await self._run(
            f"mysql -u root -e \"GRANT ALL PRIVILEGES ON \\`{db_name}\\`.* TO '{username}'@'localhost'; FLUSH PRIVILEGES;\""
        )
        return {"status": "success", "message": f"User '{username}' created for '{db_name}'"}

    async def drop_database(self, name: str, engine: str = "mysql") -> dict:
        if engine in ("mysql", "mariadb"):
            await self._run(f"mysql -u root -e \"DROP DATABASE IF EXISTS \\`{name}\\`;\"")
        elif engine == "postgresql":
            await self._run(f"sudo -u postgres psql -c \"DROP DATABASE IF EXISTS \\\"{name}\\\";\"")

        return {"status": "success", "message": f"Database '{name}' dropped"}
