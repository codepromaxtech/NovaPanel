import api from './api';
import type { Domain, PaginatedResponse } from '../types';

export const domainService = {
    async list(page = 1, perPage = 20): Promise<PaginatedResponse<Domain>> {
        const { data } = await api.get('/domains', { params: { page, per_page: perPage } });
        return data;
    },

    async getById(id: string): Promise<Domain> {
        const { data } = await api.get(`/domains/${id}`);
        return data;
    },

    async create(domain: {
        name: string;
        type?: string;
        server_id?: string;
        web_server?: string;
        php_version?: string;
        is_load_balancer?: boolean;
        backend_server_ids?: string[];
    }): Promise<Domain> {
        const { data } = await api.post('/domains', domain);
        return data;
    },

    async update(id: string, updates: Partial<Domain>): Promise<Domain> {
        const { data } = await api.put(`/domains/${id}`, updates);
        return data;
    },

    async remove(id: string): Promise<void> {
        await api.delete(`/domains/${id}`);
    },
};
