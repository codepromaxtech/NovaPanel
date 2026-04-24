import { useState, useEffect, FormEvent } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { Zap, Mail, Lock, User, ArrowRight, Eye, EyeOff, Shield, Server, Activity, Globe, Check } from 'lucide-react';
import { authService } from '../services/auth';
import { useAuthStore } from '../store/authStore';
import axios from 'axios';

export default function Login() {
    const [isLogin, setIsLogin] = useState(true);
    const [mode, setMode] = useState<'auth' | 'forgot' | 'totp'>('auth');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [firstName, setFirstName] = useState('');
    const [lastName, setLastName] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [forgotSent, setForgotSent] = useState(false);
    const [totpCode, setTotpCode] = useState('');
    const navigate = useNavigate();
    const { setAuth } = useAuthStore();

    const [buildInfo, setBuildInfo] = useState<{
        version: string;
        plan_type: string;
        license_valid: boolean;
        days_left: number;
    }>({ version: '...', plan_type: 'community', license_valid: false, days_left: -1 });

    useEffect(() => {
        axios.get('/health').then(r => {
            setBuildInfo({
                version: r.data.version || 'unknown',
                plan_type: r.data.plan_type || 'community',
                license_valid: r.data.license_valid ?? false,
                days_left: r.data.days_left ?? -1,
            });
        }).catch(() => {});
    }, []);

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);
        try {
            let res;
            if (isLogin) {
                res = await authService.login(email, password);
            } else {
                res = await authService.register(email, password, firstName, lastName);
            }
            if ((res as any).two_factor_required) {
                setMode('totp');
                setLoading(false);
                return;
            }
            setAuth(res.user, res.token);
            navigate('/');
        } catch (err: any) {
            setError(err.response?.data?.error || 'Authentication failed. Please check your credentials.');
        } finally {
            setLoading(false);
        }
    };

    const handleForgot = async (e: FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);
        try {
            await authService.forgotPassword(email);
            setForgotSent(true);
        } catch (err: any) {
            setError(err.response?.data?.error || 'Failed to send reset email.');
        } finally {
            setLoading(false);
        }
    };

    const handleTotpVerify = async (e: FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);
        try {
            const res = await authService.totpVerify(email, totpCode);
            setAuth(res.user, res.token);
            navigate('/');
        } catch (err: any) {
            setError(err.response?.data?.error || 'Invalid code.');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="flex min-h-screen bg-surface-950 text-surface-50 font-sans selection:bg-nova-500/30">
            {/* Left side: Premium Branding & Atmosphere */}
            <div className="hidden lg:flex lg:w-[55%] relative overflow-hidden flex-col justify-between p-12 border-r border-white/5">
                {/* Background layers */}
                <div className="absolute inset-0 bg-grid-pattern opacity-40 mix-blend-overlay pointer-events-none" />
                <div className="absolute inset-0 bg-aurora opacity-70 pointer-events-none" />
                <div className="absolute bottom-0 left-0 right-0 h-1/2 bg-gradient-to-t from-surface-950 to-transparent pointer-events-none" />

                {/* Branding Top */}
                <div className="relative z-10 animate-fade-in flex items-center gap-4">
                    <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-nova-500 to-nova-700 flex items-center justify-center shadow-xl shadow-nova-500/30 ring-1 ring-white/20">
                        <Zap className="w-8 h-8 text-white drop-shadow-md" />
                    </div>
                    <h1 className="text-4xl font-bold tracking-tight text-white drop-shadow-lg">
                        Nova<span className="text-nova-400">Panel</span>
                    </h1>
                </div>

                {/* Center Content */}
                <div className="relative z-10 max-w-xl animate-fade-in delay-150 mt-12 mb-auto pb-20">
                    <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full glass-panel border border-nova-500/30 mb-8 backdrop-blur-md">
                        <span className={`w-2 h-2 rounded-full animate-pulse-ring ${buildInfo.license_valid ? 'bg-nova-400' : 'bg-yellow-400'}`} />
                        <span className="text-xs font-semibold uppercase tracking-wider text-nova-300">
                            NovaPanel {buildInfo.version}
                            {' '}
                            {buildInfo.plan_type === 'community' ? 'Community' :
                             buildInfo.plan_type === 'trial' ? `Trial${buildInfo.days_left > 0 ? ` · ${buildInfo.days_left}d left` : ''}` :
                             buildInfo.plan_type === 'reseller' ? 'Reseller' : 'Enterprise'}
                        </span>
                    </div>

                    <h2 className="text-6xl font-extrabold tracking-tight mb-6 leading-[1.15] text-white">
                        The Operating System for your <span className="text-gradient">Cloud Infrastructure</span>.
                    </h2>

                    <p className="text-xl text-surface-200/80 leading-relaxed max-w-lg mb-12 font-light">
                        Deploy, secure, and scale applications seamlessly across any server. Agent-based architecture built for the modern DevOps era.
                    </p>

                    {/* Feature badges */}
                    <div className="grid grid-cols-2 gap-4">
                        <div className="flex items-center gap-3 p-3 rounded-xl glass-panel animate-float">
                            <Server className="w-6 h-6 text-nova-400" />
                            <span className="text-sm font-medium text-surface-200">Multi-Server Sync</span>
                        </div>
                        <div className="flex items-center gap-3 p-3 rounded-xl glass-panel animate-float-delayed">
                            <Shield className="w-6 h-6 text-success" />
                            <span className="text-sm font-medium text-surface-200">Zero-Trust RBAC</span>
                        </div>
                        <div className="flex items-center gap-3 p-3 rounded-xl glass-panel animate-float delay-150">
                            <Globe className="w-6 h-6 text-info" />
                            <span className="text-sm font-medium text-surface-200">Automated SSL & DNS</span>
                        </div>
                        <div className="flex items-center gap-3 p-3 rounded-xl glass-panel animate-float-delayed delay-150">
                            <Activity className="w-6 h-6 text-warning" />
                            <span className="text-sm font-medium text-surface-200">Live Telemetry</span>
                        </div>
                    </div>
                </div>

                {/* Bottom stats/social proof */}
                <div className="relative z-10 flex items-center justify-between pt-8 border-t border-white/10 mt-auto animate-fade-in delay-300">
                    <p className="text-sm text-surface-200/50">© {new Date().getFullYear()} NovaPanel · {buildInfo.version}</p>
                    <div className="flex flex-col items-end">
                        <div className="flex -space-x-3 mb-2">
                            <img className="w-8 h-8 rounded-full border-2 border-surface-950" src="https://i.pravatar.cc/100?img=1" alt="User" />
                            <img className="w-8 h-8 rounded-full border-2 border-surface-950" src="https://i.pravatar.cc/100?img=2" alt="User" />
                            <img className="w-8 h-8 rounded-full border-2 border-surface-950" src="https://i.pravatar.cc/100?img=3" alt="User" />
                            <div className="w-8 h-8 rounded-full border-2 border-surface-950 bg-surface-800 flex items-center justify-center text-[10px] font-bold">+2k</div>
                        </div>
                        <p className="text-xs text-surface-200/60 font-medium tracking-wide">TRUSTED BY DEVELOPERS</p>
                    </div>
                </div>
            </div>

            {/* Right side: Login Form */}
            <div className="flex-1 flex flex-col justify-center items-center p-6 lg:p-12 relative overflow-hidden bg-[#050B14]">
                {/* Subtle right side ambient glow */}
                <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-nova-900/20 rounded-full blur-[100px] pointer-events-none" />

                <div className="w-full max-w-[440px] relative z-10 animate-fade-in delay-150">

                    {/* Mobile Logo Header */}
                    <div className="lg:hidden flex flex-col items-center justify-center mb-10 gap-3">
                        <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-nova-500 to-nova-700 flex items-center justify-center shadow-nova-500/20 shadow-xl">
                            <Zap className="w-7 h-7 text-white" />
                        </div>
                        <h1 className="text-3xl font-bold text-white tracking-tight">NovaPanel</h1>
                    </div>

                    <div className="mb-10 text-center lg:text-left">
                        <h2 className="text-3xl font-bold text-white mb-3">
                            {mode === 'forgot' ? 'Reset Password' : mode === 'totp' ? 'Two-Factor Auth' : isLogin ? 'Welcome back' : 'Create an account'}
                        </h2>
                        <p className="text-surface-200/60 text-base">
                            {mode === 'forgot' ? 'Enter your email and we\'ll send you a reset link.'
                                : mode === 'totp' ? 'Enter the 6-digit code from your authenticator app.'
                                : isLogin ? 'Enter your credentials to access your control plane.'
                                : 'Join the next generation of server management.'}
                        </p>
                    </div>

                    {error && (
                        <div className="mb-6 p-4 rounded-xl glass-input border-danger/30 text-danger text-sm flex items-center gap-3 animate-slide-in">
                            <div className="w-1.5 h-1.5 rounded-full bg-danger animate-pulse" />
                            {error}
                        </div>
                    )}

                    {/* Forgot Password Form */}
                    {mode === 'forgot' && (
                        <form onSubmit={handleForgot} className="space-y-6">
                            {forgotSent ? (
                                <div className="p-4 rounded-xl bg-success/10 border border-success/20 text-success text-sm flex items-center gap-3">
                                    <Check className="w-5 h-5 flex-shrink-0" />
                                    Check your inbox — a reset link has been sent to {email}
                                </div>
                            ) : (
                                <div className="space-y-2">
                                    <label className="block text-sm font-medium text-surface-200">Email Address</label>
                                    <div className="relative group">
                                        <Mail className="absolute left-3.5 top-1/2 -translate-y-1/2 w-5 h-5 text-surface-200/40 group-focus-within:text-nova-400 transition-colors" />
                                        <input type="email" value={email} onChange={e => setEmail(e.target.value)} required
                                            className="block w-full pl-11 pr-4 py-3 rounded-xl glass-input text-white focus:outline-none focus:ring-2 focus:ring-nova-500/50 transition-all sm:text-sm placeholder:text-surface-200/20"
                                            placeholder="admin@company.com" />
                                    </div>
                                </div>
                            )}
                            {!forgotSent && (
                                <button type="submit" disabled={loading}
                                    className="w-full rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 py-3.5 px-4 text-sm font-semibold text-white shadow-lg hover:shadow-nova-500/40 transition-all flex items-center justify-center gap-2 disabled:opacity-70">
                                    {loading ? <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" /> : <><ArrowRight className="w-4 h-4" /> Send Reset Link</>}
                                </button>
                            )}
                            <button type="button" onClick={() => { setMode('auth'); setForgotSent(false); setError(''); }}
                                className="w-full text-sm text-surface-200/60 hover:text-white transition-colors">
                                ← Back to sign in
                            </button>
                        </form>
                    )}

                    {/* TOTP Verify Form */}
                    {mode === 'totp' && (
                        <form onSubmit={handleTotpVerify} className="space-y-6">
                            <div className="space-y-2">
                                <label className="block text-sm font-medium text-surface-200">Authentication Code</label>
                                <input type="text" value={totpCode} onChange={e => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                                    className="block w-full text-center tracking-[0.5em] text-xl px-4 py-3 rounded-xl glass-input text-white focus:outline-none focus:ring-2 focus:ring-nova-500/50 transition-all"
                                    placeholder="000000" maxLength={6} autoFocus required />
                                <p className="text-xs text-surface-200/40">Enter the code from your authenticator app, or a backup code.</p>
                            </div>
                            <button type="submit" disabled={loading || totpCode.length < 6}
                                className="w-full rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 py-3.5 px-4 text-sm font-semibold text-white shadow-lg hover:shadow-nova-500/40 transition-all flex items-center justify-center gap-2 disabled:opacity-70">
                                {loading ? <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" /> : <><Shield className="w-4 h-4" /> Verify</>}
                            </button>
                            <button type="button" onClick={() => { setMode('auth'); setTotpCode(''); setError(''); }}
                                className="w-full text-sm text-surface-200/60 hover:text-white transition-colors">
                                ← Use a different account
                            </button>
                        </form>
                    )}

                    {mode === 'auth' && <form onSubmit={handleSubmit} className="space-y-6">
                        {!isLogin && (
                            <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                    <label className="block text-sm font-medium text-surface-200">First Name</label>
                                    <div className="relative group">
                                        <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                                            <User className="w-5 h-5 text-surface-200/40 group-focus-within:text-nova-400 transition-colors" />
                                        </div>
                                        <input
                                            type="text"
                                            value={firstName}
                                            onChange={(e) => setFirstName(e.target.value)}
                                            className="block w-full pl-11 pr-4 py-3 rounded-xl glass-input text-white focus:outline-none focus:ring-2 focus:ring-nova-500/50 transition-all sm:text-sm placeholder:text-surface-200/20"
                                            placeholder="John"
                                            required={!isLogin}
                                        />
                                    </div>
                                </div>
                                <div className="space-y-2">
                                    <label className="block text-sm font-medium text-surface-200">Last Name</label>
                                    <div className="relative group">
                                        <input
                                            type="text"
                                            value={lastName}
                                            onChange={(e) => setLastName(e.target.value)}
                                            className="block w-full px-4 py-3 rounded-xl glass-input text-white focus:outline-none focus:ring-2 focus:ring-nova-500/50 transition-all sm:text-sm placeholder:text-surface-200/20"
                                            placeholder="Doe"
                                            required={!isLogin}
                                        />
                                    </div>
                                </div>
                            </div>
                        )}

                        <div className="space-y-2">
                            <label className="block text-sm font-medium text-surface-200">Work Email</label>
                            <div className="relative group">
                                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                                    <Mail className="w-5 h-5 text-surface-200/40 group-focus-within:text-nova-400 transition-colors" />
                                </div>
                                <input
                                    type="email"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    className="block w-full pl-11 pr-4 py-3 rounded-xl glass-input text-white focus:outline-none focus:ring-2 focus:ring-nova-500/50 transition-all sm:text-sm placeholder:text-surface-200/20"
                                    placeholder="admin@company.com"
                                    required
                                />
                            </div>
                        </div>

                        <div className="space-y-2">
                            <div className="flex items-center justify-between">
                                <label className="block text-sm font-medium text-surface-200">Password</label>
                                {isLogin && (
                                    <button type="button" onClick={() => { setMode('forgot'); setError(''); }}
                                        className="text-sm font-medium text-nova-400 hover:text-nova-300 transition-colors">
                                        Forgot password?
                                    </button>
                                )}
                            </div>
                            <div className="relative group">
                                <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none">
                                    <Lock className="w-5 h-5 text-surface-200/40 group-focus-within:text-nova-400 transition-colors" />
                                </div>
                                <input
                                    type={showPassword ? 'text' : 'password'}
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    className="block w-full pl-11 pr-12 py-3 rounded-xl glass-input text-white focus:outline-none focus:ring-2 focus:ring-nova-500/50 transition-all sm:text-sm placeholder:text-surface-200/20 tracking-wide"
                                    placeholder="••••••••"
                                    required
                                    minLength={8}
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowPassword(!showPassword)}
                                    className="absolute inset-y-0 right-0 pr-3.5 flex items-center text-surface-200/40 hover:text-surface-200/80 transition-colors"
                                    tabIndex={-1}
                                >
                                    {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                                </button>
                            </div>
                        </div>

                        <button
                            type="submit"
                            disabled={loading}
                            className="w-full relative overflow-hidden group rounded-xl bg-gradient-to-r from-nova-600 to-nova-700 py-3.5 px-4 text-sm font-semibold text-white shadow-lg shadow-nova-600/25 hover:shadow-nova-500/40 hover:from-nova-500 hover:to-nova-600 focus:outline-none focus:ring-2 focus:ring-nova-500/50 focus:ring-offset-2 focus:ring-offset-surface-950 transition-all duration-300 disabled:opacity-70 disabled:cursor-not-allowed flex items-center justify-center transform hover:-translate-y-0.5"
                        >
                            {/* Animated reflection effect over button */}
                            <div className="absolute inset-0 w-full h-full bg-white/20 translate-x-[-100%] skew-x-[-15deg] group-hover:animate-[shimmer_1.5s_infinite] pointer-events-none" />

                            {loading ? (
                                <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" />
                            ) : (
                                <div className="flex items-center gap-2">
                                    <span className="tracking-wide">{isLogin ? 'Sign in to dashboard' : 'Create your account'}</span>
                                    <ArrowRight className="w-4 h-4 transition-transform duration-300 group-hover:translate-x-1" />
                                </div>
                            )}
                        </button>
                    </form>}

                    {mode === 'auth' && <div className="mt-8 pt-8 border-t border-white/10 text-center space-y-3">
                        <p className="text-sm text-surface-200/60">
                            {isLogin ? "New to NovaPanel?" : "Already managing servers?"}{' '}
                            <button
                                onClick={() => { setIsLogin(!isLogin); setError(''); }}
                                className="font-semibold text-nova-400 hover:text-nova-300 transition-colors relative after:absolute after:bottom-0 after:left-0 after:h-[1px] auto:w-0 hover:after:w-full after:bg-nova-400 after:transition-all after:duration-300 pb-0.5"
                            >
                                {isLogin ? 'Create an account' : 'Sign in here'}
                            </button>
                        </p>
                        <p className="text-sm text-surface-200/40">
                            <Link to="/pricing" className="text-nova-400/80 hover:text-nova-300 transition-colors">
                                View plans &amp; pricing →
                            </Link>
                        </p>
                    </div>}
                </div>
            </div>
        </div>
    );
}
