import api from './api';

export const securityService = {
    async listRules(serverId?: string) {
        const params = serverId ? { server_id: serverId } : {};
        const { data } = await api.get('/security/rules', { params });
        return data;
    },
    async createRule(payload: {
        server_id: string; port: string; protocol?: string;
        action?: string; source_ip?: string; description?: string;
    }) {
        const { data } = await api.post('/security/rules', payload);
        return data;
    },
    async deleteRule(id: string) {
        await api.delete(`/security/rules/${id}`);
    },
    async listEvents(page = 1, perPage = 50) {
        const { data } = await api.get('/security/events', { params: { page, per_page: perPage } });
        return data;
    },
};
