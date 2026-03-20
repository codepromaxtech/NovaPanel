"""Application deployment automation."""

from fastapi import APIRouter, HTTPException
from app.schemas.schemas import DeployRequest, DeployResponse
from app.services.deploy_service import DeployService

router = APIRouter()
deploy_service = DeployService()


@router.post("/", response_model=DeployResponse)
async def deploy_application(req: DeployRequest):
    """Deploy an application (git, upload, or container)."""
    try:
        result = await deploy_service.deploy(req)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/status/{app_name}")
async def deployment_status(app_name: str):
    """Get deployment status for an application."""
    try:
        result = await deploy_service.get_status(app_name)
        return result
    except Exception as e:
        raise HTTPException(status_code=404, detail=str(e))


@router.post("/rollback/{app_name}")
async def rollback_deployment(app_name: str):
    """Rollback an application to the previous deployment."""
    try:
        result = await deploy_service.rollback(app_name)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
