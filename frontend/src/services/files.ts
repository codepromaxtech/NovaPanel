import api from './api';

export const fileService = {
    async list(path = '/var/www') {
        const { data } = await api.get('/files', { params: { path } });
        return data;
    },
    async read(path: string) {
        const { data } = await api.get('/files/content', { params: { path } });
        return data;
    },
    async write(path: string, content: string) {
        const { data } = await api.put('/files/content', { path, content });
        return data;
    },
    async upload(path: string, file: File) {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('path', path);
        const { data } = await api.post('/files/upload', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
        });
        return data;
    },
    async remove(path: string) {
        await api.delete('/files', { params: { path } });
    },
};
