"""SSL certificate management automation."""

from fastapi import APIRouter, HTTPException
from app.schemas.schemas import SSLRequest, SSLResponse
from app.services.ssl_service import SSLService

router = APIRouter()
ssl_service = SSLService()


@router.post("/issue", response_model=SSLResponse)
async def issue_certificate(req: SSLRequest):
    """Issue a new SSL certificate via Let's Encrypt."""
    try:
        result = await ssl_service.issue_certificate(req)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/renew/{domain}", response_model=SSLResponse)
async def renew_certificate(domain: str):
    """Renew an existing SSL certificate."""
    try:
        result = await ssl_service.renew_certificate(domain)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/status/{domain}", response_model=SSLResponse)
async def certificate_status(domain: str):
    """Check SSL certificate status and expiry."""
    try:
        result = await ssl_service.get_certificate_status(domain)
        return result
    except FileNotFoundError:
        raise HTTPException(status_code=404, detail=f"No certificate found for {domain}")


@router.delete("/revoke/{domain}")
async def revoke_certificate(domain: str):
    """Revoke an SSL certificate."""
    try:
        result = await ssl_service.revoke_certificate(domain)
        return {"status": "success", "message": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
