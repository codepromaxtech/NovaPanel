"""DNS zone and record management automation."""

from fastapi import APIRouter, HTTPException
from app.schemas.schemas import DNSRecordCreate, DNSZoneResponse
from app.services.dns_service import DNSService

router = APIRouter()
dns_service = DNSService()


@router.post("/zone/{zone_name}", response_model=DNSZoneResponse)
async def create_zone(zone_name: str):
    """Create a new DNS zone."""
    try:
        result = await dns_service.create_zone(zone_name)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/zone/{zone_name}", response_model=DNSZoneResponse)
async def get_zone(zone_name: str):
    """Get DNS zone and its records."""
    try:
        result = await dns_service.get_zone(zone_name)
        return result
    except FileNotFoundError:
        raise HTTPException(status_code=404, detail=f"Zone {zone_name} not found")


@router.post("/record", response_model=DNSZoneResponse)
async def add_record(req: DNSRecordCreate):
    """Add a DNS record to a zone."""
    try:
        result = await dns_service.add_record(req)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/record/{zone_name}/{record_name}/{record_type}")
async def delete_record(zone_name: str, record_name: str, record_type: str):
    """Delete a DNS record from a zone."""
    try:
        result = await dns_service.delete_record(zone_name, record_name, record_type)
        return {"status": "success", "message": result}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
