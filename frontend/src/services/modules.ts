import api from './api';

export interface ModuleInfo {
    id: string;
    label: string;
    category: string;
    icon: string;
}

export interface ServerModule {
    id: string;
    server_id: string;
    module: string;
    enabled: boolean;
    config: string;
    installed_at: string;
}

export const modulesService = {
    // Per-server
    async listModules(serverId: string): Promise<ServerModule[]> {
        const { data } = await api.get(`/servers/${serverId}/modules`);
        return data.data || [];
    },
    async enableModule(serverId: string, module: string) {
        return api.post(`/servers/${serverId}/modules`, { module });
    },
    async disableModule(serverId: string, module: string) {
        return api.delete(`/servers/${serverId}/modules/${module}`);
    },
    async setModules(serverId: string, modules: string[]) {
        return api.put(`/servers/${serverId}/modules`, { modules });
    },

    // Global
    async listAvailable(): Promise<ModuleInfo[]> {
        const { data } = await api.get('/modules/available');
        return data.data || [];
    },
    async listActive(): Promise<string[]> {
        const { data } = await api.get('/modules/active');
        return data.data || [];
    },
    async moduleCounts(): Promise<Record<string, number>> {
        const { data } = await api.get('/modules/counts');
        return data.data || {};
    },
};
