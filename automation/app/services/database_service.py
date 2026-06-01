import asyncio
import re
import secrets
import string


_DB_NAME_RE = re.compile(r'^[a-zA-Z0-9_]{1,64}$')
_DB_USER_RE = re.compile(r'^[a-zA-Z0-9_]{1,32}$')
_CHARSET_RE = re.compile(r'^[a-zA-Z0-9_]+$')


class DatabaseOpsService:
    """Manages MySQL and PostgreSQL database operations via CLI."""

    async def _run_exec(self, args: list) -> str:
        proc = await asyncio.create_subprocess_exec(
            *args, stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def _run_mysql(self, sql: str) -> str:
        """Execute MySQL SQL via stdin to avoid shell injection."""
        proc = await asyncio.create_subprocess_exec(
            "mysql", "-u", "root",
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await proc.communicate(input=sql.encode())
        if proc.returncode != 0:
            raise Exception(f"Command failed: {stderr.decode()}")
        return stdout.decode()

    async def create_database(self, name: str, engine: str = "mysql", charset: str = "utf8mb4") -> dict:
        if not _DB_NAME_RE.match(name):
            raise ValueError(f"Invalid database name: {name}")
        if not _CHARSET_RE.match(charset):
            raise ValueError(f"Invalid charset: {charset}")

        if engine in ("mysql", "mariadb"):
            await self._run_mysql(f"CREATE DATABASE IF NOT EXISTS `{name}` CHARACTER SET {charset};\n")
        elif engine == "postgresql":
            await self._run_exec(
                ["sudo", "-u", "postgres", "psql", "-c", f'CREATE DATABASE "{name}" ENCODING \'UTF8\';']
            )
        else:
            raise ValueError(f"Unsupported engine: {engine}")

        return {"status": "success", "message": f"Database '{name}' created ({engine})"}

    async def create_user(self, db_name: str, username: str, password: str = None) -> dict:
        if not _DB_USER_RE.match(username):
            raise ValueError(f"Invalid username: {username}")
        if not _DB_NAME_RE.match(db_name):
            raise ValueError(f"Invalid database name: {db_name}")
        if not password:
            password = ''.join(secrets.choice(string.ascii_letters + string.digits) for _ in range(16))

        # Escape single quotes for MySQL string literals
        escaped_pw = password.replace("'", "''")
        await self._run_mysql(
            f"CREATE USER IF NOT EXISTS '{username}'@'localhost' IDENTIFIED BY '{escaped_pw}';\n"
            f"GRANT ALL PRIVILEGES ON `{db_name}`.* TO '{username}'@'localhost'; FLUSH PRIVILEGES;\n"
        )
        return {"status": "success", "message": f"User '{username}' created for '{db_name}'"}

    async def drop_database(self, name: str, engine: str = "mysql") -> dict:
        if not _DB_NAME_RE.match(name):
            raise ValueError(f"Invalid database name: {name}")

        if engine in ("mysql", "mariadb"):
            await self._run_mysql(f"DROP DATABASE IF EXISTS `{name}`;\n")
        elif engine == "postgresql":
            await self._run_exec(
                ["sudo", "-u", "postgres", "psql", "-c", f'DROP DATABASE IF EXISTS "{name}";']
            )

        return {"status": "success", "message": f"Database '{name}' dropped"}
