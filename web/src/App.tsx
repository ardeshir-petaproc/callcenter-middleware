import { lazy, Suspense, type ReactNode } from 'react';
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { ShieldAlert } from 'lucide-react';
import { AuthProvider, useAuth } from './state/auth';
import { Shell } from './components/layout/Shell';
import { Toaster } from './components/ui/Toast';
import { ConfirmHost } from './components/ui/Confirm';
import { Empty, SkeletonRows } from './components/ui/primitives';
import { hasRole, isSupPlus } from './lib/roles';
import type { Role } from './api/types';
import LoginPage from './pages/LoginPage';

const ConsolePage = lazy(() => import('./pages/console/ConsolePage'));
const DashboardPage = lazy(() => import('./pages/dashboard/DashboardPage'));
const WallboardPage = lazy(() => import('./pages/dashboard/WallboardPage'));
const LiveCallsPage = lazy(() => import('./pages/live/LiveCallsPage'));
const CallHistoryPage = lazy(() => import('./pages/history/CallHistoryPage'));
const CallDetailPage = lazy(() => import('./pages/history/CallDetailPage'));
const QueuesPage = lazy(() => import('./pages/queues/QueuesPage'));
const UsersPage = lazy(() => import('./pages/users/UsersPage'));
const FlowsPage = lazy(() => import('./pages/flows/FlowsPage'));
const AIPage = lazy(() => import('./pages/ai/AIPage'));
const SchedulesPage = lazy(() => import('./pages/schedules/SchedulesPage'));
const CallbacksPage = lazy(() => import('./pages/callbacks/CallbacksPage'));
const VoicemailPage = lazy(() => import('./pages/voicemail/VoicemailPage'));
const ContactsPage = lazy(() => import('./pages/contacts/ContactsPage'));
const ReportsPage = lazy(() => import('./pages/reports/ReportsPage'));
const CampaignsPage = lazy(() => import('./pages/campaigns/CampaignsPage'));
const SettingsPage = lazy(() => import('./pages/settings/SettingsPage'));

function Loading() {
  return <div className="card"><SkeletonRows rows={6} /></div>;
}

function RequireAuth({ children }: { children: ReactNode }) {
  const { user, ready } = useAuth();
  const loc = useLocation();
  if (!user) {
    if (!ready) return <div className="login-page"><span className="spinner" /></div>;
    return <Navigate to="/login" replace state={{ from: loc.pathname + loc.search }} />;
  }
  return <>{children}</>;
}

function Gate({ min, children }: { min: Role; children: ReactNode }) {
  const { user } = useAuth();
  if (!hasRole(user?.role, min)) {
    return (
      <div className="card">
        <Empty title="Not available for your role" icon={<ShieldAlert />}>This page requires {min} access.</Empty>
      </div>
    );
  }
  return <Suspense fallback={<Loading />}>{children}</Suspense>;
}

function Home() {
  const { user } = useAuth();
  return <Navigate to={isSupPlus(user?.role) ? '/dashboard' : '/console'} replace />;
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/wallboard" element={<RequireAuth><Gate min="supervisor"><WallboardPage /></Gate></RequireAuth>} />
          <Route element={<RequireAuth><Shell /></RequireAuth>}>
            <Route index element={<Home />} />
            <Route path="console" element={<Gate min="agent"><ConsolePage /></Gate>} />
            <Route path="dashboard" element={<Gate min="supervisor"><DashboardPage /></Gate>} />
            <Route path="live" element={<Gate min="supervisor"><LiveCallsPage /></Gate>} />
            <Route path="calls" element={<Gate min="agent"><CallHistoryPage /></Gate>} />
            <Route path="calls/:id" element={<Gate min="agent"><CallDetailPage /></Gate>} />
            <Route path="queues/:id?" element={<Gate min="agent"><QueuesPage /></Gate>} />
            <Route path="users/:id?" element={<Gate min="supervisor"><UsersPage /></Gate>} />
            <Route path="flows/:id?" element={<Gate min="admin"><FlowsPage /></Gate>} />
            <Route path="ai/:id?" element={<Gate min="supervisor"><AIPage /></Gate>} />
            <Route path="schedules/:id?" element={<Gate min="admin"><SchedulesPage /></Gate>} />
            <Route path="callbacks" element={<Gate min="agent"><CallbacksPage /></Gate>} />
            <Route path="voicemail" element={<Gate min="agent"><VoicemailPage /></Gate>} />
            <Route path="contacts" element={<Gate min="agent"><ContactsPage /></Gate>} />
            <Route path="campaigns/:id?" element={<Gate min="supervisor"><CampaignsPage /></Gate>} />
            <Route path="reports" element={<Gate min="supervisor"><ReportsPage /></Gate>} />
            <Route path="settings" element={<Gate min="admin"><SettingsPage /></Gate>} />
            <Route path="*" element={<div className="card"><Empty title="Page not found">Check the address or use the navigation.</Empty></div>} />
          </Route>
        </Routes>
        <Toaster />
        <ConfirmHost />
      </BrowserRouter>
    </AuthProvider>
  );
}
