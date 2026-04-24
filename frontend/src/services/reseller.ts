import api from './api';

export interface ResellerClient {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    role: string;
    status: string;
    created_at: string;
    quota: {
        id: string;
        max_domains: number;
        max_databases: number;
        max_email: number;
        disk_gb: number;
    };
}

export interface ClientUsage {
    domains: number;
    databases: number;
    email_accounts: number;
}

export const resellerService = {
    async listClients() {
        const { data } = await api.get('/reseller/clients');
        return (data.data || data || []) as ResellerClient[];
    },
    async allocateClient(payload: {
        email: string; password: string; first_name: string; last_name?: string;
        max_domains?: number; max_databases?: number; max_email?: number; disk_gb?: number;
    }) {
        const { data } = await api.post('/reseller/clients', payload);
        return data as ResellerClient;
    },
    async updateQuota(clientId: string, payload: {
        max_domains?: number; max_databases?: number; max_email?: number; disk_gb?: number;
    }) {
        const { data } = await api.put(`/reseller/clients/${clientId}`, payload);
        return data;
    },
    async suspendClient(clientId: string) {
        const { data } = await api.delete(`/reseller/clients/${clientId}`);
        return data;
    },
    async getClientUsage(clientId: string) {
        const { data } = await api.get(`/reseller/clients/${clientId}/usage`);
        return data as ClientUsage;
    },
};
