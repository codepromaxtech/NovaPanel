"""NovaPanel Automation Service — FastAPI application."""

import os
import logging

from fastapi import FastAPI, Request, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware

from app.routers import nginx, ssl, dns, deploy, database, email, backup, monitoring, git, security

logger = logging.getLogger(__name__)

# API key for authenticating requests from the NovaPanel backend.
# Set AUTOMATION_API_KEY in the environment to enable authentication.
_API_KEY = os.getenv("AUTOMATION_API_KEY", "")
if not _API_KEY:
    logger.warning(
        "AUTOMATION_API_KEY is not set — automation service is running without authentication. "
        "Set this variable to a strong secret and ensure the service is not publicly reachable."
    )

# CORS origins — comma-separated list via env var, defaulting to localhost only
_CORS_ORIGINS_RAW = os.getenv(
    "AUTOMATION_CORS_ORIGINS",
    "http://localhost:3000,http://localhost:8080",
)
_CORS_ORIGINS = [o.strip() for o in _CORS_ORIGINS_RAW.split(",") if o.strip()]

app = FastAPI(
    title="NovaPanel Automation Service",
    description="Automation APIs for web server management, SSL certificates, DNS, and deployments",
    version="1.0.5",
    docs_url="/docs",
    redoc_url="/redoc",
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=_CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.middleware("http")
async def verify_api_key(request: Request, call_next):
    """Require X-API-Key header on all routes except /health and /docs."""
    if not _API_KEY:
        return await call_next(request)

    exempt = {"/health", "/docs", "/redoc", "/openapi.json"}
    if request.url.path in exempt:
        return await call_next(request)

    provided = request.headers.get("X-API-Key", "")
    if provided != _API_KEY:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or missing API key",
        )
    return await call_next(request)


# Register routers
app.include_router(nginx.router, prefix="/api/nginx", tags=["Nginx"])
app.include_router(ssl.router, prefix="/api/ssl", tags=["SSL"])
app.include_router(dns.router, prefix="/api/dns", tags=["DNS"])
app.include_router(deploy.router, prefix="/api/deploy", tags=["Deploy"])
app.include_router(database.router, prefix="/api/database", tags=["Database"])
app.include_router(email.router, prefix="/api/email", tags=["Email"])
app.include_router(backup.router, prefix="/api/backup", tags=["Backup"])
app.include_router(monitoring.router, prefix="/api/monitoring", tags=["Monitoring"])
app.include_router(git.router, prefix="/api/git", tags=["Git"])
app.include_router(security.router, prefix="/api/security", tags=["Security"])


@app.get("/health")
async def health_check():
    return {"status": "ok", "service": "novapanel-automation", "version": "1.0.5"}
