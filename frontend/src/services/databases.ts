import api from './api';

export const databaseService = {
    // ──── CRUD ────
    async list(page = 1, perPage = 20) {
        const { data } = await api.get('/databases', { params: { page, per_page: perPage } });
        return data;
    },
    async create(payload: { name: string; engine?: string; charset?: string; server_id?: string }) {
        const { data } = await api.post('/databases', payload);
        return data;
    },
    async remove(id: string) {
        await api.delete(`/databases/${id}`);
    },

    // ──── DB Manager ────
    async runQuery(serverId: string, engine: string, database: string, query: string) {
        const { data } = await api.post('/db-manager/query', { server_id: serverId, engine, database, query });
        return data;
    },
    async listTables(serverId: string, engine: string, database: string) {
        const { data } = await api.post('/db-manager/tables', { server_id: serverId, engine, database });
        return data;
    },
    async describeTable(serverId: string, engine: string, database: string, table: string) {
        const { data } = await api.post('/db-manager/describe', { server_id: serverId, engine, database, table });
        return data;
    },
    async getDBSize(serverId: string, engine: string, database: string) {
        const { data } = await api.post('/db-manager/size', { server_id: serverId, engine, database });
        return data;
    },
    async listDBsOnServer(serverId: string, engine: string) {
        const { data } = await api.post('/db-manager/databases', { server_id: serverId, engine });
        return data;
    },
    async listUsers(serverId: string, engine: string) {
        const { data } = await api.post('/db-manager/users', { server_id: serverId, engine });
        return data;
    },
    async createUser(serverId: string, engine: string, username: string, password: string, database: string) {
        const { data } = await api.post('/db-manager/users/create', { server_id: serverId, engine, username, password, database });
        return data;
    },
    async exportDB(serverId: string, engine: string, database: string) {
        const { data } = await api.post('/db-manager/export', { server_id: serverId, engine, database });
        return data;
    },
    async importDB(serverId: string, engine: string, database: string, filePath: string) {
        const { data } = await api.post('/db-manager/import', { server_id: serverId, engine, database, file_path: filePath });
        return data;
    },

    // ──── Web Tools ────
    async deployTool(serverId: string, engine: string) {
        const { data } = await api.post('/db-manager/tools/deploy', { server_id: serverId, engine });
        return data;
    },
    async toolStatus(serverId: string, engine: string) {
        const { data } = await api.post('/db-manager/tools/status', { server_id: serverId, engine });
        return data;
    },
    async stopTool(serverId: string, engine: string) {
        const { data } = await api.post('/db-manager/tools/stop', { server_id: serverId, engine });
        return data;
    },
};
