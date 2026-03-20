from fastapi import APIRouter, HTTPException
from ..schemas.schemas import BackupCreate
from ..services.backup_service import BackupOpsService

router = APIRouter(prefix="/backup", tags=["Backup"])
backup_service = BackupOpsService()


@router.post("/create")
async def create_backup(req: BackupCreate):
    try:
        result = await backup_service.create_backup(req.path, req.type, req.storage, req.destination)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/restore")
async def restore_backup(archive_path: str, restore_to: str):
    try:
        result = await backup_service.restore(archive_path, restore_to)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/list")
async def list_backups(storage: str = "local"):
    try:
        result = await backup_service.list_backups(storage)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
