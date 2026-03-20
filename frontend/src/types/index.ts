// API Types for NovaPanel

export interface User {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    role: 'admin' | 'reseller' | 'client';
    status: string;
    two_factor_enabled: boolean;
    last_login_at: string | null;
    created_at: string;
    updated_at: string;
}

export interface AuthResponse {
    token: string;
    expires_at: number;
    user: User;
}

export interface Server {
    id: string;
    name: string;
    hostname: string;
    ip_address: string;
    port: number;
    os: string;
    role: string;
    status: 'pending' | 'active' | 'maintenance' | 'offline';
    agent_version: string;
    agent_status: 'connected' | 'disconnected';
    last_heartbeat: string | null;
    created_at: string;
    updated_at: string;
}

export interface Domain {
    id: string;
    user_id: string;
    server_id: string | null;
    name: string;
    type: 'primary' | 'subdomain' | 'parked' | 'addon';
    document_root: string;
    web_server: string;
    php_version: string;
    ssl_enabled: boolean;
    status: 'active' | 'suspended' | 'pending';
    is_load_balancer?: boolean;
    backend_server_ids?: string[];
    created_at: string;
    updated_at: string;
}

export interface DashboardStats {
    total_servers: number;
    active_servers: number;
    total_domains: number;
    total_users: number;
    total_apps: number;
    pending_tasks: number;
}

export interface PaginatedResponse<T> {
    data: T[];
    total: number;
    page: number;
    per_page: number;
    total_pages: number;
}

export interface Task {
    id: string;
    type: string;
    status: 'queued' | 'running' | 'completed' | 'failed';
    priority: number;
    error: string;
    attempts: number;
    created_at: string;
}
