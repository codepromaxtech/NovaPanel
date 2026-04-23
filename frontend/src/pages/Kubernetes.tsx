import { useEffect, useState } from 'react';
import {
    Box, Plus, Trash2, RefreshCw, Loader, X, Server, Layers,
    Container, Globe, Shield, HardDrive, Activity, Settings,
    Play, Pause, FileText, ChevronDown, Upload, Scale,
} from 'lucide-react';
import { k8sService } from '../services/k8s';
import { useToast } from '../components/ui/ToastProvider';

type Tab = 'clusters' | 'workloads' | 'jobs' | 'services' | 'config' | 'storage' | 'rbac' | 'scaling' | 'events' | 'crds';

const TABS: { id: Tab; label: string; icon: any }[] = [
    { id: 'clusters', label: 'Clusters', icon: Server },
    { id: 'workloads', label: 'Workloads', icon: Box },
    { id: 'jobs', label: 'Jobs', icon: Play },
    { id: 'services', label: 'Services', icon: Globe },
    { id: 'config', label: 'Config', icon: FileText },
    { id: 'storage', label: 'Storage', icon: HardDrive },
    { id: 'rbac', label: 'RBAC', icon: Shield },
    { id: 'scaling', label: 'Scaling', icon: Scale },
    { id: 'events', label: 'Events', icon: Activity },
    { id: 'crds', label: 'CRDs', icon: Settings },
];

export default function Kubernetes() {
    const toast = useToast();
    const [tab, setTab] = useState<Tab>('clusters');
    const [clusters, setClusters] = useState<any[]>([]);
    const [selectedCluster, setSelectedCluster] = useState('');
    const [loading, setLoading] = useState(false);
    const [showAdd, setShowAdd] = useState(false);
    const [addForm, setAddForm] = useState({ name: '', kubeconfig: '' });
    const [namespaces, setNamespaces] = useState<string[]>([]);
    const [selectedNs, setSelectedNs] = useState('');
    const [data, setData] = useState<any[]>([]);

    const loadClusters = async () => {
        try {
            const list = await k8sService.listClusters();
            setClusters(list || []);
            if (list?.length && !selectedCluster) setSelectedCluster(list[0].id);
        } catch { setClusters([]); }
    };

    useEffect(() => { loadClusters(); }, []);

    useEffect(() => {
        if (!selectedCluster) return;
        k8sService.listNamespaces(selectedCluster).then(ns => {
            setNamespaces(ns || []);
            if (!selectedNs && ns?.length) setSelectedNs(ns[0]);
        }).catch(() => { });
    }, [selectedCluster]);

    useEffect(() => {
        if (!selectedCluster || tab === 'clusters') return;
        loadTabData();
    }, [selectedCluster, selectedNs, tab]);

    const loadTabData = async () => {
        setLoading(true);
        try {
            const c = selectedCluster;
            const ns = selectedNs;
            switch (tab) {
                case 'workloads': {
                    const [pods, deps, sts, ds] = await Promise.all([
                        k8sService.listPods(c, ns), k8sService.listDeployments(c, ns),
                        k8sService.listStatefulSets(c, ns), k8sService.listDaemonSets(c, ns),
                    ]);
                    setData([
                        ...pods.map((p: any) => ({ ...p, _type: 'Pod' })),
                        ...deps.map((d: any) => ({ ...d, _type: 'Deployment' })),
                        ...sts.map((s: any) => ({ ...s, _type: 'StatefulSet' })),
                        ...ds.map((d: any) => ({ ...d, _type: 'DaemonSet' })),
                    ]);
                    break;
                }
                case 'jobs': {
                    const [jobs, crons] = await Promise.all([
                        k8sService.listJobs(c, ns), k8sService.listCronJobs(c, ns),
                    ]);
                    setData([
                        ...jobs.map((j: any) => ({ ...j, _type: 'Job' })),
                        ...crons.map((c: any) => ({ ...c, _type: 'CronJob' })),
                    ]);
                    break;
                }
                case 'services': {
                    const [svcs, ings, eps] = await Promise.all([
                        k8sService.listServices(c, ns), k8sService.listIngresses(c, ns),
                        k8sService.listEndpoints(c, ns),
                    ]);
                    setData([
                        ...svcs.map((s: any) => ({ ...s, _type: 'Service' })),
                        ...ings.map((i: any) => ({ ...i, _type: 'Ingress' })),
                        ...eps.map((e: any) => ({ ...e, _type: 'Endpoint' })),
                    ]);
                    break;
                }
                case 'config': {
                    const [cms, secs] = await Promise.all([
                        k8sService.listConfigMaps(c, ns), k8sService.listSecrets(c, ns),
                    ]);
                    setData([
                        ...cms.map((m: any) => ({ ...m, _type: 'ConfigMap' })),
                        ...secs.map((s: any) => ({ ...s, _type: 'Secret' })),
                    ]);
                    break;
                }
                case 'storage': {
                    const [pvcs, pvs, scs] = await Promise.all([
                        k8sService.listPVCs(c, ns), k8sService.listPVs(c), k8sService.listStorageClasses(c),
                    ]);
                    setData([
                        ...pvcs.map((p: any) => ({ ...p, _type: 'PVC' })),
                        ...pvs.map((p: any) => ({ ...p, _type: 'PV' })),
                        ...scs.map((s: any) => ({ ...s, _type: 'StorageClass' })),
                    ]);
                    break;
                }
                case 'rbac': {
                    const [roles, crs, sas] = await Promise.all([
                        k8sService.listRoles(c, ns), k8sService.listClusterRoles(c),
                        k8sService.listServiceAccounts(c, ns),
                    ]);
                    setData([...roles, ...crs, ...sas]);
                    break;
                }
                case 'scaling': setData(await k8sService.listHPAs(c, ns)); break;
                case 'events': setData(await k8sService.listEvents(c, ns)); break;
                case 'crds': setData(await k8sService.listCRDs(c)); break;
            }
        } catch { setData([]); }
        setLoading(false);
    };

    const handleAddCluster = async () => {
        if (!addForm.name || !addForm.kubeconfig) return;
        try {
            await k8sService.addCluster(addForm.name, addForm.kubeconfig);
            toast.success('Cluster added');
            setShowAdd(false);
            setAddForm({ name: '', kubeconfig: '' });
            loadClusters();
        } catch { toast.error('Failed to add cluster'); }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Remove this cluster?')) return;
        try {
            await k8sService.removeCluster(id);
            toast.success('Cluster removed');
            loadClusters();
        } catch { toast.error('Failed'); }
    };

    const typeColor = (t: string) => {
        const c: Record<string, string> = {
            Pod: 'text-blue-400 bg-blue-500/10', Deployment: 'text-green-400 bg-green-500/10',
            StatefulSet: 'text-purple-400 bg-purple-500/10', DaemonSet: 'text-orange-400 bg-orange-500/10',
            Job: 'text-yellow-400 bg-yellow-500/10', CronJob: 'text-amber-400 bg-amber-500/10',
            Service: 'text-cyan-400 bg-cyan-500/10', Ingress: 'text-teal-400 bg-teal-500/10',
            ConfigMap: 'text-indigo-400 bg-indigo-500/10', Secret: 'text-red-400 bg-red-500/10',
            PVC: 'text-violet-400 bg-violet-500/10', PV: 'text-fuchsia-400 bg-fuchsia-500/10',
            StorageClass: 'text-pink-400 bg-pink-500/10',
        };
        return c[t] || 'text-surface-200/60 bg-white/5';
    };

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Layers className="w-7 h-7 text-blue-400" /> Kubernetes
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">{clusters.length} cluster{clusters.length !== 1 ? 's' : ''} connected</p>
                </div>
                <div className="flex items-center gap-3">
                    {clusters.length > 0 && (
                        <select value={selectedCluster} onChange={e => setSelectedCluster(e.target.value)}
                            className="px-3 py-2 rounded-lg glass-input text-white text-sm bg-transparent">
                            {clusters.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                        </select>
                    )}
                    {selectedCluster && namespaces.length > 0 && tab !== 'clusters' && (
                        <select value={selectedNs} onChange={e => setSelectedNs(e.target.value)}
                            className="px-3 py-2 rounded-lg glass-input text-white text-sm bg-transparent">
                            <option value="">All Namespaces</option>
                            {namespaces.map(n => <option key={n} value={n}>{n}</option>)}
                        </select>
                    )}
                    <button onClick={() => setShowAdd(true)}
                        className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-blue-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-blue-500/25 transition-all">
                        <Plus className="w-4 h-4" /> Add Cluster
                    </button>
                </div>
            </div>

            {/* Tabs */}
            <div className="flex gap-1 overflow-x-auto pb-1">
                {TABS.map(t => (
                    <button key={t.id} onClick={() => setTab(t.id)}
                        className={`flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium whitespace-nowrap transition-all ${tab === t.id ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-surface-200/50 hover:text-white hover:bg-white/5 border border-transparent'}`}>
                        <t.icon className="w-3.5 h-3.5" /> {t.label}
                    </button>
                ))}
            </div>

            {/* Content */}
            {tab === 'clusters' ? (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {clusters.map(c => (
                        <div key={c.id} className="glass-card rounded-xl p-5 group">
                            <div className="flex items-start justify-between mb-3">
                                <div className="flex items-center gap-3">
                                    <div className="p-2.5 rounded-lg bg-blue-500/10"><Server className="w-5 h-5 text-blue-400" /></div>
                                    <div>
                                        <h3 className="text-sm font-semibold text-white">{c.name}</h3>
                                        <span className={`text-[10px] px-1.5 py-0.5 rounded ${c.status === 'active' ? 'bg-green-500/10 text-green-400' : 'bg-yellow-500/10 text-yellow-400'}`}>{c.status}</span>
                                    </div>
                                </div>
                                <button onClick={() => handleDelete(c.id)} className="p-1.5 rounded-lg hover:bg-danger/10 text-surface-200/40 hover:text-danger transition-colors opacity-0 group-hover:opacity-100">
                                    <Trash2 className="w-3.5 h-3.5" />
                                </button>
                            </div>
                            <p className="text-xs text-surface-200/50">{c.context_name || 'default context'}</p>
                        </div>
                    ))}
                </div>
            ) : loading ? (
                <div className="flex items-center justify-center py-20"><Loader className="w-6 h-6 text-blue-500 animate-spin" /></div>
            ) : data.length === 0 ? (
                <div className="glass-card rounded-2xl flex flex-col items-center justify-center py-20 text-surface-200/40">
                    <Box className="w-12 h-12 mb-3 opacity-40" />
                    <p className="text-sm">No {tab} found{selectedNs ? ` in ${selectedNs}` : ''}</p>
                </div>
            ) : (
                <div className="glass-card rounded-xl overflow-hidden">
                    <table className="w-full text-xs">
                        <thead>
                            <tr className="border-b border-white/5">
                                {tab === 'events' ? (
                                    <><th className="text-left p-3 text-surface-200/40 font-medium">Type</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Reason</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Object</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Message</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Count</th></>
                                ) : tab === 'crds' ? (
                                    <><th className="text-left p-3 text-surface-200/40 font-medium">Name</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Group</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Kind</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Scope</th></>
                                ) : (
                                    <><th className="text-left p-3 text-surface-200/40 font-medium">Type</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Name</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Namespace</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Status</th>
                                        <th className="text-left p-3 text-surface-200/40 font-medium">Age</th></>
                                )}
                            </tr>
                        </thead>
                        <tbody>
                            {data.map((item: any, i: number) => (
                                <tr key={i} className="border-b border-white/[0.02] hover:bg-white/[0.02]">
                                    {tab === 'events' ? (
                                        <><td className="p-3"><span className={`px-1.5 py-0.5 rounded text-[10px] ${item.type === 'Warning' ? 'bg-yellow-500/10 text-yellow-400' : 'bg-green-500/10 text-green-400'}`}>{item.type}</span></td>
                                            <td className="p-3 text-white">{item.reason}</td>
                                            <td className="p-3 text-surface-200/60">{item.object}</td>
                                            <td className="p-3 text-surface-200/40 max-w-xs truncate">{item.message}</td>
                                            <td className="p-3 text-surface-200/40">{item.count}</td></>
                                    ) : tab === 'crds' ? (
                                        <><td className="p-3 text-white font-mono">{item.name}</td>
                                            <td className="p-3 text-surface-200/60">{item.group}</td>
                                            <td className="p-3 text-surface-200/60">{item.kind}</td>
                                            <td className="p-3 text-surface-200/40">{item.scope}</td></>
                                    ) : (
                                        <><td className="p-3"><span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${typeColor(item._type || item.kind || '')}`}>{item._type || item.kind || '-'}</span></td>
                                            <td className="p-3 text-white font-medium">{item.name}</td>
                                            <td className="p-3 text-surface-200/40">{item.namespace || '-'}</td>
                                            <td className="p-3">
                                                <span className={`px-1.5 py-0.5 rounded text-[10px] ${(item.status === 'Running' || item.status === 'Bound' || item.status === 'Active' || item.ready === item.replicas) ? 'bg-green-500/10 text-green-400' : (item.status === 'Pending' || item.status === 'ContainerCreating') ? 'bg-yellow-500/10 text-yellow-400' : 'bg-surface-200/10 text-surface-200/40'}`}>
                                                    {item.status || item.ready || (item.replicas !== undefined ? `${item.ready || 0}/${item.replicas}` : '-')}
                                                </span>
                                            </td>
                                            <td className="p-3 text-surface-200/50">{item.age || '-'}</td></>
                                    )}
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {/* Add Cluster Modal */}
            {showAdd && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowAdd(false)}>
                    <div className="glass-card rounded-2xl p-6 w-full max-w-lg mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-5">
                            <h3 className="text-lg font-semibold text-white">Add Kubernetes Cluster</h3>
                            <button onClick={() => setShowAdd(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        <div className="space-y-4">
                            <div>
                                <label className="text-xs text-surface-200/50 mb-1 block">Cluster Name</label>
                                <input value={addForm.name} onChange={e => setAddForm({ ...addForm, name: e.target.value })}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm" placeholder="production-cluster" />
                            </div>
                            <div>
                                <label className="text-xs text-surface-200/50 mb-1 block">Kubeconfig (YAML)</label>
                                <textarea value={addForm.kubeconfig} onChange={e => setAddForm({ ...addForm, kubeconfig: e.target.value })}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-xs font-mono h-48 custom-scrollbar"
                                    placeholder="apiVersion: v1&#10;kind: Config&#10;clusters:&#10;  - cluster:&#10;      server: https://..." />
                            </div>
                            <div className="flex gap-3">
                                <button onClick={() => setShowAdd(false)}
                                    className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5">Cancel</button>
                                <button onClick={handleAddCluster}
                                    className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-blue-600 to-blue-700 text-white text-sm font-medium hover:shadow-lg flex items-center justify-center gap-2">
                                    <Plus className="w-4 h-4" /> Add Cluster
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
