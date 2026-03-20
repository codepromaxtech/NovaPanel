import api from './api';
import type { Server, PaginatedResponse, DashboardStats } from '../types';

export const serverService = {
    async list(page = 1, perPage = 20): Promise<PaginatedResponse<Server>> {
        const { data } = await api.get('/servers', { params: { page, per_page: perPage } });
        return data;
    },

    async getById(id: string): Promise<Server> {
        const { data } = await api.get(`/servers/${id}`);
        return data;
    },

    async create(payload: {
        name: string;
        hostname: string;
        ip_address: string;
        port?: number;
        os?: string;
        role?: string;
        ssh_user?: string;
        ssh_key?: string;
        ssh_password?: string;
        auth_method?: string;
        modules?: string[];
    }): Promise<Server> {
        const { data } = await api.post('/servers', payload);
        return data;
    },

    async remove(id: string): Promise<void> {
        await api.delete(`/servers/${id}`);
    },

    async getDashboardStats(): Promise<DashboardStats> {
        const { data } = await api.get('/dashboard/stats');
        return data;
    },

    // ──── Setup / Provisioning ────
    async getSetupLogs(serverId: string) {
        const { data } = await api.get(`/servers/${serverId}/setup`);
        return data;
    },

    async triggerSetup(serverId: string, modules: string[]) {
        const { data } = await api.post(`/servers/${serverId}/setup`, { modules });
        return data;
    },

    // ──── Modules ────
    async listModules(serverId: string) {
        const { data } = await api.get(`/servers/${serverId}/modules`);
        return data;
    },

    async setModules(serverId: string, modules: string[]) {
        const { data } = await api.put(`/servers/${serverId}/modules`, { modules });
        return data;
    },

    async getAvailableModules() {
        const { data } = await api.get('/modules/available');
        return data;
    },
};
