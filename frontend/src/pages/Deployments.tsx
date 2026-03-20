import { useState, useEffect } from 'react';
import { GitBranch, Rocket, CheckCircle, XCircle, Loader, Clock, ChevronDown } from 'lucide-react';
import { deployService } from '../services/deployments';

interface DeployEntry {
    id: string; app_id: string; branch: string; commit_hash: string;
    status: string; build_log: string; created_at: string; completed_at: string;
}

const statusConfig: Record<string, { icon: any; color: string }> = {
    success: { icon: CheckCircle, color: 'text-success' },
    failed: { icon: XCircle, color: 'text-danger' },
    running: { icon: Loader, color: 'text-info' },
    pending: { icon: Clock, color: 'text-warning' },
};

export default function Deployments() {
    const [deploys, setDeploys] = useState<DeployEntry[]>([]);
    const [loading, setLoading] = useState(true);
    const [expandedId, setExpandedId] = useState<string | null>(null);

    const fetchDeploys = async () => {
        try {
            const res = await deployService.list();
            setDeploys(Array.isArray(res) ? res : res?.data || []);
        } catch {
            setDeploys([]);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { fetchDeploys(); }, []);

    return (
        <div className="space-y-6 animate-fade-in">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Rocket className="w-7 h-7 text-nova-400" />
                        Deployments
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">Git-based CI/CD pipeline history</p>
                </div>
                <button className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all">
                    <Rocket className="w-4 h-4" /> Deploy Now
                </button>
            </div>

            {loading ? (
                <div className="flex items-center justify-center py-20"><Loader className="w-6 h-6 text-nova-500 animate-spin" /></div>
            ) : deploys.length === 0 ? (
                <div className="glass-card rounded-2xl flex flex-col items-center justify-center py-20 text-surface-200/40">
                    <Rocket className="w-10 h-10 mb-3 opacity-40" />
                    <p className="text-sm">No deployments yet</p>
                    <p className="text-xs mt-1">Trigger your first deployment to see it here</p>
                </div>
            ) : (
                <div className="space-y-3">
                    {deploys.map((d, i) => {
                        const { icon: StatusIcon, color } = statusConfig[d.status] || statusConfig.pending;
                        const isExpanded = expandedId === d.id;
                        return (
                            <div key={d.id} className="glass-card rounded-xl overflow-hidden hover:border-white/10 transition-all"
                                style={{ animationDelay: `${i * 60}ms` }}>
                                <div className="flex items-center justify-between p-4 cursor-pointer" onClick={() => setExpandedId(isExpanded ? null : d.id)}>
                                    <div className="flex items-center gap-4">
                                        <StatusIcon className={`w-5 h-5 ${color} ${d.status === 'running' ? 'animate-spin' : ''}`} />
                                        <div>
                                            <div className="flex items-center gap-2">
                                                <span className="text-white font-medium text-sm">{d.app_id}</span>
                                                <span className="text-xs px-2 py-0.5 rounded-full border border-white/10 text-surface-200/50 flex items-center gap-1">
                                                    <GitBranch className="w-3 h-3" /> {d.branch}
                                                </span>
                                            </div>
                                            <div className="flex items-center gap-3 mt-1">
                                                {d.commit_hash && <span className="text-xs font-mono text-surface-200/40">{d.commit_hash}</span>}
                                                <span className="text-xs text-surface-200/30">{d.created_at ? new Date(d.created_at).toLocaleString() : ''}</span>
                                            </div>
                                        </div>
                                    </div>
                                    <ChevronDown className={`w-4 h-4 text-surface-200/30 transition-transform ${isExpanded ? 'rotate-180' : ''}`} />
                                </div>
                                {isExpanded && (
                                    <div className="border-t border-white/5 p-4 bg-black/20">
                                        <pre className="text-xs text-surface-200/50 font-mono leading-5 whitespace-pre-wrap">
                                            {d.build_log || `[deploy] Deployment ${d.status}\n[deploy] Branch: ${d.branch}\n[deploy] No build log available`}
                                        </pre>
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
