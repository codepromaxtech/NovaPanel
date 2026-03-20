"""Pydantic schemas for automation service."""

from pydantic import BaseModel, Field
from typing import Optional
from enum import Enum


class WebServerType(str, Enum):
    NGINX = "nginx"
    APACHE = "apache"
    OPENLITESPEED = "openlitespeed"
    CADDY = "caddy"


class PHPVersion(str, Enum):
    PHP74 = "7.4"
    PHP80 = "8.0"
    PHP81 = "8.1"
    PHP82 = "8.2"
    PHP83 = "8.3"


# --- Nginx schemas ---

class NginxVhostCreate(BaseModel):
    domain: str = Field(..., description="Domain name, e.g. example.com")
    document_root: str = Field(default="/var/www", description="Web root directory")
    php_version: Optional[str] = Field(default="8.2", description="PHP-FPM version")
    ssl_enabled: bool = Field(default=False)
    proxy_pass: Optional[str] = Field(default=None, description="Reverse proxy upstream URL")
    template: str = Field(default="default", description="Vhost template name")
    is_load_balancer: bool = Field(default=False, description="Generate an upstream block")
    target_ips: list[str] = Field(default_factory=list, description="Backend worker IPs")


class NginxVhostResponse(BaseModel):
    domain: str
    config_path: str
    status: str
    message: str


# --- SSL schemas ---

class SSLRequest(BaseModel):
    domain: str = Field(..., description="Domain to issue certificate for")
    email: str = Field(..., description="Admin email for Let's Encrypt")
    webroot: Optional[str] = Field(default="/var/www", description="Webroot path")


class SSLResponse(BaseModel):
    domain: str
    status: str
    certificate_path: Optional[str] = None
    private_key_path: Optional[str] = None
    expires_at: Optional[str] = None
    message: str


# --- DNS schemas ---

class DNSRecordType(str, Enum):
    A = "A"
    AAAA = "AAAA"
    CNAME = "CNAME"
    MX = "MX"
    TXT = "TXT"
    NS = "NS"
    SRV = "SRV"


class DNSRecordCreate(BaseModel):
    zone: str = Field(..., description="DNS zone, e.g. example.com")
    name: str = Field(..., description="Record name, e.g. www")
    record_type: DNSRecordType
    content: str = Field(..., description="Record content/value")
    ttl: int = Field(default=3600, ge=60, le=86400)
    priority: Optional[int] = Field(default=None, description="Priority for MX records")


class DNSZoneResponse(BaseModel):
    zone: str
    records: list
    status: str
    message: str


# --- Deploy schemas ---

class DeployMethod(str, Enum):
    GIT = "git"
    UPLOAD = "upload"
    CONTAINER = "container"


class DeployRequest(BaseModel):
    app_name: str
    domain: str
    deploy_method: DeployMethod
    git_repo: Optional[str] = None
    git_branch: str = Field(default="main")
    runtime: Optional[str] = None
    env_vars: dict = Field(default_factory=dict)


class DeployResponse(BaseModel):
    app_name: str
    domain: str
    status: str
    deploy_log: Optional[str] = None
    message: str


# --- Phase 2: Database schemas ---

class DatabaseCreate(BaseModel):
    name: str = Field(..., description="Database name")
    engine: str = Field(default="mysql", description="mysql, mariadb, or postgresql")
    charset: str = Field(default="utf8mb4")
    db_user: Optional[str] = None
    db_password: Optional[str] = None


class DatabaseResponse(BaseModel):
    status: str
    message: str


# --- Phase 2: Email schemas ---

class EmailAccountCreate(BaseModel):
    address: str = Field(..., description="Full email address, e.g. user@domain.com")
    password: str = Field(..., min_length=8)
    quota_mb: int = Field(default=1024)


class EmailForwarderCreate(BaseModel):
    source: str = Field(..., description="Source address")
    destination: str = Field(..., description="Destination address")


# --- Phase 2: Backup schemas ---

class BackupCreate(BaseModel):
    path: str = Field(default="/var/www", description="Path to back up")
    type: str = Field(default="full", description="full or incremental")
    storage: str = Field(default="local", description="local or s3")
    destination: Optional[str] = Field(default=None, description="S3 bucket path")

