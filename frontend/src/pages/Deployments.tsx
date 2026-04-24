import { useState, useEffect, useRef } from 'react';
import {
    Rocket, GitBranch, CheckCircle, XCircle, Loader, Clock,
    ChevronDown, Plus, X, Play, RotateCcw, ScrollText,
    Code2, Globe, Server, AlertCircle, Trash2, KeyRound, Eye, EyeOff, Lock,
} from 'lucide-react';
import { deployService } from '../services/deployments';
import { appService, Application } from '../services/applications';
import { serverService } from '../services/servers';
import api from '../services/api';
import { useToast } from '../components/ui/ToastProvider';

interface DeployEntry {
    id: string; app_id: string; branch: string; commit_hash: string;
    status: string; build_log: string; created_at: string; completed_at: string;
}

const statusConfig: Record<string, { icon: any; color: string; bg: string }> = {
    success: { icon: CheckCircle, color: 'text-green-400', bg: 'bg-green-500/10' },
    failed: { icon: XCircle, color: 'text-red-400', bg: 'bg-red-500/10' },
    running: { icon: Loader, color: 'text-blue-400', bg: 'bg-blue-500/10' },
    pending: { icon: Clock, color: 'text-yellow-400', bg: 'bg-yellow-500/10' },
};

const runtimeOptions = [
    { id: 'node', label: 'Node.js', icon: '🟩' },
    { id: 'python', label: 'Python', icon: '🐍' },
    { id: 'go', label: 'Go', icon: '🔵' },
    { id: 'php', label: 'PHP / Laravel', icon: '🐘' },
    { id: 'static', label: 'Static / HTML', icon: '📄' },
];

export default function Deployments() {
    const toast = useToast();
    const [deploys, setDeploys] = useState<DeployEntry[]>([]);
    const [apps, setApps] = useState<Application[]>([]);
    const [servers, setServers] = useState<any[]>([]);
    const [loading, setLoading] = useState(true);
    const [expandedId, setExpandedId] = useState<string | null>(null);

    // Create App modal
    const [showCreateApp, setShowCreateApp] = useState(false);
    const [appForm, setAppForm] = useState({
        name: '', server_id: '', runtime: 'node', git_repo: '', git_branch: 'main',
        deploy_method: 'git', app_type: 'web',
    });
    const [creatingApp, setCreatingApp] = useState(false);
    const [appError, setAppError] = useState('');

    // Deploy modal
    const [showDeploy, setShowDeploy] = useState(false);
    const [deployAppId, setDeployAppId] = useState('');
    const [deployBranch, setDeployBranch] = useState('main');
    const [deploying, setDeploying] = useState(false);

    // Env vars modal
    const [envApp, setEnvApp] = useState<Application | null>(null);
    const [envVars, setEnvVars] = useState<Record<string, string>>({});
    const [envRevealed, setEnvRevealed] = useState(false);
    const [envSaving, setEnvSaving] = useState(false);
    const [envLoading, setEnvLoading] = useState(false);
    const [newEnvKey, setNewEnvKey] = useState('');
    const [newEnvVal, setNewEnvVal] = useState('');

    // Live log modal
    const [logDeploy, setLogDeploy] = useState<string | null>(null);
    const [logContent, setLogContent] = useState('');
    const [logStatus, setLogStatus] = useState('');
    const logRef = useRef<HTMLPreElement>(null);
    const wsRef = useRef<WebSocket | null>(null);

    const fetchAll = async () => {
        setLoading(true);
        try {
            const [deployRes, appRes] = await Promise.all([
                deployService.list(),
                appService.list(),
            ]);
            setDeploys(Array.isArray(deployRes) ? deployRes : deployRes?.data || []);
            setApps(Array.isArray(appRes) ? appRes : appRes?.data || []);
        } catch {
            setDeploys([]);
            setApps([]);
        } finally {
            setLoading(false);
        }
    };

    const fetchServers = async () => {
        try {
            const res = await serverService.list();
            setServers(res?.data || []);
        } catch { setServers([]); }
    };

    useEffect(() => { fetchAll(); fetchServers(); }, []);

    const handleCreateApp = async () => {
        if (!appForm.name || !appForm.server_id) {
            setAppError('Name and server are required');
            return;
        }
        if (appForm.deploy_method === 'git' && !appForm.git_repo) {
            setAppError('Git repository URL is required for git deployments');
            return;
        }
        setCreatingApp(true);
        setAppError('');
        try {
            await appService.create(appForm);
            toast.success('Application created');
            setShowCreateApp(false);
            setAppForm({ name: '', server_id: '', runtime: 'node', git_repo: '', git_branch: 'main', deploy_method: 'git', app_type: 'web' });
            fetchAll();
        } catch (err: any) {
            setAppError(err?.response?.data?.error || 'Failed to create application');
        } finally {
            setCreatingApp(false);
        }
    };

    const handleDeploy = async () => {
        if (!deployAppId) return;
        setDeploying(true);
        try {
            const d = await deployService.create({ app_id: deployAppId, branch: deployBranch });
            toast.success('Deployment triggered');
            setShowDeploy(false);
            fetchAll();
            // Auto-open logs for the new deployment
            openLogs(d.id);
        } catch (err: any) {
            toast.error(err?.response?.data?.error || 'Deployment failed');
        } finally {
            setDeploying(false);
        }
    };

    const handleRedeploy = async (deployId: string) => {
        try {
            const d = await deployService.redeploy(deployId);
            toast.success('Redeployment triggered');
            fetchAll();
            openLogs(d.id);
        } catch (err: any) {
            toast.error(err?.response?.data?.error || 'Redeploy failed');
        }
    };

    const openEnvModal = async (app: Application) => {
        setEnvApp(app);
        setEnvRevealed(false);
        setEnvLoading(true);
        try {
            // Masked view first — from regular getById
            const res = await appService.getById(app.id);
            setEnvVars(res.env_vars || {});
        } catch { setEnvVars({}); }
        finally { setEnvLoading(false); }
    };

    const revealEnv = async () => {
        if (!envApp) return;
        setEnvLoading(true);
        try {
            const { data } = await api.get(`/apps/${envApp.id}/env`);
            setEnvVars(data.env_vars || {});
            setEnvRevealed(true);
        } catch { toast.error('Cannot reveal — admin only'); }
        finally { setEnvLoading(false); }
    };

    const saveEnv = async () => {
        if (!envApp) return;
        setEnvSaving(true);
        try {
            await api.put(`/apps/${envApp.id}/env`, { env_vars: envVars });
            toast.success('Environment variables saved (encrypted)');
            setEnvRevealed(false);
            openEnvModal(envApp);
        } catch (err: any) {
            toast.error(err.response?.data?.error || 'Failed to save env vars');
        } finally { setEnvSaving(false); }
    };

    const addEnvRow = () => {
        if (!newEnvKey.trim()) return;
        setEnvVars(v => ({ ...v, [newEnvKey.trim()]: newEnvVal }));
        setNewEnvKey(''); setNewEnvVal('');
    };

    const removeEnvRow = (k: string) => {
        setEnvVars(v => { const n = { ...v }; delete n[k]; return n; });
    };

    const handleDeleteApp = async (appId: string) => {
        if (!confirm('Delete this application and all its deployments? This cannot be undone.')) return;
        try {
            await appService.remove(appId);
            toast.success('Application deleted');
            fetchAll();
        } catch { toast.error('Failed to delete application'); }
    };

    const openLogs = (deployId: string) => {
        if (wsRef.current) wsRef.current.close();

        setLogDeploy(deployId);
        setLogContent('Connecting...');
        setLogStatus('pending');

        const token = localStorage.getItem('novapanel_token') || '';
        const base = import.meta.env.VITE_API_URL || '';
        const wsUrl = base.startsWith('http')
            ? base.replace(/^http/, 'ws') + `/deployments/${deployId}/ws?token=${token}`
            : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/deployments/${deployId}/ws?token=${token}`;

        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === 'log') {
                    setLogContent(prev => prev === 'Connecting...' ? msg.data : prev + msg.data);
                    setLogStatus(msg.status || 'running');
                } else if (msg.type === 'complete') {
                    setLogContent(msg.logs || '');
                    setLogStatus(msg.status);
                    fetchAll();
                } else if (msg.type === 'error') {
                    setLogContent(prev => prev + '\n[error] ' + msg.error);
                }
                setTimeout(() => {
                    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
                }, 30);
            } catch { /* ignore malformed */ }
        };

        ws.onerror = () => {
            setLogContent(prev => prev + '\n[ws] Connection error — retrying via HTTP...');
            // Fallback: fetch logs once via HTTP
            deployService.getLogs(deployId).then(res => {
                setLogContent(res.logs || 'No logs available');
                setLogStatus(res.status);
            }).catch(() => {});
        };

        ws.onclose = () => {
            wsRef.current = null;
        };
    };

    const closeLogs = () => {
        if (wsRef.current) { wsRef.current.close(); wsRef.current = null; }
        setLogDeploy(null);
    };

    const getAppName = (appId: string) => apps.find(a => a.id === appId)?.name || appId.slice(0, 8);
    const getAppRuntime = (appId: string) => apps.find(a => a.id === appId)?.runtime || '';

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Rocket className="w-7 h-7 text-nova-400" /> Deployments
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">
                        {apps.length} app{apps.length !== 1 ? 's' : ''} · {deploys.length} deployment{deploys.length !== 1 ? 's' : ''}
                    </p>
                </div>
                <div className="flex gap-2">
                    <button onClick={() => { setShowCreateApp(true); setAppError(''); }}
                        className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-white/10 text-surface-200/70 text-sm font-medium hover:bg-white/5 transition-all">
                        <Plus className="w-4 h-4" /> New App
                    </button>
                    <button onClick={() => { setShowDeploy(true); setDeployAppId(apps[0]?.id || ''); setDeployBranch('main'); }}
                        disabled={apps.length === 0}
                        className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-40">
                        <Rocket className="w-4 h-4" /> Deploy Now
                    </button>
                </div>
            </div>

            {/* ─── Applications Grid ─── */}
            {apps.length > 0 && (
                <div>
                    <h2 className="text-sm font-semibold text-surface-200/50 mb-3 flex items-center gap-2">
                        <Code2 className="w-4 h-4" /> Applications
                    </h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
                        {apps.map(app => (
                            <div key={app.id} className="glass-card rounded-xl p-4 group hover:border-surface-600/50 transition-all">
                                <div className="flex items-start justify-between mb-2">
                                    <div className="flex items-center gap-2">
                                        <span className="text-lg">{runtimeOptions.find(r => r.id === app.runtime)?.icon || '📦'}</span>
                                        <div>
                                            <h3 className="text-sm font-semibold text-white">{app.name}</h3>
                                            <p className="text-[10px] text-surface-200/40">{app.runtime} · {app.deploy_method}</p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-1">
                                        <button onClick={() => openEnvModal(app)}
                                            className="p-1.5 rounded-lg hover:bg-surface-600/50 text-surface-200/40 hover:text-surface-200 transition-colors opacity-0 group-hover:opacity-100"
                                            title="Environment Variables">
                                            <KeyRound className="w-3.5 h-3.5" />
                                        </button>
                                        <button onClick={() => { setShowDeploy(true); setDeployAppId(app.id); setDeployBranch(app.git_branch || 'main'); }}
                                            className="p-1.5 rounded-lg hover:bg-nova-500/10 text-surface-200/40 hover:text-nova-400 transition-colors opacity-0 group-hover:opacity-100"
                                            title="Deploy">
                                            <Play className="w-3.5 h-3.5" />
                                        </button>
                                        <button onClick={() => handleDeleteApp(app.id)}
                                            className="p-1.5 rounded-lg hover:bg-danger/10 text-surface-200/40 hover:text-danger transition-colors opacity-0 group-hover:opacity-100"
                                            title="Delete">
                                            <Trash2 className="w-3.5 h-3.5" />
                                        </button>
                                    </div>
                                </div>
                                <div className="flex items-center gap-3 text-xs text-surface-200/50">
                                    {app.git_repo && (
                                        <span className="flex items-center gap-1 truncate max-w-[60%]">
                                            <GitBranch className="w-3 h-3 flex-shrink-0" /> {app.git_repo.replace(/^https?:\/\/github\.com\//, '')}
                                        </span>
                                    )}
                                    <span className="flex items-center gap-1">
                                        <Globe className="w-3 h-3" /> {app.git_branch || 'main'}
                                    </span>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            )}

            {/* ─── Deployment History ─── */}
            {loading ? (
                <div className="flex items-center justify-center py-20"><Loader className="w-6 h-6 text-nova-400 animate-spin" /></div>
            ) : deploys.length === 0 && apps.length === 0 ? (
                <div className="glass-card rounded-2xl flex flex-col items-center justify-center py-20 text-surface-200/40">
                    <Rocket className="w-16 h-16 mb-4 opacity-40" />
                    <p className="text-lg font-medium mb-2">No applications yet</p>
                    <p className="text-sm mb-6">Create an application and deploy it from a git repository</p>
                    <button onClick={() => setShowCreateApp(true)}
                        className="px-6 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg transition-all flex items-center gap-2">
                        <Plus className="w-4 h-4" /> Create Application
                    </button>
                </div>
            ) : (
                <div>
                    <h2 className="text-sm font-semibold text-surface-200/50 mb-3 flex items-center gap-2">
                        <ScrollText className="w-4 h-4" /> Deployment History
                    </h2>
                    <div className="space-y-2">
                        {deploys.map((d, i) => {
                            const { icon: StatusIcon, color, bg } = statusConfig[d.status] || statusConfig.pending;
                            const isExpanded = expandedId === d.id;
                            return (
                                <div key={d.id} className="glass-card rounded-xl overflow-hidden hover:border-white/10 transition-all"
                                    style={{ animationDelay: `${i * 40}ms` }}>
                                    <div className="flex items-center justify-between p-4 cursor-pointer" onClick={() => setExpandedId(isExpanded ? null : d.id)}>
                                        <div className="flex items-center gap-4">
                                            <div className={`p-2 rounded-lg ${bg}`}>
                                                <StatusIcon className={`w-4 h-4 ${color} ${d.status === 'running' ? 'animate-spin' : ''}`} />
                                            </div>
                                            <div>
                                                <div className="flex items-center gap-2">
                                                    <span className="text-white font-medium text-sm">{getAppName(d.app_id)}</span>
                                                    <span className="text-xs px-2 py-0.5 rounded-full border border-white/10 text-surface-200/50 flex items-center gap-1">
                                                        <GitBranch className="w-3 h-3" /> {d.branch}
                                                    </span>
                                                    {d.commit_hash && (
                                                        <span className="text-xs font-mono text-surface-200/50">{d.commit_hash}</span>
                                                    )}
                                                </div>
                                                <div className="flex items-center gap-3 mt-0.5">
                                                    <span className={`text-xs font-medium ${color}`}>{d.status}</span>
                                                    <span className="text-[10px] text-surface-200/40">
                                                        {d.created_at ? new Date(d.created_at).toLocaleString() : ''}
                                                    </span>
                                                </div>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {d.status === 'running' && (
                                                <button onClick={e => { e.stopPropagation(); openLogs(d.id); }}
                                                    className="text-xs px-3 py-1.5 rounded-lg bg-blue-500/10 text-blue-400 hover:bg-blue-500/20 transition-colors flex items-center gap-1">
                                                    <ScrollText className="w-3 h-3" /> Live Logs
                                                </button>
                                            )}
                                            {(d.status === 'success' || d.status === 'failed') && (
                                                <button onClick={e => { e.stopPropagation(); handleRedeploy(d.id); }}
                                                    className="text-xs px-3 py-1.5 rounded-lg bg-nova-500/10 text-nova-400 hover:bg-nova-500/20 transition-colors flex items-center gap-1">
                                                    <RotateCcw className="w-3 h-3" /> Redeploy
                                                </button>
                                            )}
                                            <ChevronDown className={`w-4 h-4 text-surface-200/50 transition-transform ${isExpanded ? 'rotate-180' : ''}`} />
                                        </div>
                                    </div>
                                    {isExpanded && (
                                        <div className="border-t border-white/5 p-4 bg-black/20">
                                            <div className="flex justify-end mb-2">
                                                <button onClick={() => openLogs(d.id)}
                                                    className="text-[10px] px-2 py-1 rounded bg-white/5 text-surface-200/40 hover:text-white hover:bg-white/10 transition-colors flex items-center gap-1">
                                                    <ScrollText className="w-3 h-3" /> Full Logs
                                                </button>
                                            </div>
                                            <pre className="text-xs text-surface-200/50 font-mono leading-5 whitespace-pre-wrap max-h-48 overflow-y-auto custom-scrollbar">
                                                {d.build_log || `[deploy] Deployment ${d.status}\n[deploy] Branch: ${d.branch}\n[deploy] No build log available`}
                                            </pre>
                                        </div>
                                    )}
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}

            {/* ═══════ Create App Modal ═══════ */}
            {showCreateApp && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowCreateApp(false)}>
                    <div className="glass-card rounded-2xl p-8 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto custom-scrollbar animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-bold text-white flex items-center gap-2"><Code2 className="w-5 h-5 text-nova-400" /> New Application</h2>
                            <button onClick={() => setShowCreateApp(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>

                        {appError && (
                            <div className="mb-4 p-3 rounded-xl border border-danger/30 bg-danger/10 text-danger text-sm flex items-center gap-2">
                                <AlertCircle className="w-4 h-4 flex-shrink-0" /> {appError}
                            </div>
                        )}

                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Application Name</label>
                                <input value={appForm.name} onChange={e => setAppForm(p => ({ ...p, name: e.target.value }))}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                    placeholder="my-web-app" />
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Target Server</label>
                                <select value={appForm.server_id} onChange={e => setAppForm(p => ({ ...p, server_id: e.target.value }))}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                    <option value="">Select a server...</option>
                                    {servers.map(s => (
                                        <option key={s.id} value={s.id}>{s.name} ({s.ip_address})</option>
                                    ))}
                                </select>
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Runtime</label>
                                <div className="grid grid-cols-5 gap-2">
                                    {runtimeOptions.map(r => (
                                        <button key={r.id} type="button" onClick={() => setAppForm(p => ({ ...p, runtime: r.id }))}
                                            className={`flex flex-col items-center gap-1 p-2.5 rounded-xl text-xs font-medium transition-all ${appForm.runtime === r.id ? 'bg-nova-500/20 text-nova-400 border border-nova-500/30' : 'bg-white/5 text-surface-200/50 border border-white/10 hover:bg-white/10'}`}>
                                            <span className="text-lg">{r.icon}</span>
                                            <span>{r.label}</span>
                                        </button>
                                    ))}
                                </div>
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Git Repository URL</label>
                                <input value={appForm.git_repo} onChange={e => setAppForm(p => ({ ...p, git_repo: e.target.value }))}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                    placeholder="https://github.com/username/repo.git" />
                            </div>

                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Branch</label>
                                <input value={appForm.git_branch} onChange={e => setAppForm(p => ({ ...p, git_branch: e.target.value }))}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                    placeholder="main" />
                            </div>

                            <div className="flex gap-3 pt-2">
                                <button onClick={() => setShowCreateApp(false)}
                                    className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                                    Cancel
                                </button>
                                <button onClick={handleCreateApp} disabled={creatingApp}
                                    className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all flex items-center justify-center gap-2 disabled:opacity-50">
                                    {creatingApp ? <Loader className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                                    {creatingApp ? 'Creating...' : 'Create Application'}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* ═══════ Deploy Modal ═══════ */}
            {showDeploy && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={() => setShowDeploy(false)}>
                    <div className="glass-card rounded-2xl p-8 w-full max-w-md mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-6">
                            <h2 className="text-xl font-bold text-white flex items-center gap-2"><Rocket className="w-5 h-5 text-nova-400" /> Trigger Deployment</h2>
                            <button onClick={() => setShowDeploy(false)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        <div className="space-y-4">
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Application</label>
                                <select value={deployAppId} onChange={e => {
                                    setDeployAppId(e.target.value);
                                    const app = apps.find(a => a.id === e.target.value);
                                    if (app) setDeployBranch(app.git_branch || 'main');
                                }}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 bg-transparent">
                                    {apps.map(a => (
                                        <option key={a.id} value={a.id}>{a.name} ({a.runtime})</option>
                                    ))}
                                </select>
                            </div>
                            <div>
                                <label className="block text-sm font-medium text-surface-200 mb-1.5">Branch</label>
                                <input value={deployBranch} onChange={e => setDeployBranch(e.target.value)}
                                    className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30 placeholder:text-surface-200/20"
                                    placeholder="main" />
                            </div>
                            <div className="flex gap-3 pt-2">
                                <button onClick={() => setShowDeploy(false)}
                                    className="flex-1 py-2.5 rounded-xl border border-white/10 text-surface-200/60 text-sm hover:bg-white/5 transition-colors">
                                    Cancel
                                </button>
                                <button onClick={handleDeploy} disabled={deploying || !deployAppId}
                                    className="flex-1 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all flex items-center justify-center gap-2 disabled:opacity-50">
                                    {deploying ? <Loader className="w-4 h-4 animate-spin" /> : <Rocket className="w-4 h-4" />}
                                    {deploying ? 'Deploying...' : 'Deploy'}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {/* ═══════ Live Log Viewer Modal ═══════ */}
            {logDeploy && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={closeLogs}>
                    <div className="glass-card rounded-2xl p-6 w-full max-w-2xl mx-4 animate-fade-in" onClick={e => e.stopPropagation()}>
                        <div className="flex items-center justify-between mb-4">
                            <h3 className="text-lg font-semibold text-white flex items-center gap-2">
                                <ScrollText className="w-5 h-5 text-nova-400" />
                                Build Log
                                {logStatus === 'running' && <span className="text-xs px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 flex items-center gap-1"><Loader className="w-3 h-3 animate-spin" /> Live</span>}
                                {logStatus === 'success' && <span className="text-xs px-2 py-0.5 rounded-full bg-green-500/10 text-green-400">✓ Success</span>}
                                {logStatus === 'failed' && <span className="text-xs px-2 py-0.5 rounded-full bg-red-500/10 text-red-400">✗ Failed</span>}
                            </h3>
                            <button onClick={closeLogs} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>
                        <pre ref={logRef} className="bg-[#0d1117] rounded-xl p-4 text-xs text-green-300 font-mono leading-5 whitespace-pre-wrap max-h-[60vh] overflow-y-auto custom-scrollbar border border-white/5">
                            {logContent || 'Waiting for logs...'}
                        </pre>
                    </div>
                </div>
            )}

            {/* ═══════ Env Vars Modal ═══════ */}
            {envApp && (
                <div className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm">
                    <div className="glass-card rounded-2xl p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto animate-fade-in">
                        <div className="flex items-center justify-between mb-5">
                            <h3 className="text-lg font-semibold text-white flex items-center gap-2">
                                <KeyRound className="w-5 h-5 text-nova-400" /> Env Vars — {envApp.name}
                                <span title="Encrypted at rest"><Lock className="w-3.5 h-3.5 text-nova-400/60" /></span>
                            </h3>
                            <button onClick={() => setEnvApp(null)} className="text-surface-200/40 hover:text-white"><X className="w-5 h-5" /></button>
                        </div>

                        {envLoading ? (
                            <div className="flex items-center justify-center py-8"><Loader className="w-5 h-5 text-nova-400 animate-spin" /></div>
                        ) : (
                            <>
                                {/* Key-value list */}
                                <div className="space-y-2 mb-4 max-h-64 overflow-y-auto">
                                    {Object.keys(envVars).length === 0 ? (
                                        <p className="text-xs text-surface-400 text-center py-4">No environment variables</p>
                                    ) : Object.entries(envVars).map(([k, v]) => (
                                        <div key={k} className="flex items-center gap-2">
                                            <span className="font-mono text-xs text-nova-300 w-2/5 truncate bg-surface-800/60 px-2.5 py-2 rounded-lg">{k}</span>
                                            <span className={`font-mono text-xs flex-1 truncate px-2.5 py-2 rounded-lg ${envRevealed ? 'bg-surface-800/60 text-surface-200' : 'bg-surface-800/40 text-surface-500 tracking-widest'}`}>
                                                {envRevealed ? v : '●●●●●●●●'}
                                            </span>
                                            {envRevealed && (
                                                <button onClick={() => removeEnvRow(k)}
                                                    className="p-1.5 rounded text-surface-400 hover:text-danger hover:bg-danger/10 transition-colors flex-shrink-0">
                                                    <X className="w-3.5 h-3.5" />
                                                </button>
                                            )}
                                        </div>
                                    ))}
                                </div>

                                {/* Add new var (only in revealed/edit mode) */}
                                {envRevealed && (
                                    <div className="flex items-center gap-2 mb-4 p-3 rounded-xl bg-surface-800/40 border border-surface-700/40">
                                        <input value={newEnvKey} onChange={e => setNewEnvKey(e.target.value)}
                                            placeholder="KEY" className="font-mono text-xs px-2.5 py-2 rounded-lg bg-surface-700/60 border border-surface-600/50 text-nova-300 w-2/5 focus:outline-none focus:border-nova-500/50" />
                                        <input value={newEnvVal} onChange={e => setNewEnvVal(e.target.value)}
                                            placeholder="value" className="font-mono text-xs px-2.5 py-2 rounded-lg bg-surface-700/60 border border-surface-600/50 text-surface-200 flex-1 focus:outline-none focus:border-nova-500/50" />
                                        <button onClick={addEnvRow} disabled={!newEnvKey.trim()}
                                            className="p-1.5 rounded-lg bg-nova-600 text-white disabled:opacity-50 hover:bg-nova-500 transition-colors flex-shrink-0">
                                            <Plus className="w-3.5 h-3.5" />
                                        </button>
                                    </div>
                                )}

                                <div className="flex gap-2 pt-2 border-t border-surface-700/40">
                                    {!envRevealed ? (
                                        <button onClick={revealEnv}
                                            className="flex-1 py-2 rounded-xl border border-surface-700/50 text-surface-300 text-sm hover:bg-surface-700/30 transition-colors flex items-center justify-center gap-2">
                                            <Eye className="w-4 h-4" /> Reveal & Edit
                                        </button>
                                    ) : (
                                        <>
                                            <button onClick={() => { setEnvRevealed(false); openEnvModal(envApp); }}
                                                className="px-4 py-2 rounded-xl border border-surface-700/50 text-surface-300 text-sm hover:bg-surface-700/30 transition-colors flex items-center gap-2">
                                                <EyeOff className="w-4 h-4" /> Hide
                                            </button>
                                            <button onClick={saveEnv} disabled={envSaving}
                                                className="flex-1 py-2 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium disabled:opacity-60 flex items-center justify-center gap-2 hover:shadow-lg transition-all">
                                                {envSaving ? <Loader className="w-4 h-4 animate-spin" /> : <Lock className="w-4 h-4" />}
                                                Save Encrypted
                                            </button>
                                        </>
                                    )}
                                </div>
                                <p className="text-[10px] text-surface-500 text-center mt-2">Values are AES-encrypted in the database. Reveal requires admin role.</p>
                            </>
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}
