import { useState, useEffect } from 'react';
import {
    Activity, Cpu, HardDrive, Wifi, Server, RefreshCw, Loader,
    MemoryStick, Clock, CheckCircle, XCircle, Database, Globe,
    Search, MessageSquare, Zap
} from 'lucide-react';
import { monitoringService } from '../services/monitoring';
import { serverService } from '../services/servers';

interface Metrics {
    cpu_percent: number;
    ram_used_mb: number;
    ram_total_mb: number;
    ram_percent: number;
    disk_used_gb: number;
    disk_total_gb: number;
    disk_percent: number;
    load_avg_1: number;
    load_avg_5: number;
    load_avg_15: number;
    network_rx_mb: number;
    network_tx_mb: number;
    uptime: string;
    process_count: number;
}

interface ServiceStatus {
    name: string;
    engine: string;
    type: string;
    status: string;
    port: number;
}

function GaugeCard({ label, value, max, unit, icon: Icon, color }: {
    label: string; value: number; max?: number; unit: string;
    icon: React.ElementType; color: string;
}) {
    const pct = max ? (value / max) * 100 : value;
    const displayVal = max ? `${value.toFixed(0)} / ${max.toFixed(0)}` : `${value.toFixed(1)}`;
    const barColor = pct > 80 ? 'from-red-500 to-red-600' : pct > 60 ? 'from-amber-500 to-amber-600' : color;

    return (
        <div className="glass-card rounded-xl p-5">
            <div className="flex items-center gap-2 mb-3">
                <div className={`p-2 rounded-lg bg-gradient-to-br ${color} bg-opacity-10`} style={{ background: `linear-gradient(135deg, ${color.includes('emerald') ? 'rgba(16,185,129,0.1)' : color.includes('blue') ? 'rgba(59,130,246,0.1)' : color.includes('violet') ? 'rgba(139,92,246,0.1)' : 'rgba(245,158,11,0.1)'} 0%, transparent 100%)` }}>
                    <Icon className="w-4 h-4 text-white/70" />
                </div>
                <span className="text-xs text-surface-200/50 font-medium uppercase tracking-wider">{label}</span>
            </div>
            <div className="flex items-baseline gap-1.5 mb-3">
                <span className="text-2xl font-bold text-white">{displayVal}</span>
                <span className="text-xs text-surface-200/40">{unit}</span>
            </div>
            <div className="w-full bg-white/5 rounded-full h-2">
                <div className={`h-2 rounded-full bg-gradient-to-r ${barColor} transition-all duration-700`}
                    style={{ width: `${Math.min(pct, 100)}%` }} />
            </div>
            <div className="text-right mt-1">
                <span className="text-xs text-surface-200/30">{pct.toFixed(1)}%</span>
            </div>
        </div>
    );
}

const typeIcons: Record<string, React.ElementType> = {
    database: Database,
    cache: Zap,
    webserver: Globe,
    search: Search,
    queue: MessageSquare,
    app: Server,
};

export default function Monitoring() {
    const [metrics, setMetrics] = useState<Metrics | null>(null);
    const [services, setServices] = useState<ServiceStatus[]>([]);
    const [serverId, setServerId] = useState('');
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);

    const loadData = async (showRefresh = false) => {
        if (showRefresh) setRefreshing(true);
        else setLoading(true);

        try {
            // Get the first server (master)
            let sid = serverId;
            if (!sid) {
                const serversResp = await serverService.list();
                const servers = serversResp.data || [];
                const master = servers.find((s: { role: string }) => s.role === 'master') || servers[0];
                if (master) {
                    sid = master.id;
                    setServerId(sid);
                }
            }

            if (sid) {
                const [metricsData, servicesData] = await Promise.all([
                    monitoringService.getLiveMetrics(sid),
                    monitoringService.getServices(sid),
                ]);
                setMetrics(metricsData);
                setServices(servicesData.services || []);
            }
        } catch (err) {
            console.error('Failed to load monitoring data:', err);
        } finally {
            setLoading(false);
            setRefreshing(false);
        }
    };

    useEffect(() => {
        loadData();
        const interval = setInterval(() => loadData(true), 30000);
        return () => clearInterval(interval);
    }, []);

    if (loading) {
        return (
            <div className="flex items-center justify-center py-20">
                <Loader className="w-6 h-6 text-nova-500 animate-spin" />
            </div>
        );
    }

    const runningCount = services.filter(s => s.status === 'running').length;
    const stoppedCount = services.filter(s => s.status !== 'running').length;

    return (
        <div className="space-y-6 animate-fade-in">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                        <Activity className="w-7 h-7 text-nova-400" />
                        Monitoring
                    </h1>
                    <p className="text-surface-200/50 text-sm mt-1">
                        Real-time server resources & service health
                    </p>
                </div>
                <button onClick={() => loadData(true)} disabled={refreshing}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-white/5 text-white/60 text-sm font-medium hover:bg-white/10 transition-all">
                    <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
                    Refresh
                </button>
            </div>

            {/* Summary Strip */}
            {metrics && (
                <div className="glass-card rounded-xl p-4 flex items-center gap-6 flex-wrap">
                    <div className="flex items-center gap-2">
                        <Clock className="w-4 h-4 text-surface-200/40" />
                        <span className="text-sm text-surface-200/60">Uptime: <span className="text-white font-medium">{metrics.uptime}</span></span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Activity className="w-4 h-4 text-surface-200/40" />
                        <span className="text-sm text-surface-200/60">Load: <span className="text-white font-medium">{metrics.load_avg_1} / {metrics.load_avg_5} / {metrics.load_avg_15}</span></span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Server className="w-4 h-4 text-surface-200/40" />
                        <span className="text-sm text-surface-200/60">Processes: <span className="text-white font-medium">{metrics.process_count}</span></span>
                    </div>
                    <div className="flex items-center gap-2">
                        <Wifi className="w-4 h-4 text-surface-200/40" />
                        <span className="text-sm text-surface-200/60">
                            Net: <span className="text-emerald-400 font-medium">↓{metrics.network_rx_mb.toFixed(1)}MB</span>
                            {' / '}
                            <span className="text-blue-400 font-medium">↑{metrics.network_tx_mb.toFixed(1)}MB</span>
                        </span>
                    </div>
                </div>
            )}

            {/* Resource Gauges */}
            {metrics && (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                    <GaugeCard label="CPU" value={metrics.cpu_percent} unit="%" icon={Cpu}
                        color="from-emerald-500 to-emerald-600" />
                    <GaugeCard label="Memory" value={metrics.ram_used_mb} max={metrics.ram_total_mb}
                        unit="MB" icon={MemoryStick} color="from-blue-500 to-blue-600" />
                    <GaugeCard label="Disk" value={metrics.disk_used_gb} max={metrics.disk_total_gb}
                        unit="GB" icon={HardDrive} color="from-violet-500 to-violet-600" />
                    <GaugeCard label="Load (1m)" value={metrics.load_avg_1} unit="avg" icon={Activity}
                        color="from-amber-500 to-amber-600" />
                </div>
            )}

            {/* Services Health */}
            <div className="glass-card rounded-xl p-5">
                <div className="flex items-center justify-between mb-4">
                    <h2 className="text-lg font-semibold text-white flex items-center gap-2">
                        <Zap className="w-5 h-5 text-nova-400" />
                        Service Health
                    </h2>
                    <div className="flex items-center gap-3 text-xs">
                        <span className="flex items-center gap-1 text-emerald-400">
                            <CheckCircle className="w-3.5 h-3.5" /> {runningCount} Running
                        </span>
                        <span className="flex items-center gap-1 text-red-400">
                            <XCircle className="w-3.5 h-3.5" /> {stoppedCount} Stopped
                        </span>
                    </div>
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                    {services.map((svc, i) => {
                        const SvcIcon = typeIcons[svc.type] || Server;
                        const isRunning = svc.status === 'running';
                        return (
                            <div key={i}
                                className={`flex items-center gap-3 p-3 rounded-lg border transition-all
                                    ${isRunning ? 'bg-emerald-500/5 border-emerald-500/20' : 'bg-red-500/5 border-red-500/20'}`}>
                                <div className={`p-2 rounded-lg ${isRunning ? 'bg-emerald-500/10' : 'bg-red-500/10'}`}>
                                    <SvcIcon className={`w-4 h-4 ${isRunning ? 'text-emerald-400' : 'text-red-400'}`} />
                                </div>
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium text-white truncate">{svc.name}</p>
                                    <p className="text-xs text-surface-200/40">
                                        {svc.engine}{svc.port > 0 ? ` :${svc.port}` : ''}
                                    </p>
                                </div>
                                <div className={`w-2 h-2 rounded-full ${isRunning ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'}`} />
                            </div>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}
