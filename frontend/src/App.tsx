import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useEffect } from 'react';
import MainLayout from './components/layout/MainLayout';
import { ToastProvider } from './components/ui/ToastProvider';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Domains from './pages/Domains';
import Servers from './pages/Servers';
import Databases from './pages/Databases';
import Email from './pages/Email';
import Backups from './pages/Backups';
import Monitoring from './pages/Monitoring';
import FileManager from './pages/FileManager';
import Deployments from './pages/Deployments';
import Security from './pages/Security';
import Billing from './pages/Billing';
import Settings from './pages/Settings';
import Terminal from './pages/Terminal';
import Docker from './pages/Docker';
import Kubernetes from './pages/Kubernetes';
import WAF from './pages/WAF';
import Transfers from './pages/Transfers';
import CronJobs from './pages/CronJobs';
import Services from './pages/Services';
import Cloudflare from './pages/Cloudflare';
import TeamManagement from './pages/TeamManagement';
import Pricing from './pages/Pricing';
import FTP from './pages/FTP';
import Alerts from './pages/Alerts';
import Reseller from './pages/Reseller';
import { useAuthStore } from './store/authStore';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, initialized } = useAuthStore();
  if (!initialized) return null; // wait for localStorage to be read
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  const { loadFromStorage } = useAuthStore();

  useEffect(() => {
    loadFromStorage();
  }, [loadFromStorage]);

  return (
    <ToastProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/pricing" element={<Pricing />} />

          <Route
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route path="/" element={<Dashboard />} />
            <Route path="/domains" element={<Domains />} />
            <Route path="/servers" element={<Servers />} />
            <Route path="/databases" element={<Databases />} />
            <Route path="/email" element={<Email />} />
            <Route path="/files" element={<FileManager />} />
            <Route path="/transfers" element={<Transfers />} />
            <Route path="/backups" element={<Backups />} />
            <Route path="/cron" element={<CronJobs />} />
            <Route path="/services" element={<Services />} />
            <Route path="/monitoring" element={<Monitoring />} />
            <Route path="/deployments" element={<Deployments />} />
            <Route path="/security" element={<Security />} />
            <Route path="/waf" element={<WAF />} />
            <Route path="/billing" element={<Billing />} />
            <Route path="/cloudflare" element={<Cloudflare />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/servers/:id/terminal" element={<Terminal />} />
            <Route path="/docker" element={<Docker />} />
            <Route path="/kubernetes" element={<Kubernetes />} />
            <Route path="/team" element={<TeamManagement />} />
            <Route path="/ftp" element={<FTP />} />
            <Route path="/alerts" element={<Alerts />} />
            <Route path="/reseller" element={<Reseller />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ToastProvider>
  );
}
