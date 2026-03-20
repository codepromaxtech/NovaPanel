from fastapi import APIRouter, HTTPException
from ..services.security_ops_service import SecurityOpsService

router = APIRouter(prefix="/security", tags=["Security"])
security_service = SecurityOpsService()


@router.post("/firewall/rule")
async def add_firewall_rule(port: str, action: str = "allow", protocol: str = "tcp", source: str = "any"):
    try:
        return await security_service.add_ufw_rule(port, action, protocol, source)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/firewall/rule")
async def delete_firewall_rule(port: str, protocol: str = "tcp"):
    try:
        return await security_service.delete_ufw_rule(port, protocol)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/fail2ban/status")
async def fail2ban_status():
    try:
        return await security_service.fail2ban_status()
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/fail2ban/{jail}/{action}")
async def fail2ban_action(jail: str, action: str):
    try:
        return await security_service.fail2ban_control(jail, action)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
