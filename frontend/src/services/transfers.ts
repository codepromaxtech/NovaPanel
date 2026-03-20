import api from './api';

export const transferService = {
    // ──── Transfer Jobs ────
    async list(page = 1, perPage = 20) {
        const { data } = await api.get('/transfers', { params: { page, per_page: perPage } });
        return data;
    },
    async create(payload: Record<string, any>) {
        const { data } = await api.post('/transfers', payload);
        return data;
    },
    async get(id: string) {
        const { data } = await api.get(`/transfers/${id}`);
        return data;
    },
    async cancel(id: string) {
        const { data } = await api.post(`/transfers/${id}/cancel`);
        return data;
    },
    async retry(id: string) {
        const { data } = await api.post(`/transfers/${id}/retry`);
        return data;
    },
    async remove(id: string) {
        await api.delete(`/transfers/${id}`);
    },
    async preview(payload: Record<string, any>) {
        const { data } = await api.post('/transfers/preview', payload);
        return data;
    },
    async diskUsage(serverId: string, path: string) {
        const { data } = await api.post('/transfers/disk-usage', { server_id: serverId, path });
        return data;
    },

    // ──── Schedules ────
    async listSchedules() {
        const { data } = await api.get('/transfers/schedules');
        return data;
    },
    async createSchedule(payload: Record<string, any>) {
        const { data } = await api.post('/transfers/schedules', payload);
        return data;
    },
    async deleteSchedule(id: string) {
        await api.delete(`/transfers/schedules/${id}`);
    },
    async toggleSchedule(id: string, isActive: boolean) {
        const { data } = await api.put(`/transfers/schedules/${id}/toggle`, { is_active: isActive });
        return data;
    },
};
