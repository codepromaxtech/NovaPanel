import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Server, Globe, Users, AppWindow, Clock, ArrowUpRight,
    Activity, Shield, Zap, TrendingUp, Rocket, Archive,
} from 'lucide-react';
import type { DashboardStats } from '../types';
import { serverService } from '../services/servers';
import api from '../services/api';

interface AuditEvent {
    id: string; action: string; resource_type: string; resource_name: string;
    created_at: string;
}

export default function Dashboard() {
    const navigate = useNavigate();
    const [stats, setStats] = useState<DashboardStats>({
        total_servers: 0, active_servers: 0, total_domains: 0,
        total_users: 0, total_apps: 0, pending_tasks: 0,
    });
    const [activity, setActivity] = useState<AuditEvent[]>([]);

    useEffect(() => {
        serverService.getDashboardStats()
            .then(setStats)
            .catch(() => setStats({
                total_servers: 0, active_servers: 0, total_domains: 0,
                total_users: 0, total_apps: 0, pending_tasks: 0,
            }));

        api.get('/audit-logs', { params: { per_page: 8 } })
            .then(res => setActivity(res.data?.data || []))
            .catch(() => setActivity([]));
    }, []);

    const actionIcons: Record<string, any> = {
        'domain.created': Globe, 'ssl.issued': Shield, 'deployment.success': Zap,
        'server.created': Server, 'backup.completed': Activity,
    };
    const actionColors: Record<string, string> = {
        'domain.created': 'text-success', 'ssl.issued': 'text-nova-400', 'deployment.success': 'text-warning',
        'server.created': 'text-info', 'backup.completed': 'text-success',
    };

    const timeAgo = (d: string) => {
        const ms = Date.now() - new Date(d).getTime();
        if (ms < 60000) return 'just now';
        if (ms < 3600000) return `${Math.floor(ms / 60000)} min ago`;
        if (ms < 86400000) return `${Math.floor(ms / 3600000)} hours ago`;
        return `${Math.floor(ms / 86400000)} days ago`;
    };

    const statCards = [
        { label: 'Total Servers', value: stats.total_servers, sub: `${stats.active_servers} active`, icon: Server, gradient: 'from-blue-500/20 to-cyan-500/20', iconColor: 'text-blue-400', borderColor: 'border-blue-500/20', path: '/servers' },
        { label: 'Domains', value: stats.total_domains, sub: 'managed', icon: Globe, gradient: 'from-nova-500/20 to-purple-500/20', iconColor: 'text-nova-400', borderColor: 'border-nova-500/20', path: '/domains' },
        { label: 'Users', value: stats.total_users, sub: 'accounts', icon: Users, gradient: 'from-emerald-500/20 to-teal-500/20', iconColor: 'text-emerald-400', borderColor: 'border-emerald-500/20', path: '/settings' },
        { label: 'Services', value: stats.total_apps, sub: `${stats.pending_tasks} pending tasks`, icon: AppWindow, gradient: 'from-amber-500/20 to-orange-500/20', iconColor: 'text-amber-400', borderColor: 'border-amber-500/20', path: '/monitoring' },
    ];

    const quickActions = [
        { label: 'Add Domain', icon: Globe, color: 'from-nova-600 to-nova-700', path: '/domains' },
        { label: 'Add Server', icon: Server, color: 'from-blue-600 to-blue-700', path: '/servers' },
        { label: 'Deploy App', icon: Rocket, color: 'from-amber-600 to-amber-700', path: '/deployments' },
        { label: 'Create Backup', icon: Archive, color: 'from-emerald-600 to-emerald-700', path: '/backups' },
    ];

    return (
        <div className="space-y-6">
            <div>
                <h1 className="text-2xl font-bold text-white">Dashboard</h1>
                <p className="text-surface-200/50 text-sm mt-1">Overview of your hosting infrastructure</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-5">
                {statCards.map((card, i) => (
                    <div key={card.label}
                        onClick={() => navigate(card.path)}
                        className={`relative overflow-hidden rounded-xl bg-gradient-to-br ${card.gradient} border ${card.borderColor} p-5 group hover:scale-[1.02] transition-all duration-300 cursor-pointer`}
                        style={{ animationDelay: `${i * 100}ms` }}>
                        <div className="flex items-start justify-between">
                            <div>
                                <p className="text-sm text-surface-200/50 font-medium">{card.label}</p>
                                <p className="text-3xl font-bold text-white mt-1">{card.value.toLocaleString()}</p>
                                <p className="text-xs text-surface-200/40 mt-1 flex items-center gap-1">
                                    <TrendingUp className="w-3 h-3 text-success" /> {card.sub}
                                </p>
                            </div>
                            <div className={`p-3 rounded-lg bg-surface-800/50 ${card.iconColor}`}>
                                <card.icon className="w-6 h-6" />
                            </div>
                        </div>
                        <div className="absolute inset-0 bg-gradient-to-r from-transparent to-white/[0.02] opacity-0 group-hover:opacity-100 transition-opacity" />
                    </div>
                ))}
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                {/* Recent Activity - from audit logs */}
                <div className="lg:col-span-2 bg-surface-800/50 border border-surface-700/50 rounded-xl p-5">
                    <div className="flex items-center justify-between mb-5">
                        <h2 className="text-lg font-semibold text-white">Recent Activity</h2>
                        <button className="text-xs text-nova-400 hover:text-nova-300 transition-colors flex items-center gap-1">
                            View all <ArrowUpRight className="w-3 h-3" />
                        </button>
                    </div>
                    <div className="space-y-3">
                        {activity.length === 0 ? (
                            <div className="flex flex-col items-center justify-center py-10 text-surface-200/30">
                                <Activity className="w-8 h-8 mb-2" />
                                <p className="text-sm">No recent activity</p>
                            </div>
                        ) : activity.map((item) => {
                            const Icon = actionIcons[item.action] || Activity;
                            const color = actionColors[item.action] || 'text-surface-200/40';
                            return (
                                <div key={item.id}
                                    className="flex items-center gap-4 p-3 rounded-lg hover:bg-surface-700/30 transition-colors group">
                                    <div className={`p-2 rounded-lg bg-surface-700/50 ${color}`}>
                                        <Icon className="w-4 h-4" />
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <p className="text-sm text-white font-medium">{item.action.replace('.', ' → ')}</p>
                                        <p className="text-xs text-surface-200/40 truncate">{item.resource_name}</p>
                                    </div>
                                    <span className="text-xs text-surface-200/30 flex items-center gap-1">
                                        <Clock className="w-3 h-3" /> {timeAgo(item.created_at)}
                                    </span>
                                </div>
                            );
                        })}
                    </div>
                </div>

                {/* Quick Actions - navigate to pages */}
                <div className="bg-surface-800/50 border border-surface-700/50 rounded-xl p-5">
                    <h2 className="text-lg font-semibold text-white mb-5">Quick Actions</h2>
                    <div className="space-y-3">
                        {quickActions.map((action) => (
                            <button key={action.label}
                                onClick={() => navigate(action.path)}
                                className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg bg-gradient-to-r ${action.color} text-white text-sm font-medium hover:opacity-90 transition-all hover:scale-[1.02] active:scale-[0.98] shadow-lg`}>
                                <action.icon className="w-5 h-5" />
                                {action.label}
                            </button>
                        ))}
                    </div>

                    <div className="mt-6 pt-5 border-t border-surface-700/50">
                        <h3 className="text-sm font-semibold text-surface-200/60 mb-3">Server Health</h3>
                        <div className="space-y-2">
                            {[
                                { label: 'CPU Usage', value: 42, gradient: 'from-success to-emerald-400' },
                                { label: 'Memory', value: 67, gradient: 'from-warning to-amber-400' },
                                { label: 'Disk', value: 28, gradient: 'from-info to-blue-400' },
                            ].map(m => (
                                <div key={m.label}>
                                    <div className="flex items-center justify-between text-sm">
                                        <span className="text-surface-200/50">{m.label}</span>
                                        <span className="text-white font-medium">{m.value}%</span>
                                    </div>
                                    <div className="w-full bg-surface-700/50 rounded-full h-2 mt-1">
                                        <div className={`bg-gradient-to-r ${m.gradient} h-2 rounded-full transition-all duration-500`} style={{ width: `${m.value}%` }} />
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
