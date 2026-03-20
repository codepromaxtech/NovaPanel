import asyncio
import psutil
from datetime import datetime


class MonitoringOpsService:
    """Deep system monitoring via psutil and systemctl."""

    async def get_metrics(self) -> dict:
        cpu = psutil.cpu_percent(interval=0.5, percpu=True)
        mem = psutil.virtual_memory()
        disk = psutil.disk_usage("/")
        net = psutil.net_io_counters()
        load = psutil.getloadavg()

        return {
            "cpu": {
                "percent_per_core": cpu,
                "percent_total": sum(cpu) / len(cpu) if cpu else 0,
                "cores": psutil.cpu_count(),
            },
            "memory": {
                "total_mb": mem.total // (1024 * 1024),
                "used_mb": mem.used // (1024 * 1024),
                "available_mb": mem.available // (1024 * 1024),
                "percent": mem.percent,
            },
            "disk": {
                "total_gb": round(disk.total / (1024**3), 2),
                "used_gb": round(disk.used / (1024**3), 2),
                "free_gb": round(disk.free / (1024**3), 2),
                "percent": disk.percent,
            },
            "network": {
                "bytes_sent": net.bytes_sent,
                "bytes_recv": net.bytes_recv,
                "packets_sent": net.packets_sent,
                "packets_recv": net.packets_recv,
            },
            "load_average": {"1m": load[0], "5m": load[1], "15m": load[2]},
            "timestamp": datetime.now().isoformat(),
        }

    async def get_top_processes(self, limit: int = 20) -> dict:
        processes = []
        for proc in psutil.process_iter(["pid", "name", "cpu_percent", "memory_percent", "status"]):
            try:
                info = proc.info
                processes.append({
                    "pid": info["pid"],
                    "name": info["name"],
                    "cpu_percent": info["cpu_percent"] or 0,
                    "memory_percent": round(info["memory_percent"] or 0, 2),
                    "status": info["status"],
                })
            except (psutil.NoSuchProcess, psutil.AccessDenied):
                continue

        processes.sort(key=lambda x: x["cpu_percent"], reverse=True)
        return {"processes": processes[:limit], "total": len(processes)}

    async def get_service_status(self) -> dict:
        services = ["nginx", "postgresql", "mysql", "redis-server", "postfix", "dovecot", "docker"]
        results = []

        for svc in services:
            try:
                proc = await asyncio.create_subprocess_shell(
                    f"systemctl is-active {svc}",
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                )
                stdout, _ = await proc.communicate()
                status = stdout.decode().strip()
                results.append({"name": svc, "status": status})
            except Exception:
                results.append({"name": svc, "status": "unknown"})

        return {"services": results}

    async def control_service(self, service_name: str, action: str) -> dict:
        if action not in ("start", "stop", "restart", "reload"):
            raise ValueError(f"Invalid action: {action}")

        proc = await asyncio.create_subprocess_shell(
            f"systemctl {action} {service_name}",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        _, stderr = await proc.communicate()
        if proc.returncode != 0:
            raise Exception(f"Failed to {action} {service_name}: {stderr.decode()}")

        return {"status": "success", "message": f"Service '{service_name}' {action}ed"}
