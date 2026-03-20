import { useState, useEffect } from 'react';
import { Settings2, User, Lock, Bell, Save, Loader } from 'lucide-react';
import { settingsService } from '../services/billing';
import { useToast } from '../components/ui/ToastProvider';

export default function Settings() {
    const toast = useToast();
    const [activeTab, setActiveTab] = useState('profile');
    const [profile, setProfile] = useState({ name: '', email: '' });
    const [passwords, setPasswords] = useState({ current: '', new_pass: '', confirm: '' });
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [message, setMessage] = useState('');

    useEffect(() => {
        settingsService.getProfile()
            .then(data => setProfile({ name: data.name || '', email: data.email || '' }))
            .catch(() => setProfile({ name: 'Admin User', email: 'admin@novapanel.io' }))
            .finally(() => setLoading(false));
    }, []);

    const handleSaveProfile = async () => {
        setSaving(true);
        setMessage('');
        try {
            await settingsService.updateProfile(profile);
            setMessage('Profile updated successfully.');
            toast.success('Profile updated');
        } catch {
            setMessage('Failed to update profile.');
            toast.error('Failed to update profile');
        } finally {
            setSaving(false);
        }
    };

    const handleChangePassword = async () => {
        if (passwords.new_pass !== passwords.confirm) {
            setMessage('Passwords do not match.');
            return;
        }
        setSaving(true);
        setMessage('');
        try {
            await settingsService.updateProfile({ password: passwords.new_pass });
            setMessage('Password updated successfully.');
            toast.success('Password changed');
            setPasswords({ current: '', new_pass: '', confirm: '' });
        } catch {
            setMessage('Failed to change password.');
            toast.error('Failed to change password');
        } finally {
            setSaving(false);
        }
    };

    const tabs = [
        { id: 'profile', label: 'Profile', icon: User },
        { id: 'security', label: 'Security', icon: Lock },
        { id: 'notifications', label: 'Notifications', icon: Bell },
    ];

    return (
        <div className="space-y-6 animate-fade-in">
            <div>
                <h1 className="text-2xl font-bold text-white flex items-center gap-3">
                    <Settings2 className="w-7 h-7 text-nova-400" />
                    Settings
                </h1>
                <p className="text-surface-200/50 text-sm mt-1">Manage your account and preferences</p>
            </div>

            {message && (
                <div className="px-4 py-3 rounded-xl glass-input border-nova-500/30 text-sm text-nova-300 animate-fade-in">
                    {message}
                </div>
            )}

            <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
                {/* Sidebar Tabs */}
                <div className="glass-card rounded-2xl p-3 h-fit">
                    {tabs.map(tab => {
                        const Icon = tab.icon;
                        return (
                            <button key={tab.id} onClick={() => { setActiveTab(tab.id); setMessage(''); }}
                                className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-xl text-sm font-medium transition-all mb-1 ${activeTab === tab.id
                                    ? 'bg-nova-500/15 text-nova-400'
                                    : 'text-surface-200/50 hover:text-white hover:bg-white/5'
                                    }`}>
                                <Icon className="w-4 h-4" /> {tab.label}
                            </button>
                        );
                    })}
                </div>

                {/* Content */}
                <div className="lg:col-span-3">
                    {loading ? (
                        <div className="glass-card rounded-2xl p-6 flex items-center justify-center py-16">
                            <Loader className="w-6 h-6 text-nova-500 animate-spin" />
                        </div>
                    ) : activeTab === 'profile' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <h3 className="text-lg font-bold text-white">Profile Information</h3>
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Full Name</label>
                                    <input value={profile.name} onChange={e => setProfile({ ...profile, name: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Email</label>
                                    <input value={profile.email} onChange={e => setProfile({ ...profile, email: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" />
                                </div>
                            </div>
                            <button onClick={handleSaveProfile} disabled={saving}
                                className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-50">
                                {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />} Save Changes
                            </button>
                        </div>
                    ) : activeTab === 'security' ? (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <h3 className="text-lg font-bold text-white">Change Password</h3>
                            <div className="space-y-4 max-w-md">
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Current Password</label>
                                    <input type="password" value={passwords.current} onChange={e => setPasswords({ ...passwords, current: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" placeholder="••••••••" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">New Password</label>
                                    <input type="password" value={passwords.new_pass} onChange={e => setPasswords({ ...passwords, new_pass: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" placeholder="••••••••" />
                                </div>
                                <div>
                                    <label className="block text-sm font-medium text-surface-200 mb-1.5">Confirm Password</label>
                                    <input type="password" value={passwords.confirm} onChange={e => setPasswords({ ...passwords, confirm: e.target.value })}
                                        className="w-full px-4 py-2.5 rounded-xl glass-input text-white text-sm focus:outline-none focus:ring-2 focus:ring-nova-500/30" placeholder="••••••••" />
                                </div>
                            </div>
                            <button onClick={handleChangePassword} disabled={saving}
                                className="flex items-center gap-2 px-5 py-2.5 rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 text-white text-sm font-medium hover:shadow-lg hover:shadow-nova-500/25 transition-all disabled:opacity-50">
                                {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Lock className="w-4 h-4" />} Update Password
                            </button>
                        </div>
                    ) : (
                        <div className="glass-card rounded-2xl p-6 space-y-6">
                            <h3 className="text-lg font-bold text-white">Notification Preferences</h3>
                            <div className="space-y-4">
                                {['Deployment alerts', 'Security warnings', 'Backup notifications', 'Billing reminders', 'Server health alerts'].map((label, i) => (
                                    <div key={i} className="flex items-center justify-between py-2">
                                        <span className="text-sm text-surface-200/70">{label}</span>
                                        <button className={`w-11 h-6 rounded-full transition-colors relative ${i < 3 ? 'bg-nova-500' : 'bg-surface-700'}`}>
                                            <div className={`w-4 h-4 rounded-full bg-white absolute top-1 transition-all ${i < 3 ? 'left-6' : 'left-1'}`} />
                                        </button>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
