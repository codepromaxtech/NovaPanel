import api from './api';

export const k8sService = {
    // Clusters
    async addCluster(name: string, kubeconfig: string, contextName?: string) {
        return api.post('/k8s/clusters', { name, kubeconfig, context_name: contextName || '' });
    },
    async listClusters() {
        const { data } = await api.get('/k8s/clusters');
        return data.data || [];
    },
    async removeCluster(id: string) {
        return api.delete(`/k8s/clusters/${id}`);
    },
    async clusterInfo(id: string) {
        const { data } = await api.get(`/k8s/clusters/${id}/info`);
        return data.data;
    },

    // Namespaces
    async listNamespaces(cluster: string) {
        const { data } = await api.get(`/k8s/${cluster}/namespaces`);
        return data.data || [];
    },
    async createNamespace(cluster: string, name: string) {
        return api.post(`/k8s/${cluster}/namespaces`, { name });
    },
    async deleteNamespace(cluster: string, ns: string) {
        return api.delete(`/k8s/${cluster}/namespaces/${ns}`);
    },

    // Pods
    async listPods(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/pods`, { params: { ns } });
        return data.data || [];
    },
    async getPod(cluster: string, ns: string, name: string) {
        const { data } = await api.get(`/k8s/${cluster}/pods/${ns}/${name}`);
        return data.data;
    },
    async deletePod(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/pods/${ns}/${name}`);
    },
    async podLogs(cluster: string, ns: string, name: string, container?: string, tail = 100) {
        const { data } = await api.get(`/k8s/${cluster}/pods/${ns}/${name}/logs`, { params: { container, tail } });
        return data.data || '';
    },

    // Deployments
    async listDeployments(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/deployments`, { params: { ns } });
        return data.data || [];
    },
    async scaleDeployment(cluster: string, ns: string, name: string, replicas: number) {
        return api.patch(`/k8s/${cluster}/deployments/${ns}/${name}/scale`, { replicas });
    },
    async restartDeployment(cluster: string, ns: string, name: string) {
        return api.post(`/k8s/${cluster}/deployments/${ns}/${name}/restart`);
    },
    async deleteDeployment(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/deployments/${ns}/${name}`);
    },

    // StatefulSets
    async listStatefulSets(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/statefulsets`, { params: { ns } });
        return data.data || [];
    },
    async scaleStatefulSet(cluster: string, ns: string, name: string, replicas: number) {
        return api.patch(`/k8s/${cluster}/statefulsets/${ns}/${name}/scale`, { replicas });
    },
    async deleteStatefulSet(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/statefulsets/${ns}/${name}`);
    },

    // DaemonSets
    async listDaemonSets(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/daemonsets`, { params: { ns } });
        return data.data || [];
    },
    async deleteDaemonSet(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/daemonsets/${ns}/${name}`);
    },

    // ReplicaSets
    async listReplicaSets(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/replicasets`, { params: { ns } });
        return data.data || [];
    },
    async deleteReplicaSet(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/replicasets/${ns}/${name}`);
    },

    // Jobs
    async listJobs(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/jobs`, { params: { ns } });
        return data.data || [];
    },
    async deleteJob(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/jobs/${ns}/${name}`);
    },

    // CronJobs
    async listCronJobs(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/cronjobs`, { params: { ns } });
        return data.data || [];
    },
    async suspendCronJob(cluster: string, ns: string, name: string, suspend: boolean) {
        return api.patch(`/k8s/${cluster}/cronjobs/${ns}/${name}/suspend`, { suspend });
    },
    async triggerCronJob(cluster: string, ns: string, name: string) {
        return api.post(`/k8s/${cluster}/cronjobs/${ns}/${name}/trigger`);
    },
    async deleteCronJob(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/cronjobs/${ns}/${name}`);
    },

    // Services
    async listServices(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/services`, { params: { ns } });
        return data.data || [];
    },
    async deleteService(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/services/${ns}/${name}`);
    },

    // Ingresses
    async listIngresses(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/ingresses`, { params: { ns } });
        return data.data || [];
    },
    async deleteIngress(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/ingresses/${ns}/${name}`);
    },

    // ConfigMaps
    async listConfigMaps(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/configmaps`, { params: { ns } });
        return data.data || [];
    },
    async getConfigMap(cluster: string, ns: string, name: string) {
        const { data } = await api.get(`/k8s/${cluster}/configmaps/${ns}/${name}`);
        return data.data;
    },
    async deleteConfigMap(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/configmaps/${ns}/${name}`);
    },

    // Secrets
    async listSecrets(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/secrets`, { params: { ns } });
        return data.data || [];
    },
    async getSecret(cluster: string, ns: string, name: string) {
        const { data } = await api.get(`/k8s/${cluster}/secrets/${ns}/${name}`);
        return data.data;
    },
    async deleteSecret(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/secrets/${ns}/${name}`);
    },

    // Storage
    async listPVCs(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/pvcs`, { params: { ns } });
        return data.data || [];
    },
    async deletePVC(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/pvcs/${ns}/${name}`);
    },
    async listPVs(cluster: string) {
        const { data } = await api.get(`/k8s/${cluster}/pvs`);
        return data.data || [];
    },
    async deletePV(cluster: string, name: string) {
        return api.delete(`/k8s/${cluster}/pvs/${name}`);
    },
    async listStorageClasses(cluster: string) {
        const { data } = await api.get(`/k8s/${cluster}/storageclasses`);
        return data.data || [];
    },

    // Nodes
    async listNodes(cluster: string) {
        const { data } = await api.get(`/k8s/${cluster}/nodes`);
        return data.data || [];
    },
    async cordonNode(cluster: string, name: string, cordon: boolean) {
        return api.post(`/k8s/${cluster}/nodes/${name}/cordon`, { cordon });
    },

    // Events
    async listEvents(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/events`, { params: { ns } });
        return data.data || [];
    },

    // RBAC
    async listRoles(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/roles`, { params: { ns } });
        return data.data || [];
    },
    async listClusterRoles(cluster: string) {
        const { data } = await api.get(`/k8s/${cluster}/clusterroles`);
        return data.data || [];
    },
    async listRoleBindings(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/rolebindings`, { params: { ns } });
        return data.data || [];
    },
    async listClusterRoleBindings(cluster: string) {
        const { data } = await api.get(`/k8s/${cluster}/clusterrolebindings`);
        return data.data || [];
    },
    async listServiceAccounts(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/serviceaccounts`, { params: { ns } });
        return data.data || [];
    },
    async deleteServiceAccount(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/serviceaccounts/${ns}/${name}`);
    },

    // HPA
    async listHPAs(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/hpa`, { params: { ns } });
        return data.data || [];
    },
    async deleteHPA(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/hpa/${ns}/${name}`);
    },

    // Network Policies
    async listNetworkPolicies(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/networkpolicies`, { params: { ns } });
        return data.data || [];
    },
    async deleteNetworkPolicy(cluster: string, ns: string, name: string) {
        return api.delete(`/k8s/${cluster}/networkpolicies/${ns}/${name}`);
    },

    // Endpoints
    async listEndpoints(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/endpoints`, { params: { ns } });
        return data.data || [];
    },

    // Resource Quotas
    async listResourceQuotas(cluster: string, ns?: string) {
        const { data } = await api.get(`/k8s/${cluster}/resourcequotas`, { params: { ns } });
        return data.data || [];
    },

    // CRDs
    async listCRDs(cluster: string) {
        const { data } = await api.get(`/k8s/${cluster}/crds`);
        return data.data || [];
    },
};
