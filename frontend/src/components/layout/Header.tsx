import { Bell, Moon, Sun, Search, LogOut, User, Download } from 'lucide-react';
import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';
import { updateService } from '../../services/system';

export default function Header() {
    const [isDark, setIsDark] = useState(true);
    const [showUserMenu, setShowUserMenu] = useState(false);
    const [updateAvailable, setUpdateAvailable] = useState(false);
    const [latestVersion, setLatestVersion] = useState('');
    const { user, clearAuth } = useAuthStore();
    const userMenuRef = useRef<HTMLDivElement>(null);
    const navigate = useNavigate();

    useEffect(() => {
        if (user?.role === 'admin') {
            updateService.getStatus()
                .then(s => { setUpdateAvailable(s.update_available); setLatestVersion(s.latest_version); })
                .catch(() => { /* non-critical */ });
        }
    }, [user?.role]);

    const toggleTheme = () => {
        setIsDark(!isDark);
        document.documentElement.classList.toggle('dark');
    };

    // Close user menu when clicking outside
    useEffect(() => {
        const handler = (e: MouseEvent) => {
            if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
                setShowUserMenu(false);
            }
        };
        if (showUserMenu) {
            document.addEventListener('mousedown', handler);
        }
        return () => document.removeEventListener('mousedown', handler);
    }, [showUserMenu]);

    return (
        <header className="h-16 bg-surface-900/80 backdrop-blur-md border-b border-surface-700/50 flex items-center justify-between px-6 sticky top-0 z-40 flex-shrink-0">
            {/* Search */}
            <div className="relative w-96 max-w-[40vw]">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-surface-200/40" />
                <input
                    type="text"
                    placeholder="Search domains, servers, tasks..."
                    className="w-full pl-10 pr-12 py-2 bg-surface-800 border border-surface-700/50 rounded-lg text-sm text-white placeholder:text-surface-200/30 focus:outline-none focus:border-nova-500/50 focus:ring-1 focus:ring-nova-500/20 transition-all"
                />
                <kbd className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] text-surface-200/50 bg-surface-700/50 px-1.5 py-0.5 rounded">
                    ⌘K
                </kbd>
            </div>

            {/* Right actions */}
            <div className="flex items-center gap-3">
                {/* Update available badge */}
                {updateAvailable && (
                    <button
                        onClick={() => navigate('/settings?tab=updates')}
                        title={`Update available: ${latestVersion}`}
                        className="relative flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-nova-500/15 text-nova-400 text-xs font-medium hover:bg-nova-500/25 transition-colors border border-nova-500/20"
                    >
                        <Download className="w-3.5 h-3.5" />
                        <span className="hidden sm:inline">Update {latestVersion}</span>
                    </button>
                )}

                {/* Notifications */}
                <button className="relative p-2 rounded-lg text-surface-200/60 hover:text-white hover:bg-surface-700/50 transition-colors">
                    <Bell className="w-5 h-5" />
                    {updateAvailable && <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-nova-500 rounded-full animate-pulse" />}
                </button>

                {/* Theme toggle */}
                <button
                    onClick={toggleTheme}
                    className="p-2 rounded-lg text-surface-200/60 hover:text-white hover:bg-surface-700/50 transition-colors"
                >
                    {isDark ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
                </button>

                {/* User menu */}
                <div className="relative" ref={userMenuRef}>
                    <button
                        onClick={() => setShowUserMenu(!showUserMenu)}
                        className="flex items-center gap-3 pl-3 pr-2 py-1.5 rounded-lg hover:bg-surface-700/50 transition-colors"
                    >
                        <div className="w-8 h-8 rounded-full bg-gradient-to-br from-nova-500 to-nova-700 flex items-center justify-center text-white text-sm font-semibold flex-shrink-0">
                            {user?.first_name?.[0]?.toUpperCase() || 'N'}
                        </div>
                        <div className="text-left hidden md:block">
                            <p className="text-sm font-medium text-white">
                                {user?.first_name || 'Admin'} {user?.last_name || ''}
                            </p>
                            <p className="text-xs text-surface-200/40 capitalize">{user?.role || 'admin'}</p>
                        </div>
                    </button>

                    {showUserMenu && (
                        <div className="absolute right-0 top-full mt-2 w-56 bg-surface-800 border border-surface-700/50 rounded-xl shadow-2xl py-2 animate-fade-in z-[200]">
                            <div className="px-4 py-2 border-b border-surface-700/50">
                                <p className="text-sm text-white font-medium truncate">{user?.email || 'admin@novapanel.io'}</p>
                                <p className="text-xs text-surface-200/40 capitalize">{user?.role}</p>
                            </div>
                            <button className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-surface-200/70 hover:bg-surface-700/50 hover:text-white transition-colors">
                                <User className="w-4 h-4" />
                                Profile Settings
                            </button>
                            <button
                                onClick={clearAuth}
                                className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-danger hover:bg-danger/10 transition-colors"
                            >
                                <LogOut className="w-4 h-4" />
                                Sign Out
                            </button>
                        </div>
                    )}
                </div>
            </div>
        </header>
    );
}
