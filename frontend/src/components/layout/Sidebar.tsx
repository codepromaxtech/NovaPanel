import { NavLink } from 'react-router-dom';
import {
    LayoutDashboard, Globe, Server, Database, Mail,
    FolderOpen, Shield, Archive, Activity, Rocket,
    Settings, ChevronLeft, ChevronRight, Zap, CreditCard, Box, Layers, ShieldCheck, ArrowRightLeft, Clock, Cog, Cloud, Users,
} from 'lucide-react';

interface SidebarProps {
    collapsed: boolean;
    onToggle: () => void;
}

const navItems = [
    { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
    { to: '/domains', icon: Globe, label: 'Domains' },
    { to: '/servers', icon: Server, label: 'Servers' },
    { to: '/databases', icon: Database, label: 'Databases' },
    { to: '/email', icon: Mail, label: 'Email' },
    { to: '/files', icon: FolderOpen, label: 'Files' },
    { to: '/transfers', icon: ArrowRightLeft, label: 'Transfers' },
    { to: '/backups', icon: Archive, label: 'Backups' },
    { to: '/cron', icon: Clock, label: 'Cron Jobs' },
    { to: '/services', icon: Cog, label: 'Services' },
    { to: '/monitoring', icon: Activity, label: 'Monitoring' },
    { to: '/docker', icon: Box, label: 'Docker' },
    { to: '/kubernetes', icon: Layers, label: 'Kubernetes' },
    { to: '/deployments', icon: Rocket, label: 'Deployments' },
    { to: '/security', icon: Shield, label: 'Security' },
    { to: '/waf', icon: ShieldCheck, label: 'WAF' },
    { to: '/billing', icon: CreditCard, label: 'Billing' },
    { to: '/cloudflare', icon: Cloud, label: 'Cloudflare' },
    { to: '/team', icon: Users, label: 'Team' },
    { to: '/settings', icon: Settings, label: 'Settings' },
];

export default function Sidebar({ collapsed, onToggle }: SidebarProps) {
    return (
        <aside
            className={`fixed left-0 top-0 h-screen bg-surface-900 border-r border-surface-700/50 flex flex-col transition-all duration-300 z-50 ${collapsed ? 'w-[68px]' : 'w-[250px]'}`}
        >
            {/* Logo */}
            <div className="flex items-center gap-3 px-4 h-16 border-b border-surface-700/50 flex-shrink-0">
                <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-nova-500 to-nova-700 flex items-center justify-center flex-shrink-0">
                    <Zap className="w-5 h-5 text-white" />
                </div>
                {!collapsed && (
                    <span className="text-lg font-bold gradient-text tracking-tight whitespace-nowrap overflow-hidden">NovaPanel</span>
                )}
            </div>

            {/* Navigation */}
            <nav className="flex-1 py-4 px-3 space-y-0.5 overflow-y-auto overflow-x-hidden">
                {navItems.map((item) => (
                    <NavLink
                        key={item.to}
                        to={item.to}
                        end={item.to === '/'}
                        title={collapsed ? item.label : undefined}
                        className={({ isActive }) =>
                            `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 group ${isActive
                                ? 'bg-nova-600/20 text-nova-400 shadow-sm'
                                : 'text-surface-200/70 hover:bg-surface-700/50 hover:text-white'
                            }`
                        }
                    >
                        <item.icon className="w-5 h-5 flex-shrink-0 transition-transform group-hover:scale-110" />
                        {!collapsed && <span className="truncate">{item.label}</span>}
                    </NavLink>
                ))}
            </nav>

            {/* Collapse Toggle */}
            <button
                onClick={onToggle}
                className="flex items-center justify-center h-12 flex-shrink-0 border-t border-surface-700/50 text-surface-200/50 hover:text-white hover:bg-surface-700/30 transition-colors"
                title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            >
                {collapsed ? <ChevronRight className="w-5 h-5" /> : <ChevronLeft className="w-5 h-5" />}
            </button>
        </aside>
    );
}
