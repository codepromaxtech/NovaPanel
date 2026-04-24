import api from './api';

export interface Application {
    id: string;
    user_id: string;
    server_id: string;
    domain_id?: string;
    name: string;
    app_type: string;
    runtime: string;
    deploy_method: string;
    git_repo: string;
    git_branch: string;
    status: string;
    env_vars?: Record<string, string>;
    created_at: string;
    updated_at: string;
}

export const appService = {
    async list(page = 1, perPage = 50) {
        const { data } = await api.get('/apps', { params: { page, per_page: perPage } });
        return data;
    },
    async create(payload: {
        name: string;
        server_id: string;
        app_type?: string;
        runtime?: string;
        deploy_method?: string;
        git_repo?: string;
        git_branch?: string;
        domain_id?: string;
        env_vars?: Record<string, string>;
    }): Promise<Application> {
        const { data } = await api.post('/apps', payload);
        return data;
    },
    async getById(id: string): Promise<Application> {
        const { data } = await api.get(`/apps/${id}`);
        return data;
    },
    async update(id: string, payload: Partial<Application>): Promise<Application> {
        const { data } = await api.put(`/apps/${id}`, payload);
        return data;
    },
    async remove(id: string) {
        const { data } = await api.delete(`/apps/${id}`);
        return data;
    },
};
