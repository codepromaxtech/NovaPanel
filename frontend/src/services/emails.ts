import api from './api';

export const emailService = {
    // ──── Accounts ────
    async list(page = 1, perPage = 20) {
        const { data } = await api.get('/emails', { params: { page, per_page: perPage } });
        return data;
    },
    async create(payload: { address: string; password: string; quota_mb?: number; domain_id: string }) {
        const { data } = await api.post('/emails', payload);
        return data;
    },
    async remove(id: string) {
        await api.delete(`/emails/${id}`);
    },
    async toggleAccount(id: string, isActive: boolean) {
        await api.put(`/emails/${id}/toggle`, { is_active: isActive });
    },
    async changePassword(id: string, password: string) {
        await api.put(`/emails/${id}/password`, { password });
    },
    async updateQuota(id: string, quotaMB: number) {
        await api.put(`/emails/${id}/quota`, { quota_mb: quotaMB });
    },

    // ──── Forwarders ────
    async createForwarder(payload: { source: string; destination: string; domain_id: string }) {
        const { data } = await api.post('/emails/forwarders', payload);
        return data;
    },
    async listForwarders(page = 1, perPage = 20) {
        const { data } = await api.get('/emails/forwarders', { params: { page, per_page: perPage } });
        return data;
    },
    async deleteForwarder(id: string) {
        await api.delete(`/emails/forwarders/${id}`);
    },

    // ──── Aliases ────
    async createAlias(payload: { source: string; destination: string; domain_id: string }) {
        const { data } = await api.post('/emails/aliases', payload);
        return data;
    },
    async listAliases(page = 1, perPage = 20) {
        const { data } = await api.get('/emails/aliases', { params: { page, per_page: perPage } });
        return data;
    },
    async deleteAlias(id: string) {
        await api.delete(`/emails/aliases/${id}`);
    },

    // ──── Autoresponders ────
    async createAutoresponder(payload: { account_id: string; subject: string; body: string; start_date?: string; end_date?: string }) {
        const { data } = await api.post('/emails/autoresponders', payload);
        return data;
    },
    async listAutoresponders(page = 1, perPage = 20) {
        const { data } = await api.get('/emails/autoresponders', { params: { page, per_page: perPage } });
        return data;
    },
    async deleteAutoresponder(id: string) {
        await api.delete(`/emails/autoresponders/${id}`);
    },
    async toggleAutoresponder(id: string, isActive: boolean) {
        await api.put(`/emails/autoresponders/${id}/toggle`, { is_active: isActive });
    },

    // ──── DNS ────
    async getDNSStatus(domain: string) {
        const { data } = await api.get(`/emails/dns/${domain}`);
        return data;
    },

    // ──── Catch-All ────
    async getCatchAll(domainId: string) {
        const { data } = await api.get(`/emails/catchall/${domainId}`);
        return data;
    },
    async setCatchAll(domainId: string, address: string) {
        await api.put(`/emails/catchall/${domainId}`, { address });
    },
};
