import api from './api';

export const fileService = {
    async list(serverId: string, path = '/') {
        const { data } = await api.post('/files/list', { server_id: serverId, path });
        return data;
    },
    async read(serverId: string, path: string) {
        const { data } = await api.post('/files/read', { server_id: serverId, path });
        return data;
    },
    async write(serverId: string, path: string, content: string) {
        const { data } = await api.post('/files/write', { server_id: serverId, path, content });
        return data;
    },
    async create(serverId: string, path: string, isDir = false) {
        const { data } = await api.post('/files/create', { server_id: serverId, path, is_dir: isDir });
        return data;
    },
    async rename(serverId: string, oldPath: string, newPath: string) {
        const { data } = await api.post('/files/rename', { server_id: serverId, old_path: oldPath, new_path: newPath });
        return data;
    },
    async copy(serverId: string, src: string, dest: string) {
        const { data } = await api.post('/files/copy', { server_id: serverId, src, dest });
        return data;
    },
    async move(serverId: string, src: string, dest: string) {
        const { data } = await api.post('/files/move', { server_id: serverId, src, dest });
        return data;
    },
    async remove(serverId: string, path: string) {
        const { data } = await api.post('/files/delete', { server_id: serverId, path });
        return data;
    },
    async chmod(serverId: string, path: string, mode: string) {
        const { data } = await api.post('/files/chmod', { server_id: serverId, path, mode });
        return data;
    },
    async chown(serverId: string, path: string, owner: string) {
        const { data } = await api.post('/files/chown', { server_id: serverId, path, owner });
        return data;
    },
    async search(serverId: string, path: string, query: string) {
        const { data } = await api.post('/files/search', { server_id: serverId, path, query });
        return data;
    },
    async info(serverId: string, path: string) {
        const { data } = await api.post('/files/info', { server_id: serverId, path });
        return data;
    },
    async extract(serverId: string, path: string, dest?: string) {
        const { data } = await api.post('/files/extract', { server_id: serverId, path, dest });
        return data;
    },
    async compress(serverId: string, path: string, dest?: string) {
        const { data } = await api.post('/files/compress', { server_id: serverId, path, dest });
        return data;
    },
    async grep(serverId: string, path: string, pattern: string) {
        const { data } = await api.post('/files/grep', { server_id: serverId, path, pattern });
        return data;
    },
};
