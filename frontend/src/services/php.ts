import api from './api';

export interface PHPVersion {
    id: string;
    server_id: string;
    version: string;
    is_default: boolean;
    status: string;
    installed_at: string;
}

export const phpService = {
    async list(serverId: string): Promise<PHPVersion[]> {
        const { data } = await api.get(`/servers/${serverId}/php`);
        return data.data || data || [];
    },
    async install(serverId: string, version: string): Promise<PHPVersion> {
        const { data } = await api.post(`/servers/${serverId}/php`, { version });
        return data.data || data;
    },
    async setDefault(serverId: string, version: string) {
        await api.put(`/servers/${serverId}/php/default`, { version });
    },
    async uninstall(serverId: string, version: string) {
        await api.delete(`/servers/${serverId}/php/${version}`);
    },
    async switchDomain(domainId: string, version: string) {
        await api.put(`/domains/${domainId}/php`, { version });
    },
};
