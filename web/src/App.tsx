import { lazy, Suspense, type ReactNode } from 'react';
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { ShieldAlert } from 'lucide-react';
import { AuthProvider, useAuth } from './state/auth';
import { Shell } from './components/layout/Shell';
import { Toaster } from './components/ui/Toast';
import { ConfirmHost } from './components/ui/Confirm';
import { Empty, SkeletonRows } from './components/ui/primitives';
import { isSupPlus } from './lib/roles';
import { hasAccess, type Access } from './lib/access';
import { PERMS } from './lib/perms';
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
const InvitePage = lazy(() => import('./pages/auth/InvitePage'));
const TicketsPage = lazy(() => import('./pages/tickets/TicketsPage'));
const TicketDetailPage = lazy(() => import('./pages/tickets/TicketDetailPage'));
const KnowledgePage = lazy(() => import('./pages/kb/KnowledgePage'));
const DepartmentsPage = lazy(() => import('./pages/org/DepartmentsPage'));
const JobProfilesPage = lazy(() => import('./pages/org/JobProfilesPage'));
const NumbersPage = lazy(() => import('./pages/org/NumbersPage'));
const RolesPage = lazy(() => import('./pages/org/RolesPage'));
const RoutingPage = lazy(() => import('./pages/routing/RoutingPage'));
const QualityPage = lazy(() => import('./pages/quality/QualityPage'));
const MyEvaluationsPage = lazy(() => import('./pages/quality/MyEvaluationsPage'));
const PlatformPage = lazy(() => import('./pages/platform/PlatformPage'));

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

/** Renders the page only for users with the permission (any of a list), the minimum role, or platform admin. */
function Gate({ children, ...access }: Access & { children: ReactNode }) {
  const { user, can } = useAuth();
  if (!hasAccess(access, user, can)) {
    const need = access.platform ? 'platform administrator' : [access.perm].flat().filter(Boolean).join(' or ') || `${access.min} role`;
    return (
      <div className="card">
        <Empty title="Not available for your account" icon={<ShieldAlert />}>This page requires {need} access.</Empty>
      </div>
    );
  }
  return <Suspense fallback={<Loading />}>{children}</Suspense>;
}

function Home() {
  const { user, can } = useAuth();
  return <Navigate to={can(PERMS.callsAll) || isSupPlus(user?.role) ? '/dashboard' : '/console'} replace />;
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/invite/:token" element={<Suspense fallback={<div className="login-page"><span className="spinner" /></div>}><InvitePage /></Suspense>} />
          <Route path="/accept-invite" element={<Suspense fallback={<div className="login-page"><span className="spinner" /></div>}><InvitePage /></Suspense>} />
          <Route path="/wallboard" element={<RequireAuth><Gate perm={PERMS.callsAll}><WallboardPage /></Gate></RequireAuth>} />
          <Route element={<RequireAuth><Shell /></RequireAuth>}>
            <Route index element={<Home />} />
            <Route path="console" element={<Gate min="agent"><ConsolePage /></Gate>} />
            <Route path="dashboard" element={<Gate perm={PERMS.callsAll}><DashboardPage /></Gate>} />
            <Route path="live" element={<Gate perm={PERMS.callsAll}><LiveCallsPage /></Gate>} />
            <Route path="calls" element={<Gate min="agent"><CallHistoryPage /></Gate>} />
            <Route path="calls/:id" element={<Gate min="agent"><CallDetailPage /></Gate>} />
            <Route path="tickets" element={<Gate min="agent"><TicketsPage /></Gate>} />
            <Route path="tickets/:id" element={<Gate min="agent"><TicketDetailPage /></Gate>} />
            <Route path="kb" element={<Gate min="agent"><KnowledgePage /></Gate>} />
            <Route path="queues/:id?" element={<Gate min="agent"><QueuesPage /></Gate>} />
            <Route path="users/:id?" element={<Gate min="supervisor" perm={PERMS.usersManage}><UsersPage /></Gate>} />
            <Route path="org/departments/:id?" element={<Gate perm={PERMS.orgManage}><DepartmentsPage /></Gate>} />
            <Route path="org/job-profiles/:id?" element={<Gate perm={PERMS.orgManage}><JobProfilesPage /></Gate>} />
            <Route path="org/numbers/:id?" element={<Gate perm={PERMS.flowsManage}><NumbersPage /></Gate>} />
            <Route path="org/roles/:id?" element={<Gate perm={PERMS.usersManage}><RolesPage /></Gate>} />
            <Route path="routing" element={<Gate perm={[PERMS.routingManage, PERMS.routingApprove]}><RoutingPage /></Gate>} />
            <Route path="quality" element={<Gate perm={PERMS.qaEvaluate}><QualityPage /></Gate>} />
            <Route path="my-evaluations" element={<Gate min="agent"><MyEvaluationsPage /></Gate>} />
            <Route path="platform" element={<Gate platform><PlatformPage /></Gate>} />
            <Route path="flows/:id?" element={<Gate perm={PERMS.flowsManage}><FlowsPage /></Gate>} />
            <Route path="ai/:id?" element={<Gate min="supervisor" perm={PERMS.aiManage}><AIPage /></Gate>} />
            <Route path="schedules/:id?" element={<Gate perm={PERMS.queuesManage}><SchedulesPage /></Gate>} />
            <Route path="callbacks" element={<Gate min="agent"><CallbacksPage /></Gate>} />
            <Route path="voicemail" element={<Gate min="agent"><VoicemailPage /></Gate>} />
            <Route path="contacts" element={<Gate min="agent"><ContactsPage /></Gate>} />
            <Route path="campaigns/:id?" element={<Gate perm={PERMS.campaignsManage}><CampaignsPage /></Gate>} />
            <Route path="reports" element={<Gate perm={PERMS.reportsView}><ReportsPage /></Gate>} />
            <Route path="settings" element={<Gate perm={PERMS.settingsManage}><SettingsPage /></Gate>} />
            <Route path="*" element={<div className="card"><Empty title="Page not found">Check the address or use the navigation.</Empty></div>} />
          </Route>
        </Routes>
        <Toaster />
        <ConfirmHost />
      </BrowserRouter>
    </AuthProvider>
  );
}
