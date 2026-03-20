"""Nginx vhost management automation."""

import os
from fastapi import APIRouter, HTTPException
from app.schemas.schemas import NginxVhostCreate, NginxVhostResponse
from app.services.nginx_service import NginxService

router = APIRouter()
nginx_service = NginxService()


@router.post("/vhost", response_model=NginxVhostResponse)
async def create_vhost(req: NginxVhostCreate):
    """Create a new Nginx virtual host configuration."""
    try:
        result = await nginx_service.create_vhost(req)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/vhost/{domain}", response_model=NginxVhostResponse)
async def delete_vhost(domain: str):
    """Remove an Nginx virtual host configuration."""
    try:
        result = await nginx_service.delete_vhost(domain)
        return result
    except FileNotFoundError:
        raise HTTPException(status_code=404, detail=f"Vhost for {domain} not found")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/vhost/{domain}")
async def get_vhost(domain: str):
    """Get Nginx virtual host configuration."""
    try:
        config = await nginx_service.get_vhost_config(domain)
        return {"domain": domain, "config": config, "status": "active"}
    except FileNotFoundError:
        raise HTTPException(status_code=404, detail=f"Vhost for {domain} not found")


@router.post("/reload")
async def reload_nginx():
    """Reload Nginx configuration."""
    try:
        result = await nginx_service.reload()
        return {"status": "success", "message": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/test")
async def test_config():
    """Test Nginx configuration syntax."""
    try:
        result = await nginx_service.test_config()
        return {"status": "success", "message": result}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
