import api from './api';

export const dockerService = {
    // Containers
    async listContainers(all = true) {
        const { data } = await api.get('/docker/containers', { params: { all } });
        return data.data || [];
    },
    async inspectContainer(id: string) {
        const { data } = await api.get(`/docker/containers/${id}`);
        return data;
    },
    async startContainer(id: string) {
        return api.post(`/docker/containers/${id}/start`);
    },
    async stopContainer(id: string) {
        return api.post(`/docker/containers/${id}/stop`);
    },
    async restartContainer(id: string) {
        return api.post(`/docker/containers/${id}/restart`);
    },
    async removeContainer(id: string, force = false) {
        return api.delete(`/docker/containers/${id}`, { params: { force } });
    },
    async containerLogs(id: string, tail = '200') {
        const { data } = await api.get(`/docker/containers/${id}/logs`, { params: { tail } });
        return data.logs || '';
    },
    async createContainer(req: { name: string; image: string; cmd?: string[]; env?: string[]; ports?: Record<string, string>; volumes?: string[]; restart?: string; network?: string; memory_mb?: number; cpus?: number; hostname?: string; labels?: Record<string, string> }) {
        const { data } = await api.post('/docker/containers', req);
        return data;
    },
    async containerStats(id: string) {
        const { data } = await api.get(`/docker/containers/${id}/stats`);
        return data;
    },
    async renameContainer(id: string, name: string) {
        return api.post(`/docker/containers/${id}/rename`, { name });
    },
    async duplicateContainer(id: string, name: string) {
        return api.post(`/docker/containers/${id}/duplicate`, { name });
    },
    async containerProcesses(id: string) {
        const { data } = await api.get(`/docker/containers/${id}/processes`);
        return data;
    },

    // Images
    async listImages() {
        const { data } = await api.get('/docker/images');
        return data.data || [];
    },
    async pullImage(image: string) {
        return api.post('/docker/images/pull', { image });
    },
    async removeImage(id: string, force = false) {
        return api.delete(`/docker/images/${id}`, { params: { force } });
    },
    async imageHistory(id: string) {
        const { data } = await api.get(`/docker/images/${id}/history`);
        return data.data || [];
    },
    async tagImage(id: string, target: string) {
        return api.post(`/docker/images/${id}/tag`, { target });
    },

    // Volumes
    async listVolumes() {
        const { data } = await api.get('/docker/volumes');
        return data.data || [];
    },
    async createVolume(name: string, driver = 'local') {
        return api.post('/docker/volumes', { name, driver });
    },
    async removeVolume(name: string, force = false) {
        return api.delete(`/docker/volumes/${name}`, { params: { force } });
    },

    // Networks
    async listNetworks() {
        const { data } = await api.get('/docker/networks');
        return data.data || [];
    },
    async createNetwork(name: string, driver = 'bridge') {
        return api.post('/docker/networks', { name, driver });
    },
    async removeNetwork(id: string) {
        return api.delete(`/docker/networks/${id}`);
    },
    async networkConnect(networkId: string, containerId: string) {
        return api.post(`/docker/networks/${networkId}/connect`, { container_id: containerId });
    },
    async networkDisconnect(networkId: string, containerId: string) {
        return api.post(`/docker/networks/${networkId}/disconnect`, { container_id: containerId });
    },

    // Stacks
    async listStacks() {
        const { data } = await api.get('/docker/stacks');
        return data.data || [];
    },
    async deployStack(req: { name: string; compose_yaml: string; env_vars?: Record<string, string> }) {
        return api.post('/docker/stacks', req);
    },
    async removeStack(name: string) {
        return api.delete(`/docker/stacks/${name}`);
    },

    // System
    async getStats() {
        const { data } = await api.get('/docker/stats');
        return data;
    },
    async systemInfo() {
        const { data } = await api.get('/docker/system/info');
        return data;
    },
    async pruneSystem() {
        const { data } = await api.post('/docker/system/prune');
        return data;
    },

    // Templates
    async listTemplates() {
        const { data } = await api.get('/docker/templates');
        return data.data || [];
    },
    async deployTemplate(id: string, env?: Record<string, string>, name?: string) {
        const { data } = await api.post(`/docker/templates/${id}/deploy`, { env, name });
        return data;
    },

    // Pause / Unpause / Kill
    async pauseContainer(id: string) {
        return api.post(`/docker/containers/${id}/pause`);
    },
    async unpauseContainer(id: string) {
        return api.post(`/docker/containers/${id}/unpause`);
    },
    async killContainer(id: string, signal = 'SIGKILL') {
        return api.post(`/docker/containers/${id}/kill`, null, { params: { signal } });
    },

    // Commit container (export to image)
    async commitContainer(id: string, repo: string, tag = 'latest', comment = '') {
        const { data } = await api.post(`/docker/containers/${id}/commit`, { repo, tag, comment });
        return data;
    },

    // Container file browser
    async browseFiles(id: string, path = '/') {
        const { data } = await api.get(`/docker/containers/${id}/files`, { params: { path } });
        return data;
    },

    // Image build & push
    async buildImage(dockerfile: string, tag: string) {
        const { data } = await api.post('/docker/images/build', { dockerfile, tag });
        return data;
    },
    async pushImage(ref: string) {
        const { data } = await api.post('/docker/images/push', { ref });
        return data;
    },

    // Events
    async getEvents(since = '1h') {
        const { data } = await api.get('/docker/events', { params: { since } });
        return data.data || [];
    },
};
