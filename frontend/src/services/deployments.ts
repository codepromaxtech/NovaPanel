import api from './api';

export const deployService = {
    async list(page = 1, perPage = 20) {
        const { data } = await api.get('/deployments', { params: { page, per_page: perPage } });
        return data;
    },
    async create(payload: { app_id: string; branch?: string }) {
        const { data } = await api.post('/deployments', payload);
        return data;
    },
    async getById(id: string) {
        const { data } = await api.get(`/deployments/${id}`);
        return data;
    },
};
