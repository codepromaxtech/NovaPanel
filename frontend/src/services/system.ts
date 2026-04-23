import api from './api';

export const updateService = {
    async getStatus() {
        const { data } = await api.get('/settings/updates');
        return data as {
            current_version: string;
            latest_version: string;
            update_available: boolean;
            release_notes: string;
            release_url: string;
            last_checked: string;
            applying: boolean;
        };
    },
    async checkNow() {
        const { data } = await api.post('/settings/updates/check');
        return data;
    },
    async applyUpdate() {
        const { data } = await api.post('/settings/updates/apply');
        return data;
    },
};

export const cronService = {
    async list(serverId: string, user = 'root') {
        const { data } = await api.post('/cron/list', { server_id: serverId, user });
        return data;
    },
    async add(serverId: string, schedule: string, command: string, user = 'root', comment = '') {
        const { data } = await api.post('/cron/add', { server_id: serverId, user, schedule, command, comment });
        return data;
    },
    async remove(serverId: string, lineNumber: number, user = 'root') {
        const { data } = await api.post('/cron/delete', { server_id: serverId, user, line_number: lineNumber });
        return data;
    },
    async update(serverId: string, lineNumber: number, newLine: string, user = 'root') {
        const { data } = await api.post('/cron/update', { server_id: serverId, user, line_number: lineNumber, new_line: newLine });
        return data;
    },
    async listUsers(serverId: string) {
        const { data } = await api.post('/cron/users', { server_id: serverId });
        return data;
    },
    async logs(serverId: string, lines = 50) {
        const { data } = await api.post('/cron/logs', { server_id: serverId, lines });
        return data;
    },
};

export const systemctlService = {
    async list(serverId: string, filter = '') {
        const { data } = await api.post('/systemctl/list', { server_id: serverId, filter });
        return data;
    },
    async status(serverId: string, service: string) {
        const { data } = await api.post('/systemctl/status', { server_id: serverId, service });
        return data;
    },
    async start(serverId: string, service: string) {
        const { data } = await api.post('/systemctl/start', { server_id: serverId, service });
        return data;
    },
    async stop(serverId: string, service: string) {
        const { data } = await api.post('/systemctl/stop', { server_id: serverId, service });
        return data;
    },
    async restart(serverId: string, service: string) {
        const { data } = await api.post('/systemctl/restart', { server_id: serverId, service });
        return data;
    },
    async reload(serverId: string, service: string) {
        const { data } = await api.post('/systemctl/reload', { server_id: serverId, service });
        return data;
    },
    async enable(serverId: string, service: string) {
        const { data } = await api.post('/systemctl/enable', { server_id: serverId, service });
        return data;
    },
    async disable(serverId: string, service: string) {
        const { data } = await api.post('/systemctl/disable', { server_id: serverId, service });
        return data;
    },
    async logs(serverId: string, service: string, lines = 50) {
        const { data } = await api.post('/systemctl/logs', { server_id: serverId, service, lines });
        return data;
    },
    async failed(serverId: string) {
        const { data } = await api.post('/systemctl/failed', { server_id: serverId });
        return data;
    },
    async timers(serverId: string) {
        const { data } = await api.post('/systemctl/timers', { server_id: serverId });
        return data;
    },
    async daemonReload(serverId: string) {
        const { data } = await api.post('/systemctl/daemon-reload', { server_id: serverId });
        return data;
    },
};

export const backupManagerService = {
    async backupDatabase(serverId: string, engine: string, database: string) {
        const { data } = await api.post('/backup-manager/database', { server_id: serverId, engine, database });
        return data;
    },
    async backupSite(serverId: string, sitePath: string, siteName = '') {
        const { data } = await api.post('/backup-manager/site', { server_id: serverId, site_path: sitePath, site_name: siteName });
        return data;
    },
    async backupFull(serverId: string) {
        const { data } = await api.post('/backup-manager/full', { server_id: serverId });
        return data;
    },
    async restoreDatabase(serverId: string, engine: string, database: string, backupPath: string) {
        const { data } = await api.post('/backup-manager/restore/database', { server_id: serverId, engine, database, backup_path: backupPath });
        return data;
    },
    async restoreSite(serverId: string, sitePath: string, backupPath: string) {
        const { data } = await api.post('/backup-manager/restore/site', { server_id: serverId, site_path: sitePath, backup_path: backupPath });
        return data;
    },
    async listFiles(serverId: string) {
        const { data } = await api.post('/backup-manager/files', { server_id: serverId });
        return data;
    },
    async deleteFile(serverId: string, filePath: string) {
        const { data } = await api.post('/backup-manager/files/delete', { server_id: serverId, file_path: filePath });
        return data;
    },
};
