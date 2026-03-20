import api from './api';

export const backupService = {
    async list(page = 1, perPage = 20) {
        const { data } = await api.get('/backups', { params: { page, per_page: perPage } });
        return data;
    },
    async create(payload: { type?: string; storage?: string }) {
        const { data } = await api.post('/backups', payload);
        return data;
    },
    async remove(id: string) {
        await api.delete(`/backups/${id}`);
    },
};
