"""NovaPanel Automation Service — FastAPI application."""

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.routers import nginx, ssl, dns, deploy, database, email, backup, monitoring, git, security

app = FastAPI(
    title="NovaPanel Automation Service",
    description="Automation APIs for web server management, SSL certificates, DNS, and deployments",
    version="0.3.0",
    docs_url="/docs",
    redoc_url="/redoc",
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://localhost:8080"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

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
    return {"status": "ok", "service": "novapanel-automation", "version": "0.3.0"}
