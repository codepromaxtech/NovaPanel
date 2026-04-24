import api from './api';

export interface FTPAccount {
    id: string;
    user_id: string;
    server_id: string;
    username: string;
    home_dir: string;
    quota_mb: number;
    is_active: boolean;
    created_at: string;
}

export interface SFTPKey {
    id: string;
    ftp_account_id: string;
    label: string;
    fingerprint: string;
    created_at: string;
}

export const ftpService = {
    async list() {
        const { data } = await api.get('/ftp');
        return (data.data || []) as FTPAccount[];
    },
    async create(serverId: string, payload: { username: string; password: string; home_dir?: string; quota_mb?: number }) {
        const { data } = await api.post(`/servers/${serverId}/ftp`, payload);
        return data as FTPAccount;
    },
    async delete(serverId: string, ftpID: string) {
        await api.delete(`/servers/${serverId}/ftp/${ftpID}`);
    },
    // SFTP key management
    async listKeys(serverId: string, ftpID: string) {
        const { data } = await api.get(`/servers/${serverId}/ftp/${ftpID}/keys`);
        return (data.data || []) as SFTPKey[];
    },
    async addKey(serverId: string, ftpID: string, payload: { label?: string; public_key: string }) {
        const { data } = await api.post(`/servers/${serverId}/ftp/${ftpID}/keys`, payload);
        return data as SFTPKey;
    },
    async deleteKey(serverId: string, ftpID: string, keyID: string) {
        await api.delete(`/servers/${serverId}/ftp/${ftpID}/keys/${keyID}`);
    },
};
