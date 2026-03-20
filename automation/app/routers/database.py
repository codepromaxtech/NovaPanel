from fastapi import APIRouter, HTTPException
from ..schemas.schemas import DatabaseCreate, DatabaseResponse
from ..services.database_service import DatabaseOpsService

router = APIRouter(prefix="/database", tags=["Database"])
db_service = DatabaseOpsService()


@router.post("/create", response_model=DatabaseResponse)
async def create_database(req: DatabaseCreate):
    try:
        result = await db_service.create_database(req.name, req.engine, req.charset)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/user/create", response_model=DatabaseResponse)
async def create_db_user(req: DatabaseCreate):
    try:
        result = await db_service.create_user(req.name, req.db_user, req.db_password)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/{db_name}")
async def drop_database(db_name: str, engine: str = "mysql"):
    try:
        result = await db_service.drop_database(db_name, engine)
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
