from fastapi import APIRouter
from ..services.monitoring_service import MonitoringOpsService

router = APIRouter(prefix="/monitoring", tags=["Monitoring"])
monitoring_service = MonitoringOpsService()


@router.get("/metrics")
async def get_system_metrics():
    return await monitoring_service.get_metrics()


@router.get("/processes")
async def get_processes():
    return await monitoring_service.get_top_processes()


@router.get("/services")
async def get_services():
    return await monitoring_service.get_service_status()


@router.post("/services/{service_name}/{action}")
async def control_service(service_name: str, action: str):
    return await monitoring_service.control_service(service_name, action)
