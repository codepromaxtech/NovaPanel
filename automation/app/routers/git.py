from fastapi import APIRouter, HTTPException
from ..services.git_service import GitOpsService

router = APIRouter(prefix="/git", tags=["Git"])
git_service = GitOpsService()


@router.post("/clone")
async def clone_repo(repo_url: str, target_dir: str, branch: str = "main"):
    try:
        return await git_service.clone(repo_url, target_dir, branch)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/pull")
async def pull_repo(target_dir: str, branch: str = "main"):
    try:
        return await git_service.pull(target_dir, branch)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/build")
async def run_build(target_dir: str, build_command: str = "npm run build"):
    try:
        return await git_service.build(target_dir, build_command)
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
