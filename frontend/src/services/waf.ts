import api from './api';

export const wafService = {
    // ──── Config ────
    async getConfig(serverId: string) {
        const { data } = await api.get(`/waf/config/${serverId}`);
        return data;
    },
    async updateConfig(serverId: string, payload: Record<string, any>) {
        const { data } = await api.put(`/waf/config/${serverId}`, payload);
        return data;
    },

    // ──── Disabled Rules ────
    async listDisabledRules(serverId: string) {
        const { data } = await api.get(`/waf/rules/${serverId}`);
        return data;
    },
    async disableRule(serverId: string, ruleId: number, description: string) {
        const { data } = await api.post(`/waf/rules/${serverId}`, { rule_id: ruleId, description });
        return data;
    },
    async enableRule(id: string) {
        await api.delete(`/waf/rules/${id}`);
    },

    // ──── Whitelist ────
    async listWhitelist(serverId: string) {
        const { data } = await api.get(`/waf/whitelist/${serverId}`);
        return data;
    },
    async addWhitelist(serverId: string, payload: { type: string; value: string; reason: string }) {
        const { data } = await api.post(`/waf/whitelist/${serverId}`, payload);
        return data;
    },
    async removeWhitelist(id: string) {
        await api.delete(`/waf/whitelist/${id}`);
    },

    // ──── Logs ────
    async listLogs(serverId: string, page = 1, perPage = 50) {
        const { data } = await api.get(`/waf/logs/${serverId}`, { params: { page, per_page: perPage } });
        return data;
    },
};
