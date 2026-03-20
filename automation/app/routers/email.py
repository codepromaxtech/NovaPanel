from fastapi import APIRouter, HTTPException
from ..schemas.schemas import EmailAccountCreate, EmailForwarderCreate
from ..services.email_service import EmailOpsService

router = APIRouter(prefix="/email", tags=["Email"])
email_service = EmailOpsService()


@router.post("/accounts")
async def create_mailbox(req: EmailAccountCreate):
    try:
        result = await email_service.create_mailbox(req.address, req.password, req.quota_mb)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/accounts/{address}")
async def delete_mailbox(address: str):
    try:
        result = await email_service.delete_mailbox(address)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/forwarders")
async def create_forwarder(req: EmailForwarderCreate):
    try:
        result = await email_service.create_forwarder(req.source, req.destination)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
