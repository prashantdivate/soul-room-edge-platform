import React from "react";
import {
  Activity,
  AlertTriangle,
  Archive,
  Bell,
  Boxes,
  Check,
  ChevronDown,
  ChevronRight,
  CircleUserRound,
  ClipboardCheck,
  Clock3,
  CloudCog,
  Copy,
  Cpu,
  Database,
  Download,
  ExternalLink,
  FileJson,
  FileText,
  Eye,
  EyeOff,
  FileClock,
  Filter,
  Gauge,
  GitBranch,
  HardDrive,
  KeyRound,
  Layers3,
  LayoutGrid,
  ListFilter,
  LoaderCircle,
  LogOut,
  LockKeyhole,
  Mail,
  MapPinned,
  Menu,
  MoreHorizontal,
  Network,
  PackageCheck,
  PackageSearch,
  PanelLeftClose,
  PanelLeftOpen,
  Play,
  Plus,
  RadioTower,
  RefreshCw,
  Rocket,
  Save,
  Search,
  ServerCog,
  Settings2,
  ShieldCheck,
  SlidersHorizontal,
  TerminalSquare,
  Trash2,
  TriangleAlert,
  UploadCloud,
  UsersRound,
  X,
} from "lucide-react";
import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import type { SimpleIcon } from "simple-icons";
import {
  siAnydesk,
  siApache,
  siCurl,
  siDocker,
  siFlatpak,
  siGit,
  siGnubash,
  siLinux,
  siMariadb,
  siMongodb,
  siMysql,
  siNginx,
  siNodedotjs,
  siOpenssl,
  siPostgresql,
  siPython,
  siRabbitmq,
  siRedis,
  siTrivy,
  siUbuntu,
} from "simple-icons";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  LabelList,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  ApiError,
  createOTACampaign,
  createTeamUser,
  createRemoteAccess,
  createEnrollmentToken,
  createJob,
  deleteDevice,
  deleteJob,
  Device,
  EnrollmentToken,
  exportJson,
  exportPdf,
  FleetData,
  Job,
  loadFleet,
  login,
  logout,
  Membership,
  promoteOTACampaign,
  updateDevice,
} from "./api";
import { FleetMap } from "./FleetMap";

type PageId =
  | "overview"
  | "devices"
  | "gateways"
  | "telemetry"
  | "inventory"
  | "map"
  | "jobs"
  | "remote"
  | "deployments"
  | "ota"
  | "alerts"
  | "profiles"
  | "applications"
  | "artifacts"
  | "enrollment"
  | "audit"
  | "users"
  | "health"
  | "onprem";

type Page = { id: PageId; label: string; description: string; icon: React.ComponentType<{ size?: number }> };
type NavigationGroup = { label: string; icon: React.ComponentType<{ size?: number }>; pages: Page[] };

const navigation: NavigationGroup[] = [
  {
    label: "Monitor",
    icon: LayoutGrid,
    pages: [
      { id: "overview", label: "Overview", description: "What needs attention across your fleet today.", icon: Gauge },
      { id: "devices", label: "Devices", description: "Search, inspect, and operate enrolled Linux devices.", icon: Cpu },
      { id: "gateways", label: "Gateways", description: "Gateway connections and downstream industrial equipment.", icon: GitBranch },
      { id: "telemetry", label: "Telemetry", description: "Inspect normalized metric records and collection freshness.", icon: Activity },
      { id: "inventory", label: "Inventory", description: "Hardware and software facts reported by each device.", icon: Database },
      { id: "map", label: "Fleet map", description: "See the reported position and connectivity of every device.", icon: MapPinned },
    ],
  },
  {
    label: "Operate",
    icon: TerminalSquare,
    pages: [
      { id: "jobs", label: "Jobs", description: "Run approved commands and follow results.", icon: TerminalSquare },
      { id: "remote", label: "Remote access", description: "Open audited ShellHub SSH sessions to online devices.", icon: Network },
      { id: "deployments", label: "Deployments", description: "Roll out applications in controlled stages.", icon: Rocket },
      { id: "ota", label: "Update campaigns", description: "Roll out operating-system and Flatpak application updates.", icon: CloudCog },
      { id: "alerts", label: "Alerts", description: "Triage active issues and assign ownership.", icon: Bell },
    ],
  },
  {
    label: "Manage",
    icon: Layers3,
    pages: [
      { id: "profiles", label: "Profiles", description: "Reusable policy for telemetry, jobs, and updates.", icon: SlidersHorizontal },
      { id: "applications", label: "Applications", description: "Installed packages, advisory posture, and managed edge applications.", icon: Boxes },
      { id: "artifacts", label: "Artifacts", description: "Signed, immutable files available for delivery.", icon: UploadCloud },
      { id: "enrollment", label: "Enrollment", description: "Bring a device into the fleet with a one-time token.", icon: KeyRound },
    ],
  },
  {
    label: "Govern",
    icon: ShieldCheck,
    pages: [
      { id: "audit", label: "Audit log", description: "A durable record of security and operator activity.", icon: ShieldCheck },
      { id: "users", label: "Users & roles", description: "Organization access and permission boundaries.", icon: UsersRound },
    ],
  },
  {
    label: "Platform",
    icon: ServerCog,
    pages: [
      { id: "health", label: "System health", description: "Live resource, network, capacity, and availability trends.", icon: RadioTower },
      { id: "onprem", label: "On-prem admin", description: "Backup, certificates, storage, and platform lifecycle.", icon: ServerCog },
    ],
  },
];

const pageMap = Object.fromEntries(navigation.flatMap((group) => group.pages).map((page) => [page.id, page])) as Record<PageId, Page>;

export default function App() {
  const [membership, setMembership] = React.useState<Membership | null>(null);
  const [data, setData] = React.useState<FleetData | null>(null);
  const [page, setPage] = React.useState<PageId>("overview");
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");
  const [sidebarOpen, setSidebarOpen] = React.useState(() => window.innerWidth >= 840);
  const [search, setSearch] = React.useState("");
  const [searchOpen, setSearchOpen] = React.useState(false);
  const [notificationsOpen, setNotificationsOpen] = React.useState(false);
  const [profileOpen, setProfileOpen] = React.useState(false);
  const [exportOpen, setExportOpen] = React.useState(false);
  const [exporting, setExporting] = React.useState(false);
  const pageCanvasRef = React.useRef<HTMLDivElement>(null);

  const refresh = React.useCallback(async (activeMembership: Membership) => {
    setLoading(true);
    setError("");
    try {
      setData(await loadFleet(activeMembership));
    } catch (reason) {
      const message = reason instanceof Error ? reason.message : "The fleet could not be loaded.";
      if (reason instanceof ApiError && reason.status === 401) setMembership(null);
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
	if (!membership) return;
	const timer = window.setInterval(() => refresh(membership), 15_000);
	return () => window.clearInterval(timer);
  }, [membership, refresh]);

  async function handleLogin(email: string, password: string) {
    setLoading(true);
    setError("");
    try {
      const activeMembership = await login(email, password);
      setMembership(activeMembership);
      await refresh(activeMembership);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Sign in failed.");
      setLoading(false);
    }
  }

  async function handleLogout() {
	try { await logout(); } finally {
	  setMembership(null);
	  setData(null);
	  setProfileOpen(false);
	  setNotificationsOpen(false);
	}
  }

  if (!membership || !data) {
    return <Login loading={loading} error={error} onSubmit={handleLogin} />;
  }

  const activePage = pageMap[page];
  const signedInUser = data.users.find((user) => user.id === data.membership.user_id);
  const alerts = deriveAlerts(data.devices);
  const exportName = `soul-room-${page}-${new Date().toISOString().slice(0, 10)}`;
  const pageExport = pageExportData(page, data);
  return (
    <div className={sidebarOpen ? "appShell" : "appShell collapsed"}>
      <Sidebar
        active={page}
        alertCount={deriveAlerts(data.devices).length}
        open={sidebarOpen}
        onNavigate={(next) => {
          setPage(next);
          if (window.innerWidth < 840) setSidebarOpen(false);
        }}
      />
      <main className="workspace">
        <header className="topbar">
          <button className="iconButton mobileMenu" onClick={() => setSidebarOpen(!sidebarOpen)} aria-label="Toggle navigation">
            {sidebarOpen ? <PanelLeftClose size={19} /> : <PanelLeftOpen size={19} />}
          </button>
          <div className="breadcrumb">
            <span>{data.organization?.name || "Organization"}</span>
            <ChevronRight size={14} />
            <strong>Production</strong>
          </div>
          <div className="topbarRight">
            <div className="globalSearchWrap">
              <label className="globalSearch">
                <Search size={17} />
                <input value={search} onFocus={() => setSearchOpen(true)} onKeyDown={(event) => { if (event.key === "Escape") { setSearch(""); setSearchOpen(false); } }} onChange={(event) => { setSearch(event.target.value); setSearchOpen(true); }} placeholder="Search fleet" aria-label="Search devices, jobs, users, and updates" />
                {search && <button type="button" onClick={() => { setSearch(""); setSearchOpen(false); }} aria-label="Clear search"><X size={14} /></button>}
              </label>
              {searchOpen && search.trim() && <GlobalSearchResults data={data} query={search} onSelect={(result) => { setPage(result.page); setSearch(""); setSearchOpen(false); }} />}
            </div>
            <div className="topbarAction">
              <button className="iconButton notificationButton" aria-label="Notifications" aria-expanded={notificationsOpen} onClick={() => { setNotificationsOpen(!notificationsOpen); setProfileOpen(false); }}>
                <Bell size={18} />
                {alerts.length > 0 && <span />}
              </button>
              {notificationsOpen && <div className="topbarPopover notificationsPopover"><header><strong>Notifications</strong><span>{alerts.length} active</span></header>{alerts.length ? alerts.slice(0, 5).map((alert) => <button key={alert.id} onClick={() => { setPage(alert.page); setNotificationsOpen(false); }}><AlertTriangle size={16} /><span><strong>{alert.title}</strong><small>{alert.device} - {alert.detail}</small></span></button>) : <div className="popoverEmpty"><Check size={18} /><span>No active device alerts</span></div>}<button className="popoverFooter" onClick={() => { setPage("alerts"); setNotificationsOpen(false); }}>View alerts</button></div>}
            </div>
            <div className="topbarAction">
              <button className="userMenu" aria-label="Account menu" aria-expanded={profileOpen} onClick={() => { setProfileOpen(!profileOpen); setNotificationsOpen(false); }}>
                <div className="avatar">{(signedInUser?.email || "PA").slice(0, 2).toUpperCase()}</div>
                <div><strong>{signedInUser?.email || "Platform admin"}</strong><span>{data.membership.role.replaceAll("_", " ")}</span></div>
                <ChevronDown className={profileOpen ? "menuChevron open" : "menuChevron"} size={15} />
              </button>
              {profileOpen && <div className="topbarPopover accountPopover"><div><span>Signed in as</span><strong>{signedInUser?.email || "Platform admin"}</strong></div><button onClick={handleLogout}><LogOut size={16} /> Log out</button></div>}
            </div>
          </div>
        </header>

        <div className="pageCanvas" ref={pageCanvasRef}>
          <section className="pageHeading">
            <div>
              <h1>{activePage.label}</h1>
              <p>{activePage.description}</p>
            </div>
            <div className="headingActions">
              <div className="exportAction">
                <button className="button secondary" aria-expanded={exportOpen} onClick={() => setExportOpen(!exportOpen)}><Download size={16} /> Export <ChevronDown size={14} /></button>
                {exportOpen && <div className="exportMenu">
                  <button onClick={() => { exportJson(`${exportName}.json`, pageExport); setExportOpen(false); }}><FileJson size={17} /><span><strong>JSON data</strong><small>Structured records from this page</small></span></button>
                  <button disabled={exporting} onClick={async () => { if (!pageCanvasRef.current) return; setExporting(true); setExportOpen(false); try { await exportPdf(`${exportName}.pdf`, pageCanvasRef.current); } finally { setExporting(false); } }}><FileText size={17} /><span><strong>PDF report</strong><small>Charts and visible page content</small></span></button>
                </div>}
              </div>
              <button className="iconButton" onClick={() => refresh(membership)} title="Refresh data"><RefreshCw className={loading ? "spin" : ""} size={17} /></button>
            </div>
          </section>

          {error && <div className="inlineError"><TriangleAlert size={17} />{error}<button onClick={() => setError("")}><X size={15} /></button></div>}
          <PageContent page={page} data={data} search={search} onData={setData} onNavigate={setPage} onRefresh={() => refresh(membership)} />
        </div>
      </main>
    </div>
  );
}

function Sidebar({ active, open, alertCount, onNavigate }: { active: PageId; open: boolean; alertCount: number; onNavigate: (page: PageId) => void }) {
  const activeGroup = navigation.find((group) => group.pages.some((page) => page.id === active)) || navigation[0];
  return (
    <aside className={open ? "sidebar open" : "sidebar"} aria-label="Primary navigation">
      <div className="navRail">
        <div className="railBrand"><img src="/soul-room-mark.png?v=2" alt="Soul Room" /></div>
        <nav aria-label="Workspace groups">{navigation.map((group) => { const Icon = group.icon; const selected = group === activeGroup; return <button key={group.label} className={selected ? "railButton active" : "railButton"} onClick={() => onNavigate(group.pages[0].id)} title={group.label} aria-label={group.label}><Icon size={20} />{group.label === "Operate" && alertCount > 0 && <i />}</button>; })}</nav>
        <span className="railStatus" title="Platform online"><i /></span>
      </div>
      <div className="navPanel">
        <div className="brand"><img src="/soul-room-mark.png?v=2" alt="Soul Room" /></div>
        <div className="navContext"><span>Workspace</span><strong>{activeGroup.label}</strong></div>
        <nav className="navScroll">
          <div className="navGroup">
            {activeGroup.pages.map((item) => {
              const Icon = item.icon;
              return <button key={item.id} className={active === item.id ? "navItem active" : "navItem"} onClick={() => onNavigate(item.id)} title={item.label}><Icon size={18} /><span>{item.label}</span>{item.id === "alerts" && alertCount > 0 && <small>{alertCount}</small>}</button>;
            })}
          </div>
        </nav>
        <div className="sidebarFoot"><span className="connectionDot" /><div><strong>Soul Room Local</strong><span>All services healthy</span></div></div>
      </div>
    </aside>
  );
}

type SearchResult = { id: string; label: string; detail: string; category: string; page: PageId };

function GlobalSearchResults({ data, query, onSelect }: { data: FleetData; query: string; onSelect: (result: SearchResult) => void }) {
  const needle = query.trim().toLowerCase();
  const results: SearchResult[] = [
    ...data.devices.map((device) => ({ id: `device-${device.id}`, label: device.display_name, detail: [device.hardware_model, device.os, device.serial].filter(Boolean).join(" · "), category: "Device", page: "devices" as PageId })),
    ...data.jobs.map((job) => ({ id: `job-${job.id}`, label: job.type.replaceAll("_", " "), detail: `${nameFor(data, job.device_id)} · ${job.state} · ${shortId(job.id)}`, category: "Job", page: "jobs" as PageId })),
    ...data.otaCampaigns.map((campaign) => ({ id: `ota-${campaign.id}`, label: campaign.name, detail: `${campaign.adapter} · ${campaign.flatpak_ref || campaign.version} · ${campaign.state}`, category: "Update", page: "ota" as PageId })),
    ...data.users.map((user) => ({ id: `user-${user.id}`, label: user.email, detail: user.role.replaceAll("_", " "), category: "User", page: "users" as PageId })),
    ...data.audit.map((event) => ({ id: `audit-${event.id}`, label: event.action, detail: `${event.resource_type} · ${event.result}`, category: "Audit", page: "audit" as PageId })),
  ].filter((result) => `${result.label} ${result.detail} ${result.category}`.toLowerCase().includes(needle)).slice(0, 8);
  return <div className="searchPopover" role="listbox"><header><strong>Fleet search</strong><span>{results.length} result{results.length === 1 ? "" : "s"}</span></header>{results.length ? results.map((result) => <button type="button" role="option" key={result.id} onMouseDown={(event) => event.preventDefault()} onClick={() => onSelect(result)}><span className="searchResultIcon"><Search size={14} /></span><span><strong>{result.label}</strong><small>{result.detail || "No additional details"}</small></span><em>{result.category}</em></button>) : <div className="popoverEmpty"><PackageSearch size={18} /><span>No fleet records match “{query.trim()}”</span></div>}</div>;
}

function Login({ loading, error, onSubmit }: { loading: boolean; error: string; onSubmit: (email: string, password: string) => void }) {
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [showPassword, setShowPassword] = React.useState(false);
  const [capsLock, setCapsLock] = React.useState(false);
  const [eventIndex, setEventIndex] = React.useState(0);
  const reduceMotion = useReducedMotion();
  const events = ["Device identity verified", "Health signal received", "Pilot update staged", "Policy change recorded"];
  const canSubmit = email.trim().length > 3 && password.length > 0 && !loading;
  React.useEffect(() => {
    if (reduceMotion) return;
    const timer = window.setInterval(() => setEventIndex((index) => (index + 1) % events.length), 2600);
    return () => window.clearInterval(timer);
  }, [reduceMotion, events.length]);
  return (
    <main className="loginPage">
      <section className="loginStory">
        <div className="loginBrand"><img src="/soul-room-mark.png?v=2" alt="Soul Room" /></div>
        <div className="storyCopy"><span>Edge operations, composed</span><h1>A calm room for every connected device.</h1><p>Observe health, understand software risk, and deliver controlled changes across embedded Linux fleets.</p></div>
        <div className="operationsScene" aria-hidden="true">
          <motion.div className="sceneLane edgeLane" initial={{ opacity: 0, x: -24 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: .55 }}>
            <span><Cpu size={18} /></span><span><RadioTower size={18} /></span><span><GitBranch size={18} /></span><small>EDGE FLEET</small>
          </motion.div>
          <div className="signalPath leftPath">{[0, 1, 2].map((packet) => <motion.i key={packet} animate={reduceMotion ? undefined : { left: ["2%", "88%"], opacity: [0, 1, 1, 0] }} transition={{ duration: 2.5, repeat: Infinity, delay: packet * .8, ease: "easeInOut" }} />)}</div>
          <motion.div className="sceneCore" initial={{ opacity: 0, scale: .78 }} animate={{ opacity: 1, scale: 1 }} transition={{ type: "spring", stiffness: 170, damping: 18, delay: .2 }}><img src="/soul-room-mark.png?v=2" alt="" /><span>CONTROL PLANE</span><motion.b animate={reduceMotion ? undefined : { scale: [1, 1.18, 1], opacity: [.45, .12, .45] }} transition={{ duration: 2.8, repeat: Infinity }} /></motion.div>
          <div className="signalPath rightPath">{[0, 1, 2].map((packet) => <motion.i key={packet} animate={reduceMotion ? undefined : { left: ["2%", "88%"], opacity: [0, 1, 1, 0] }} transition={{ duration: 2.8, repeat: Infinity, delay: packet * .9, ease: "easeInOut" }} />)}</div>
          <motion.div className="sceneLane controlLane" initial={{ opacity: 0, x: 24 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: .55, delay: .15 }}>
            <span><ShieldCheck size={18} /></span><span><CloudCog size={18} /></span><span><TerminalSquare size={18} /></span><small>OPERATIONS</small>
          </motion.div>
        </div>
        <div className="sceneEvent"><span className="connectionDot" /><AnimatePresence mode="wait"><motion.strong key={events[eventIndex]} initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }} transition={{ duration: .28 }}>{events[eventIndex]}</motion.strong></AnimatePresence><small>Encrypted and audit recorded</small></div>
      </section>
      <section className="loginFormWrap">
        <motion.form className="loginForm" initial={{ opacity: 0, y: 18 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: .5, delay: .12 }} onSubmit={(event) => { event.preventDefault(); if (canSubmit) onSubmit(email.trim().toLowerCase(), password); }}>
          <div className="mobileLoginBrand"><img src="/soul-room-mark.png?v=2" alt="Soul Room" /></div>
          <div className="workspaceBadge"><ShieldCheck size={14} /><span>Private company workspace</span></div>
          <div className="formIntro"><h2>Welcome back</h2><p>Use the account provided by your organization administrator.</p></div>
          {error && <div className="formError"><AlertTriangle size={17} />{error}</div>}
          <label>Email address<div className="loginInput"><Mail size={16} /><input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="you@company.com" autoComplete="username" inputMode="email" required autoFocus /></div></label>
          <label>Password<div className="loginInput"><LockKeyhole size={16} /><input type={showPassword ? "text" : "password"} value={password} onKeyUp={(event) => setCapsLock(event.getModifierState("CapsLock"))} onChange={(event) => setPassword(event.target.value)} placeholder="Enter your password" autoComplete="current-password" required /><button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? "Hide password" : "Show password"}>{showPassword ? <EyeOff size={16} /> : <Eye size={16} />}</button></div>{capsLock && <span className="capsNotice">Caps Lock is on</span>}</label>
          <motion.button whileHover={!canSubmit || reduceMotion ? undefined : { y: -1 }} whileTap={!canSubmit || reduceMotion ? undefined : { scale: .985 }} className="button primary loginButton" disabled={!canSubmit}>{loading ? <LoaderCircle className="spin" size={18} /> : <LockKeyhole size={17} />}{loading ? "Signing in..." : "Sign in securely"}</motion.button>
          <div className="loginHelp"><span>Need access?</span><strong>Contact your company administrator.</strong></div>
          <div className="loginAssurance"><ShieldCheck size={16} /><span>Encrypted session</span><i /><span>Tenant-isolated access</span></div>
        </motion.form>
      </section>
    </main>
  );
}

function PageContent({ page, data, search, onData, onNavigate, onRefresh }: { page: PageId; data: FleetData; search: string; onData: (data: FleetData) => void; onNavigate: (page: PageId) => void; onRefresh: () => void }) {
  const props = { data, search, onData, onNavigate, onRefresh };
  switch (page) {
    case "overview": return <Overview {...props} />;
    case "devices": return <Devices {...props} />;
    case "gateways": return <Gateways {...props} />;
    case "telemetry": return <Telemetry {...props} />;
    case "inventory": return <InventoryView {...props} />;
    case "map": return <FleetMap data={data} onData={onData} onRemote={(device) => { sessionStorage.setItem("soul_room_remote_device", device.id); onNavigate("remote"); }} />;
    case "jobs": return <Jobs {...props} />;
    case "remote": return <RemoteAccess {...props} />;
    case "deployments": return <Deployments data={data} />;
    case "ota": return <OTA data={data} onData={onData} />;
    case "alerts": return <Alerts data={data} />;
    case "profiles": return <Profiles data={data} />;
    case "applications": return <Applications data={data} onData={onData} />;
    case "artifacts": return <Artifacts data={data} />;
    case "enrollment": return <Enrollment {...props} />;
    case "audit": return <Audit data={data} search={search} />;
    case "users": return <Users data={data} onData={onData} />;
    case "health": return <Health data={data} />;
    case "onprem": return <OnPrem data={data} />;
  }
}

type ViewProps = { data: FleetData; search: string; onData: (data: FleetData) => void; onNavigate: (page: PageId) => void; onRefresh: () => void };

function Overview({ data, onNavigate }: ViewProps) {
  if (data.devices.length === 0) return <EmptyFleet onEnroll={() => onNavigate("enrollment")} />;
  const connected = data.devices.filter((device) => device.presence === "connected").length;
  const alerts = deriveAlerts(data.devices);
  const current = data.devices.filter((device) => device.update_state === "current").length;
  return <>
    <section className="metricStrip">
      <Metric label="Connected now" value={`${connected} / ${data.devices.length}`} note="Last heartbeat within 5 minutes" tone="green" icon={<RadioTower size={18} />} />
      <Metric label="Needs attention" value={String(alerts.length)} note={alerts.length ? "Derived from live device state" : "No active device issues"} tone={alerts.length ? "red" : "green"} icon={<AlertTriangle size={18} />} />
      <Metric label="Jobs waiting" value={String(data.jobs.filter((job) => job.state === "queued").length)} note="Will deliver on reconnect" tone="amber" icon={<Clock3 size={18} />} />
      <Metric label="Update coverage" value={`${Math.round(current * 100 / data.devices.length)}%`} note={`${current} of ${data.devices.length} devices current`} tone="blue" icon={<PackageCheck size={18} />} />
    </section>
    <Panel title="Fleet availability" subtitle="Current presence reported by agents" action={<button className="textButton" onClick={() => onNavigate("health")}>Open system health <ChevronRight size={15} /></button>}>
      <div className="overviewAvailability"><div className="donutChart"><PresenceChart connected={connected} total={data.devices.length} /></div><div className="availabilityCopy"><span className="overline">Heartbeat posture</span><strong>{connected === data.devices.length ? "Every device is reachable" : `${data.devices.length - connected} device${data.devices.length - connected === 1 ? "" : "s"} need a connection check`}</strong><p>Soul Room marks a device offline only when its real heartbeat exceeds the presence window. Host resource charts live in System Health.</p></div></div>
    </Panel>
    <Panel title="Attention queue" subtitle="Generated from current device health, presence, and certificate dates">
      {alerts.length ? <div className="attentionList">{alerts.map((alert) => <Attention key={alert.id} severity={alert.severity} title={alert.title} detail={alert.detail} action="Review" onClick={() => onNavigate(alert.page)} />)}</div> : <EmptyState text="No device issues are active." />}
    </Panel>
    <Panel title="Device posture" subtitle="Presence, policy, and software state at a glance" action={<button className="textButton" onClick={() => onNavigate("devices")}>View all devices <ChevronRight size={15} /></button>}>
      <DeviceTable devices={data.devices} compact />
    </Panel>
  </>;
}

function Devices({ data, search, onNavigate, onData }: ViewProps) {
  const [status, setStatus] = React.useState("all");
  const [selected, setSelected] = React.useState<Device | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<Device | null>(null);
  const [deleting, setDeleting] = React.useState(false);
  const [deleteError, setDeleteError] = React.useState("");
  const canDelete = canAdminister(data.membership);
  async function removeDevice() {
    if (!deleteTarget) return;
    setDeleting(true); setDeleteError("");
    try {
      await deleteDevice(data.membership, deleteTarget.id);
      onData({
        ...data,
        devices: data.devices.filter((device) => device.id !== deleteTarget.id),
        metrics: data.metrics.filter((metric) => metric.device_id !== deleteTarget.id),
        jobs: data.jobs.filter((job) => job.device_id !== deleteTarget.id),
        downstream: data.downstream.filter((device) => device.parent_gateway_id !== deleteTarget.id),
        otaCampaigns: data.otaCampaigns.filter((campaign) => campaign.target_device_ids.some((id) => id !== deleteTarget.id)).map((campaign) => ({ ...campaign, target_device_ids: campaign.target_device_ids.filter((id) => id !== deleteTarget.id) })),
      });
      setDeleteTarget(null);
    } catch (reason) { setDeleteError(reason instanceof Error ? reason.message : "The device could not be removed."); }
    finally { setDeleting(false); }
  }
  const filtered = data.devices.filter((device) => (status === "all" || device.presence === status) && `${device.display_name} ${device.serial} ${device.tags?.join(" ")}`.toLowerCase().includes(search.toLowerCase()));
  return <>
    <Toolbar><div className="segmented"><button className={status === "all" ? "active" : ""} onClick={() => setStatus("all")}>All <span>{data.devices.length}</span></button><button className={status === "connected" ? "active" : ""} onClick={() => setStatus("connected")}>Connected</button><button className={status === "offline" ? "active" : ""} onClick={() => setStatus("offline")}>Offline</button></div><button className="button primary" onClick={() => onNavigate("enrollment")}><Plus size={16} /> Add device</button></Toolbar>
    <Panel title={`${filtered.length} devices`} subtitle="Click a device to inspect its identity and current state">
      <DeviceTable devices={filtered} onSelect={setSelected} />
    </Panel>
    {selected && <Drawer title={selected.display_name} onClose={() => setSelected(null)}>
      <div className="deviceIdentity"><div className="deviceGlyph"><Cpu size={25} /></div><div><Badge value={selected.presence} /><p>{selected.hardware_model}</p></div></div>
      <DetailList items={[["Serial", selected.serial], ["Profile", selected.profile_id], ["Operating system", selected.os], ["Kernel", selected.kernel], ["Architecture", selected.architecture], ["Agent", selected.agent_version], ["Last seen", formatDate(selected.last_seen_at)], ["Certificate expires", formatDate(selected.certificate_expires_at)]]} />
      <h3>Capabilities</h3><div className="tagRow">{selected.capabilities?.map((value) => <span className="tag" key={value}>{value}</span>)}</div>
      <div className="drawerActions"><button className="button warningButton" onClick={() => { setSelected(null); onNavigate("jobs"); }}><TerminalSquare size={16} /> Run job</button><button className="button infoButton" onClick={() => { setSelected(null); onNavigate("health"); }}><Activity size={16} /> System health</button>{canDelete && <button className="button dangerButton" onClick={() => { setDeleteTarget(selected); setSelected(null); }}><Trash2 size={16} /> Remove device</button>}</div>
    </Drawer>}
    {deleteTarget && <Modal title="Remove registered device" onClose={() => !deleting && setDeleteTarget(null)}><div className="modalForm"><div className="destructiveNotice"><Trash2 size={20} /><div><strong>{deleteTarget.display_name}</strong><span>This removes its registration, telemetry, inventory, pending jobs, and update targeting. Audit records remain.</span></div></div>{deleteError && <div className="formError"><AlertTriangle size={17} />{deleteError}</div>}<div className="modalActions"><button className="button secondary" disabled={deleting} onClick={() => setDeleteTarget(null)}>Cancel</button><button className="button dangerButton" disabled={deleting} onClick={removeDevice}>{deleting ? <LoaderCircle className="spin" size={16} /> : <Trash2 size={16} />} Remove device</button></div></div></Modal>}
  </>;
}

function Gateways({ data }: ViewProps) {
  const gateways = data.devices.filter((device) => device.capabilities?.includes("gateway"));
  if (gateways.length === 0) return <EmptySection icon={<GitBranch size={28} />} title="No gateways enrolled" text="Devices that report the gateway capability and their downstream equipment will appear here." />;
  return <section className="gatewayLayout">
    <Panel title="Gateway topology" subtitle={`${gateways.length} gateway and ${data.downstream.length} downstream connections`}>
      {gateways.map((gateway) => <div className="gatewayTree" key={gateway.id}>
        <div className="gatewayRoot"><div className="nodeIcon"><RadioTower size={20} /></div><div><strong>{gateway.display_name}</strong><span>{gateway.hardware_model} · {gateway.presence}</span></div><Badge value={gateway.health_state || "unknown"} /></div>
        <div className="childrenLine">
          {data.downstream.filter((child) => child.parent_gateway_id === gateway.id).map((child) => <div className="childNode" key={child.id}><div className="nodeIcon muted"><GitBranch size={18} /></div><div><strong>{child.model}</strong><span>{child.connector_type} · {child.protocol_address}</span></div><Badge value={child.health} /></div>)}
        </div>
      </div>)}
    </Panel>
    <Panel title="Protocol distribution" subtitle="Live downstream devices grouped by connector">
      <div className="barChart">{data.downstream.length ? <ConnectorChart devices={data.downstream} /> : <ChartEmpty text="No downstream connectors have reported yet" />}</div>
    </Panel>
  </section>;
}

function Telemetry({ data }: ViewProps) {
  const [deviceId, setDeviceId] = React.useState(data.devices[0]?.id || "");
  const [range, setRange] = React.useState<"15m" | "1h" | "24h">("1h");
  React.useEffect(() => {
    if (!deviceId && data.devices[0]) setDeviceId(data.devices[0].id);
  }, [data.devices, deviceId]);
  if (data.devices.length === 0) return <EmptySection icon={<Activity size={28} />} title="No telemetry yet" text="Enroll and start an agent. CPU, memory, disk, temperature, load, and network samples will appear automatically." />;
  const selected = data.devices.find((device) => device.id === deviceId);
  const cutoff = Date.now() - { "15m": 15 * 60_000, "1h": 60 * 60_000, "24h": 24 * 60 * 60_000 }[range];
  const metrics = data.metrics.filter((metric) => metric.device_id === deviceId && new Date(metric.received_at || metric.device_time).getTime() >= cutoff);
  const series = Array.from(new Set(metrics.map((metric) => metric.name))).sort();
  const lastReceived = metrics.reduce((latest, metric) => Math.max(latest, new Date(metric.received_at || metric.device_time).getTime()), 0);
  return <>
    <Toolbar><label className="selectLabel">Device<select value={deviceId} onChange={(event) => setDeviceId(event.target.value)}>{data.devices.map((device) => <option value={device.id} key={device.id}>{device.display_name}</option>)}</select></label><div className="segmented" aria-label="Telemetry time range"><button className={range === "15m" ? "active" : ""} onClick={() => setRange("15m")}>15 min</button><button className={range === "1h" ? "active" : ""} onClick={() => setRange("1h")}>1 hour</button><button className={range === "24h" ? "active" : ""} onClick={() => setRange("24h")}>24 hours</button></div></Toolbar>
    <section className="metricStrip telemetryMetrics"><Metric label="Records" value={String(metrics.length)} note={`Accepted during ${range}`} tone="blue" icon={<Database size={18} />} /><Metric label="Metric series" value={String(series.length)} note="Unique normalized names" tone="green" icon={<ListFilter size={18} />} /><Metric label="Last accepted" value={lastReceived ? relativeTime(new Date(lastReceived).toISOString()) : "Never"} note="Control-plane receive time" tone="amber" icon={<Clock3 size={18} />} /><Metric label="Agent state" value={selected?.presence || "unknown"} note={selected?.agent_version ? `Version ${selected.agent_version}` : "No version reported"} tone={selected?.presence === "connected" ? "green" : "red"} icon={<RadioTower size={18} />} /></section>
    <Panel title="Metric stream" subtitle={`${selected?.display_name || "Device"} - normalized records after server validation`}><SimpleTable headers={["Received", "Metric", "Value", "Unit"]} rows={metrics.slice(-30).reverse().map((metric) => [formatDate(metric.received_at || metric.device_time), metric.name, metric.value.toFixed(1), metric.unit])} empty={`No metric records were received during the selected ${range} window.`} /></Panel>
    <Panel title="Series catalog" subtitle="Names currently emitted by this agent"><div className="seriesCatalog">{series.map((name) => <span key={name}>{name}</span>)}{series.length === 0 && <EmptyState text="No series are available in this time window." />}</div></Panel>
  </>;
}

function InventoryView({ data, search }: ViewProps) {
  const rows = data.devices.filter((device) => device.display_name.toLowerCase().includes(search.toLowerCase())).map((device) => {
    const facts = data.inventory[device.id] || {};
    return [<div className="primaryCell"><Cpu size={17} /><div><strong>{device.display_name}</strong><span>{device.serial}</span></div></div>, device.hardware_model, facts.cpu_model ? String(facts.cpu_model) : "—", device.os, device.kernel, facts.cpu_cores ? String(facts.cpu_cores) : "—", formatBytes(Number(facts.memory_total_bytes || 0)), formatBytes(Number(facts.storage_total_bytes || 0))];
  });
  return <><Toolbar><div className="toolbarSummary"><strong>{data.devices.length} reporting devices</strong><span>Inventory is refreshed by every agent collection cycle</span></div></Toolbar><Panel title="Hardware & software inventory" subtitle="Latest accepted snapshot from each agent"><SimpleTable headers={["Device", "Hardware", "CPU", "Operating system", "Kernel", "Cores", "Memory", "Storage"]} rows={rows} empty="Enroll a device to receive its hardware and operating-system inventory." /></Panel></>;
}

function Jobs({ data, onData }: ViewProps) {
  const [showCreate, setShowCreate] = React.useState(false);
  const [saving, setSaving] = React.useState(false);
  const [deviceId, setDeviceId] = React.useState(data.devices[0]?.id || "");
  const [type, setType] = React.useState("collect_diagnostics");
  const [filter, setFilter] = React.useState<"all" | "queued" | "completed">("all");
  const [jobError, setJobError] = React.useState("");
  const [selectedJob, setSelectedJob] = React.useState<Job | null>(null);
  const [deleteArmed, setDeleteArmed] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  React.useEffect(() => {
    if (!deviceId && data.devices[0]) setDeviceId(data.devices[0].id);
  }, [data.devices, deviceId]);
  async function submit() {
    setSaving(true);
    setJobError("");
    try {
      const job = await createJob(data.membership, deviceId, type);
      onData({ ...data, jobs: [job, ...data.jobs.filter((item) => item.id !== job.id)] });
      setShowCreate(false);
      setFilter("queued");
    } catch (reason) {
      setJobError(reason instanceof Error ? reason.message : "The job could not be queued.");
    } finally { setSaving(false); }
  }
  const visibleJobs = data.jobs.filter((job) => filter === "all" || (filter === "queued" ? ["queued", "delivered"].includes(job.state) : ["succeeded", "failed", "expired"].includes(job.state)));
  const queuedCount = data.jobs.filter((job) => ["queued", "delivered"].includes(job.state)).length;
  const completedCount = data.jobs.filter((job) => ["succeeded", "failed", "expired"].includes(job.state)).length;
  const canDelete = canAdminister(data.membership) && selectedJob && ["succeeded", "failed", "expired", "cancelled"].includes(selectedJob.state);
  async function removeJob() {
    if (!selectedJob) return;
    setDeleting(true); setJobError("");
    try { await deleteJob(data.membership, selectedJob.id); onData({ ...data, jobs: data.jobs.filter((job) => job.id !== selectedJob.id) }); setSelectedJob(null); setDeleteArmed(false); }
    catch (reason) { setJobError(reason instanceof Error ? reason.message : "The job history entry could not be removed."); }
    finally { setDeleting(false); }
  }
  return <><Toolbar><div className="segmented" aria-label="Job state filter"><button className={filter === "all" ? "active" : ""} onClick={() => setFilter("all")}>All jobs <span>{data.jobs.length}</span></button><button className={filter === "queued" ? "active" : ""} onClick={() => setFilter("queued")}>Queued <span>{queuedCount}</span></button><button className={filter === "completed" ? "active" : ""} onClick={() => setFilter("completed")}>Completed <span>{completedCount}</span></button></div><button className="button warningButton" disabled={data.devices.length === 0} onClick={() => { setJobError(""); setShowCreate(true); }}><Play size={16} /> Run job</button></Toolbar><Panel title="Job history" subtitle="Safe device operations and their returned results"><SimpleTable headers={["Job", "Device", "Type", "State", "Approval", "Created", "Expires", "Result"]} rows={visibleJobs.map((job) => [shortId(job.id), nameFor(data, job.device_id), job.type.replaceAll("_", " "), <Badge value={job.state} />, job.approval_status, formatDate(job.created_at), formatDate(job.expires_at), <button className="textButton" onClick={() => { setJobError(""); setDeleteArmed(false); setSelectedJob(job); }}>View</button>])} empty={filter === "all" ? "No jobs have been sent to a real device." : `No ${filter} jobs.`} /></Panel>{showCreate && <Modal title="Run a device job" onClose={() => setShowCreate(false)}><div className="modalForm"><label>Target device<select value={deviceId} onChange={(event) => setDeviceId(event.target.value)}>{data.devices.map((device) => <option value={device.id} key={device.id}>{device.display_name} - {device.presence}</option>)}</select></label><label>Job type<select value={type} onChange={(event) => setType(event.target.value)}><option value="collect_diagnostics">Collect diagnostics</option><option value="collect_logs">Collect logs</option><option value="collect_inventory">Refresh inventory</option></select></label>{jobError && <div className="formError"><AlertTriangle size={17} />{jobError}</div>}<div className="notice"><ShieldCheck size={18} /><div><strong>Safe operation</strong><span>No arbitrary shell command, reboot, or service restart is permitted.</span></div></div><div className="modalActions"><button className="button secondary" onClick={() => setShowCreate(false)}>Cancel</button><button className="button warningButton" onClick={submit} disabled={saving || !deviceId}>{saving ? <LoaderCircle className="spin" size={16} /> : <Play size={16} />} Queue job</button></div></div></Modal>}{selectedJob && <Modal title="Job details" onClose={() => { if (!deleting) { setSelectedJob(null); setDeleteArmed(false); } }}><div className="jobResult"><DetailList items={[["Job", selectedJob.id], ["Device", nameFor(data, selectedJob.device_id)], ["Type", selectedJob.type.replaceAll("_", " ")], ["State", <Badge value={selectedJob.state} />], ["Created", formatDate(selectedJob.created_at)]]} /><h3>Returned output</h3><pre>{jobOutput(selectedJob)}</pre>{jobError && <div className="formError"><AlertTriangle size={17} />{jobError}</div>}{canDelete && <div className="jobDelete"><div><strong>Remove this history entry</strong><span>Completed output is deleted. Its audit trail is retained.</span></div>{deleteArmed ? <div className="jobDeleteConfirm"><button className="button secondary" onClick={() => setDeleteArmed(false)}>Cancel</button><button className="button dangerButton" disabled={deleting} onClick={removeJob}>{deleting ? <LoaderCircle className="spin" size={16} /> : <Trash2 size={16} />} Confirm delete</button></div> : <button className="button dangerButton" onClick={() => setDeleteArmed(true)}><Trash2 size={16} /> Delete history</button>}</div>}</div></Modal>}</>;
}

function RemoteAccess({ data, onData }: ViewProps) {
  const initial = sessionStorage.getItem("soul_room_remote_device") || data.devices[0]?.id || "";
  const [deviceId, setDeviceId] = React.useState(initial);
  const [systemUser, setSystemUser] = React.useState("root");
  const device = data.devices.find((item) => item.id === deviceId) || data.devices[0];
  const [sshId, setSSHId] = React.useState(device?.remote_access_id || "");
  const [message, setMessage] = React.useState("");
  const [saving, setSaving] = React.useState(false);
  React.useEffect(() => { setSSHId(device?.remote_access_id || ""); }, [device?.id, device?.remote_access_id]);
  function openShellHub() { window.open(data.platform.remote_access_url, "_blank", "noopener,noreferrer"); }
  if (!device) return <>
    <section className="remoteHero">
      <div className="remoteHeroIcon"><TerminalSquare size={26} /></div>
      <div><span className="overline">Included ShellHub gateway</span><h2>Remote access is ready for setup</h2><p>Create the ShellHub administrator and namespace, then enroll your first fleet device.</p></div>
      <Badge value="included" />
    </section>
    <Panel title="ShellHub administration" subtitle="One-time setup for the remote-access service included with Soul Room">
      <div className="emptyAction"><p>Open the local ShellHub portal, complete its setup wizard, and keep the generated tenant ID for the device agent.</p><button className="button primary" onClick={openShellHub}><ExternalLink size={16} /> Open ShellHub portal</button></div>
    </Panel>
  </>;
  async function saveMapping() {
    setSaving(true); setMessage("");
    try { const updated = await updateDevice(data.membership, device.id, { remote_access_id: sshId.trim() }); onData({ ...data, devices: data.devices.map((item) => item.id === updated.id ? updated : item) }); setMessage("ShellHub identity saved."); }
    catch (reason) { setMessage(reason instanceof Error ? reason.message : "ShellHub identity could not be saved."); }
    finally { setSaving(false); }
  }
  async function connect(copyOnly = false) {
    setMessage("");
    try {
      const access = await createRemoteAccess(data.membership, device.id, systemUser);
      if (copyOnly) { await navigator.clipboard.writeText(access.ssh_command); setMessage("SSH command copied."); }
      else window.open(access.launch_url, "_blank", "noopener,noreferrer");
    } catch (reason) { setMessage(reason instanceof Error ? reason.message : "Remote access could not be opened."); }
  }
  return <>
    <section className="remoteHero">
      <div className="remoteHeroIcon"><TerminalSquare size={26} /></div>
      <div><span className="overline">Included ShellHub gateway</span><h2>Audited remote maintenance</h2><p>Device connections stay outbound-only; the bundled ShellHub service owns SSH keys, firewall policy, and session records.</p></div>
      <button className="button secondary" onClick={openShellHub}><ExternalLink size={16} /> ShellHub portal</button>
    </section>
    <section className="remoteLayout">
      <Panel title="Open a terminal" subtitle="Select an online device and its existing Linux user">
        <div className="remoteForm">
          <label>Device<select value={device.id} onChange={(event) => { setDeviceId(event.target.value); sessionStorage.setItem("soul_room_remote_device", event.target.value); }}>{data.devices.map((item) => <option value={item.id} key={item.id}>{item.display_name} - {item.presence}</option>)}</select></label>
          <label>Linux user<input value={systemUser} onChange={(event) => setSystemUser(event.target.value)} placeholder="root" /></label>
          <div className="remoteTarget"><div><span>ShellHub SSHID</span><strong>{device.remote_access_id || "Not mapped"}</strong></div><Badge value={device.presence} /></div>
          {message && <div className="inlineNotice">{message}</div>}
          <div className="remoteActions"><button className="button primary" disabled={!data.platform.remote_access_configured || !device.remote_access_id || device.presence !== "connected"} onClick={() => connect(false)}><TerminalSquare size={16} /> Open web terminal</button><button className="button secondary" disabled={!data.platform.remote_access_configured || !device.remote_access_id} onClick={() => connect(true)}><Copy size={16} /> Copy SSH command</button></div>
        </div>
      </Panel>
      <Panel title="Device mapping" subtitle="Associate this fleet record with the SSHID shown by ShellHub">
        <div className="remoteForm"><label>ShellHub SSHID<input value={sshId} onChange={(event) => setSSHId(event.target.value)} placeholder="namespace.device@your-shellhub-host" /></label><button className="button successButton" onClick={saveMapping} disabled={saving || !sshId.trim()}><Save size={16} />{saving ? "Saving..." : "Save mapping"}</button><div className="remoteChecklist"><div className={device.presence === "connected" ? "done" : ""}><Check size={15} /><span>Fleet agent online</span></div><div className={Boolean(device.remote_access_id) ? "done" : ""}><Check size={15} /><span>ShellHub device accepted</span></div><div className={data.platform.remote_access_managed ? "done" : ""}><Check size={15} /><span>Local ShellHub service included</span></div></div></div>
      </Panel>
    </section>
  </>;
}

function Deployments({ data }: { data: FleetData }) {
  const [filter, setFilter] = React.useState<"all" | "active" | "completed">("all");
  const activeStates = ["pending", "active", "running", "in_progress", "paused"];
  const rows = data.deployments.filter((deployment) => filter === "all" || (filter === "active" ? activeStates.includes(recordText(deployment, "state").toLowerCase()) : !activeStates.includes(recordText(deployment, "state").toLowerCase())));
  return <><Toolbar><div className="segmented" aria-label="Deployment state filter"><button className={filter === "all" ? "active" : ""} onClick={() => setFilter("all")}>All <span>{data.deployments.length}</span></button><button className={filter === "active" ? "active" : ""} onClick={() => setFilter("active")}>Active</button><button className={filter === "completed" ? "active" : ""} onClick={() => setFilter("completed")}>Completed</button></div></Toolbar><Panel title="Application rollouts" subtitle="Only deployments created through the control plane appear here"><SimpleTable headers={["Deployment", "Target", "Progress", "State", "Created"]} rows={rows.map((deployment) => [recordText(deployment, "name"), recordText(deployment, "target"), recordText(deployment, "progress"), <Badge value={recordText(deployment, "state")} />, recordText(deployment, "created_at")])} empty={`No ${filter === "all" ? "" : `${filter} `}deployments.`} /></Panel></>;
}

function OTA({ data, onData }: { data: FleetData; onData: (data: FleetData) => void }) {
  const [showCreate, setShowCreate] = React.useState(false);
  const [name, setName] = React.useState(""); const [version, setVersion] = React.useState(""); const [product, setProduct] = React.useState("");
  const [architecture, setArchitecture] = React.useState("arm64"); const [adapter, setAdapter] = React.useState("mender"); const [customAdapter, setCustomAdapter] = React.useState("");
  const [artifactUrl, setArtifactURL] = React.useState(""); const [digest, setDigest] = React.useState(""); const [signature, setSignature] = React.useState(""); const [signingKeyID, setSigningKeyID] = React.useState("");
  const [artifactSize, setArtifactSize] = React.useState(""); const [compatibleFrom, setCompatibleFrom] = React.useState("");
  const [flatpakRef, setFlatpakRef] = React.useState(""); const [flatpakRemote, setFlatpakRemote] = React.useState("flathub"); const [flatpakCommit, setFlatpakCommit] = React.useState("");
  const [flatpakRepositoryURL, setFlatpakRepositoryURL] = React.useState("");
  const [canary, setCanary] = React.useState(10); const [targets, setTargets] = React.useState<string[]>([]); const [error, setError] = React.useState(""); const [saving, setSaving] = React.useState(false);
  const selectedAdapter = adapter === "custom" ? customAdapter.trim().toLowerCase() : adapter;
  const isFlatpak = selectedAdapter === "flatpak";
  const capable = data.devices.filter((device) => selectedAdapter && device.capabilities?.includes(`ota:${selectedAdapter}`));
  async function create() {
    setError("");
    if (!name || !version || !targets.length) { setError("Enter a campaign name and version, then select at least one target."); return; }
    if (isFlatpak && (!/^[A-Za-z0-9][A-Za-z0-9._/-]{2,199}$/.test(flatpakRef) || !/^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/.test(flatpakRemote))) { setError("Enter a valid Flatpak application reference and configured remote."); return; }
    if (isFlatpak && flatpakCommit && !/^[a-f0-9]{64}$/i.test(flatpakCommit)) { setError("Flatpak commit must be a 64-character OSTree commit."); return; }
    if (isFlatpak && flatpakRepositoryURL && !/^https:\/\/[^/]+\/.+\.flatpakrepo$/i.test(flatpakRepositoryURL)) { setError("Repository URL must use HTTPS and point to a .flatpakrepo descriptor."); return; }
    if (!selectedAdapter || !/^[a-z][a-z0-9-]{1,31}$/.test(selectedAdapter)) { setError("Select an update mechanism or enter a valid installed plugin name."); return; }
    if (!isFlatpak && (!artifactUrl || !digest || !signature || !signingKeyID || !product)) { setError("Complete the signed operating-system update metadata."); return; }
    if (!isFlatpak && !/^https:\/\//i.test(artifactUrl)) { setError("Artifact URL must use HTTPS."); return; }
    if (!isFlatpak && !/^[a-f0-9]{64}$/i.test(digest)) { setError("Digest must be a 64-character SHA-256 value."); return; }
    setSaving(true);
    try { const campaign = await createOTACampaign(data.membership, { name, artifact_id: "", artifact_url: isFlatpak ? "" : artifactUrl, artifact_size: isFlatpak || !artifactSize ? 0 : Number(artifactSize), version, product: isFlatpak ? "flatpak-application" : product, architecture: isFlatpak ? "" : architecture, digest: isFlatpak ? "" : digest, signature: isFlatpak ? "" : signature, signing_key_id: isFlatpak ? "" : signingKeyID, compatible_from: compatibleFrom.split(",").map((value) => value.trim()).filter(Boolean), adapter: selectedAdapter, flatpak_ref: flatpakRef, flatpak_remote: flatpakRemote, flatpak_commit: flatpakCommit, flatpak_repository_url: flatpakRepositoryURL, target_device_ids: targets, canary_percent: canary }); onData({ ...data, otaCampaigns: [campaign, ...data.otaCampaigns] }); setShowCreate(false); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Campaign could not be created."); }
    finally { setSaving(false); }
  }
  async function promote(campaignId: string) {
    try { const campaign = await promoteOTACampaign(data.membership, campaignId); onData({ ...data, otaCampaigns: data.otaCampaigns.map((item) => item.id === campaign.id ? campaign : item) }); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "The campaign could not be promoted."); }
  }
  return <>
    <section className="otaReadiness"><div><span className="otaIcon"><ShieldCheck size={22} /></span><span><strong>Controlled update delivery</strong><small>Signed updates through the mechanism installed on each device</small></span></div><div><strong>{capable.length}</strong><span>{selectedAdapter || "adapter"} ready</span></div><div><strong>{data.otaCampaigns.length}</strong><span>campaigns</span></div><button className="button primary" onClick={() => setShowCreate(true)} disabled={!data.devices.length}><Plus size={16} /> Create campaign</button></section>
    <div className="safetyBanner"><ShieldCheck size={21} /><div><strong>Start small, then continue with confidence</strong><span>The pilot group receives the update first. Only devices advertising the selected adapter can be targeted, and the remaining fleet waits for approval.</span></div></div>
    {error && !showCreate && <div className="inlineError"><AlertTriangle size={17} />{error}<button onClick={() => setError("")}><X size={15} /></button></div>}
    <Panel title="Update campaigns" subtitle="Operating-system releases and Flatpak application rollouts"><SimpleTable headers={["Campaign", "Targets", "Type", "Release", "Pilot group", "State", "Created", "Action"]} rows={data.otaCampaigns.map((campaign) => [campaign.name, campaign.target_device_ids.length, campaign.adapter === "flatpak" ? "Flatpak app" : `${campaign.adapter} OS`, campaign.adapter === "flatpak" ? campaign.flatpak_ref || campaign.version : `${campaign.version} / ${campaign.architecture}`, `${campaign.canary_percent}%`, <Badge value={formatCampaignState(campaign.state)} />, formatDate(campaign.created_at), campaign.state === "canary_complete" ? <button className="button successButton compactButton" onClick={() => promote(campaign.id)}><Play size={14} /> Continue rollout</button> : <span className="tableMuted">-</span>])} empty="Create an update campaign to begin a controlled rollout." /></Panel>
    {showCreate && <Modal title="Create update campaign" onClose={() => setShowCreate(false)}><div className="modalForm otaForm"><div className="formGrid"><label>Campaign name<input value={name} onChange={(event) => setName(event.target.value)} placeholder={isFlatpak ? "Kiosk application 1.4" : "August security release"} /></label><label>Release version<input value={version} onChange={(event) => setVersion(event.target.value)} placeholder="2026.08.1" /></label><label>Update mechanism<select value={adapter} onChange={(event) => { setAdapter(event.target.value); setTargets([]); }}><option value="mender">Mender</option><option value="rauc">RAUC</option><option value="ostree">OSTree</option><option value="swupdate">SWUpdate</option><option value="flatpak">Flatpak application</option><option value="custom">Installed plugin</option></select></label><label>Pilot group (%)<div className="rangeField"><input type="range" min="1" max="100" value={canary} onChange={(event) => setCanary(Number(event.target.value))} /><output>{canary}%</output></div><span className="formHint">This percentage updates first. Continue only after those devices succeed.</span></label>{adapter === "custom" && <label>Plugin name<input value={customAdapter} onChange={(event) => { setCustomAdapter(event.target.value); setTargets([]); }} placeholder="my-updater" /></label>}{!isFlatpak && <><label>Device product<input value={product} onChange={(event) => setProduct(event.target.value)} placeholder="embedded-linux" /></label><label>Architecture<select value={architecture} onChange={(event) => { setArchitecture(event.target.value); setTargets([]); }}><option value="arm64">64-bit ARM (arm64 / aarch64)</option><option value="armv7">32-bit ARM (armv7 / armhf)</option><option value="amd64">x86-64 (amd64)</option></select></label></>}</div>{isFlatpak ? <><div className="formSectionLabel">Flatpak application</div><label>Application reference<input value={flatpakRef} onChange={(event) => setFlatpakRef(event.target.value.trim())} placeholder="com.example.Kiosk" /></label><div className="formGrid"><label>Remote name<input value={flatpakRemote} onChange={(event) => setFlatpakRemote(event.target.value.trim())} placeholder="factory" /></label><label>OSTree commit (optional)<input value={flatpakCommit} onChange={(event) => setFlatpakCommit(event.target.value.trim())} placeholder="Pin an exact 64-character commit" /></label></div><label>Repository descriptor URL (optional)<input type="url" value={flatpakRepositoryURL} onChange={(event) => setFlatpakRepositoryURL(event.target.value.trim())} placeholder="https://updates.example.com/factory.flatpakrepo" /></label></> : <><label>HTTPS artifact URL<input value={artifactUrl} onChange={(event) => setArtifactURL(event.target.value.trim())} placeholder="https://releases.example.com/device-v2.bundle" /></label><div className="formGrid"><label>Artifact size in bytes (optional)<input type="number" min="0" value={artifactSize} onChange={(event) => setArtifactSize(event.target.value)} /></label><label>Allowed current versions (optional)<input value={compatibleFrom} onChange={(event) => setCompatibleFrom(event.target.value)} placeholder="1.4.0, 1.4.1" /></label></div><label>SHA-256 digest<input value={digest} onChange={(event) => setDigest(event.target.value.trim())} placeholder="64 hexadecimal characters" /></label><div className="formGrid"><label>Signing key ID<input value={signingKeyID} onChange={(event) => setSigningKeyID(event.target.value.trim())} placeholder="production-2026" /></label><label>Detached Ed25519 signature<input value={signature} onChange={(event) => setSignature(event.target.value.trim())} placeholder="Base64 signature of the lowercase digest" /></label></div></>}<fieldset className="targetPicker"><legend>Compatible target devices</legend>{data.devices.filter((device) => isFlatpak || !architecture || !device.architecture || normalizeArchitecture(device.architecture) === architecture).map((device) => { const ready = Boolean(selectedAdapter && device.capabilities?.includes(`ota:${selectedAdapter}`)); return <label key={device.id} className={!ready ? "disabledTarget" : ""}><input type="checkbox" disabled={!ready} checked={targets.includes(device.id)} onChange={(event) => setTargets(event.target.checked ? [...targets, device.id] : targets.filter((id) => id !== device.id))} /><span><strong>{device.display_name}</strong><small>{device.presence} - {ready ? `${selectedAdapter} ready` : `${selectedAdapter || "adapter"} is not installed`}</small></span></label>; })}</fieldset>{error && <div className="formError"><AlertTriangle size={17} />{error}</div>}<div className="modalActions"><button className="button secondary" onClick={() => setShowCreate(false)}>Cancel</button><button className="button primary" disabled={saving || !capable.length} onClick={create}>{saving ? <LoaderCircle className="spin" size={16} /> : <CloudCog size={16} />} Queue pilot update</button></div></div></Modal>}
  </>;
}

function Alerts({ data }: { data: FleetData }) {
  const alerts = deriveAlerts(data.devices);
  const critical = alerts.filter((alert) => alert.severity === "critical").length;
  return <><section className="metricStrip"><Metric label="Active" value={String(alerts.length)} note="Derived from live device state" tone={alerts.length ? "red" : "green"} icon={<Bell size={18} />} /><Metric label="Critical" value={String(critical)} note="Offline devices" tone={critical ? "red" : "green"} icon={<AlertTriangle size={18} />} /><Metric label="Warnings" value={String(alerts.length - critical)} note="Health and certificate checks" tone="amber" icon={<ClipboardCheck size={18} />} /><Metric label="Devices checked" value={String(data.devices.length)} note="No synthetic alert records" tone="blue" icon={<Check size={18} />} /></section><Panel title="Active alerts" subtitle="Calculated from the latest accepted device reports"><SimpleTable headers={["Severity", "Issue", "Source", "Detail"]} rows={alerts.map((alert) => [<Badge value={alert.severity} />, alert.title, alert.device, alert.detail])} empty="No device issues are active." /></Panel></>;
}

function Profiles({ data }: { data: FleetData }) {
  const assignments = Array.from(new Set(data.devices.map((device) => device.profile_id || "default"))).map((id) => ({ id, devices: data.devices.filter((device) => (device.profile_id || "default") === id).length }));
  return <Panel title="Profile assignments" subtitle="Profile identifiers reported by enrolled devices"><SimpleTable headers={["Profile", "Assigned devices"]} rows={assignments.map((profile) => [<div className="primaryCell"><div className="profileIcon"><Settings2 size={17} /></div><strong>{profile.id}</strong></div>, profile.devices])} empty="No profile assignments exist until a device is enrolled." /></Panel>;
}

type PackageAdvisory = { name: string; installed_version: string; fixed_version?: string; highest_severity: string; critical: number; high: number; medium: number; low: number; advisory_ids?: string[] };
type VulnerabilityReport = { scanner: string; scanned_at: string; total: number; counts: Record<string, number>; packages: PackageAdvisory[] };

function Applications({ data, onData }: { data: FleetData; onData: (data: FleetData) => void }) {
  const [tab, setTab] = React.useState<"packages" | "managed">("packages");
  const [deviceId, setDeviceId] = React.useState(data.devices[0]?.id || "");
  const [packageSearch, setPackageSearch] = React.useState("");
  const [visible, setVisible] = React.useState(40);
  const [scanning, setScanning] = React.useState(false);
  const [scanError, setScanError] = React.useState("");
  React.useEffect(() => { if (!deviceId && data.devices[0]) setDeviceId(data.devices[0].id); }, [data.devices, deviceId]);
  const device = data.devices.find((item) => item.id === deviceId) || data.devices[0];
  const packages = device ? (data.inventory[device.id]?.installed_packages || []) : [];
  const scanJob = device ? data.jobs.filter((job) => job.device_id === device.id && job.type === "scan_vulnerabilities" && job.state === "succeeded").sort((left, right) => new Date(right.created_at).getTime() - new Date(left.created_at).getTime())[0] : undefined;
  const report = scanJob ? vulnerabilityReport(scanJob) : null;
  const risks = new Map((report?.packages || []).map((item) => [item.name, item]));
  const filtered = packages.filter((item) => `${item.name} ${item.version} ${item.architecture || ""}`.toLowerCase().includes(packageSearch.toLowerCase()));
  const canScan = Boolean(device?.capabilities?.includes("security:trivy"));
  async function scan() {
    if (!device) return;
    setScanning(true); setScanError("");
    try { const job = await createJob(data.membership, device.id, "scan_vulnerabilities"); onData({ ...data, jobs: [job, ...data.jobs] }); }
    catch (reason) { setScanError(reason instanceof Error ? reason.message : "The vulnerability scan could not be queued."); }
    finally { setScanning(false); }
  }
  return <>
    <Toolbar><div className="segmented"><button className={tab === "packages" ? "active" : ""} onClick={() => setTab("packages")}>Device packages</button><button className={tab === "managed" ? "active" : ""} onClick={() => setTab("managed")}>Managed apps</button></div>{tab === "packages" && <><label className="selectLabel">Device<select value={device?.id || ""} onChange={(event) => { setDeviceId(event.target.value); setVisible(40); }}>{data.devices.map((item) => <option value={item.id} key={item.id}>{item.display_name}</option>)}</select></label><button className="button warningButton" onClick={scan} disabled={!device || !canScan || scanning || device.presence !== "connected"}>{scanning ? <LoaderCircle className="spin" size={16} /> : <ShieldCheck size={16} />} Scan advisories</button></>}</Toolbar>
    {tab === "managed" ? <><Panel title="Managed applications" subtitle="Validated application records from the control plane"><SimpleTable headers={["Application", "Version", "Target", "State"]} rows={data.applications.map((app) => [recordText(app, "name"), recordText(app, "version"), recordText(app, "target"), <Badge value={recordText(app, "state")} />])} empty="No managed applications have been added." /></Panel><div className="policyFoot"><ShieldCheck size={18} /><span>Host networking, privileged containers, Docker socket mounts, and unrestricted host paths are rejected by default.</span></div></> : !device ? <EmptySection icon={<PackageSearch size={28} />} title="No package inventory" text="Enroll a device to see its installed software packages." /> : <>
      <section className="packageSummary"><Metric label="Installed packages" value={String(packages.length)} note={`Reported by ${device.display_name}`} tone="blue" icon={<Boxes size={18} />} /><Metric label="Critical advisories" value={report ? String(report.counts.CRITICAL || 0) : "—"} note={report ? "Review before production" : "Run a device scan"} tone={report?.counts.CRITICAL ? "red" : "green"} icon={<AlertTriangle size={18} />} /><Metric label="High advisories" value={report ? String(report.counts.HIGH || 0) : "—"} note={report ? "Evidence from Trivy" : "No scan result yet"} tone={report?.counts.HIGH ? "amber" : "green"} icon={<ShieldCheck size={18} />} /><Metric label="Last scan" value={report ? relativeTime(report.scanned_at) : "Not run"} note={report ? `${report.total} advisory matches` : canScan ? "Scanner is ready" : "Trivy not reported"} tone="cyan" icon={<Clock3 size={18} />} /></section>
      {!canScan && <div className="safetyBanner warning"><TriangleAlert size={20} /><div><strong>Package inventory is available; advisory scanning is not</strong><span>Install Trivy on this device and restart the Soul Room agent to enable evidence-backed CVE matching.</span></div></div>}
      {scanError && <div className="formError"><AlertTriangle size={17} />{scanError}</div>}
      <Panel title="Software package inventory" subtitle="Real packages reported by the selected device; advisory matches come from the latest completed scan" action={<label className="packageFilter"><Search size={15} /><input value={packageSearch} onChange={(event) => { setPackageSearch(event.target.value); setVisible(40); }} placeholder="Filter packages" aria-label="Filter installed packages" /></label>}>
        {filtered.length ? <><div className="packageGrid">{filtered.slice(0, visible).map((item) => { const risk = risks.get(item.name); return <article className="packageTile" key={`${item.name}-${item.version}`}><PackageBrandIcon name={item.name} /><div><strong title={item.name}>{item.name}</strong><span title={item.version}>{item.version}</span></div><Badge value={risk ? risk.highest_severity.toLowerCase() : report ? "clear" : "not scanned"} />{risk && <small>{risk.critical + risk.high + risk.medium + risk.low} advisories{risk.fixed_version ? ` · fix ${risk.fixed_version}` : ""}</small>}</article>; })}</div>{visible < filtered.length && <div className="showMore"><button className="button secondary" onClick={() => setVisible(visible + 40)}>Show 40 more</button><span>{visible} of {filtered.length}</span></div>}</> : <EmptyState text={packages.length ? "No packages match this filter." : "This agent has not reported package inventory yet."} />}
      </Panel>
      <div className="policyFoot"><ShieldCheck size={18} /><span>Advisory matches are guidance, not proof of exploitability. Review package use, exposure, and available fixes before making a production decision.</span></div>
    </>}
  </>;
}

function Artifacts({ data }: { data: FleetData }) {
  return <><Toolbar><div className="toolbarSummary"><strong>{formatBytes(data.artifacts.reduce((sum, artifact) => sum + artifact.size_bytes, 0))} stored</strong><span>Content-addressed and immutable after upload</span></div><button className="button primary" disabled title="Object upload is not configured yet"><UploadCloud size={16} /> Upload artifact</button></Toolbar><Panel title="Artifact registry" subtitle="Files are identified by digest and verified before delivery"><SimpleTable headers={["Artifact", "Version", "Type", "Size", "Digest", "Signature", "Uploaded"]} rows={data.artifacts.map((artifact) => [<div className="primaryCell"><Archive size={17} /><strong>{artifact.name}</strong></div>, artifact.version, artifact.content_type, formatBytes(artifact.size_bytes), <code>{artifact.digest.slice(0, 22)}…</code>, <Badge value={artifact.signature ? "verified" : "unsigned"} />, formatDate(artifact.created_at)])} empty="No artifacts are registered." /></Panel></>;
}

function Enrollment({ data, onData }: ViewProps) {
  const [profile, setProfile] = React.useState("");
  const [ttl, setTtl] = React.useState("24h");
  const [created, setCreated] = React.useState<EnrollmentToken | null>(null);
  const [saving, setSaving] = React.useState(false);
  async function create() { setSaving(true); try { const token = await createEnrollmentToken(data.membership, profile, ttl); setCreated(token); onData({ ...data, enrollmentTokens: [token, ...data.enrollmentTokens] }); } finally { setSaving(false); } }
  return <section className="enrollmentLayout"><Panel title="Create enrollment token" subtitle="A token can be used once and expires automatically"><div className="enrollForm"><label>Profile ID (optional)<input value={profile} onChange={(event) => setProfile(event.target.value)} placeholder="embedded-linux-arm64" /></label><label>Valid for<select value={ttl} onChange={(event) => setTtl(event.target.value)}><option value="1h">1 hour</option><option value="24h">24 hours</option><option value="168h">7 days</option></select></label><button className="button successButton" onClick={create} disabled={saving}>{saving ? <LoaderCircle className="spin" size={16} /> : <KeyRound size={16} />} Generate token</button>{created?.token && <div className="tokenResult"><span>Copy this token now. It is shown only once.</span><code>{created.token}</code><button className="button infoButton" onClick={() => navigator.clipboard.writeText(created.token || "")}><ClipboardCheck size={16} /> Copy token</button></div>}</div></Panel><Panel title="Connect the device" subtitle="The embedded Linux agent needs two values"><ol className="steps"><li><span>1</span><div><strong>Download the development CA</strong><p>Install it on the device at <code>/etc/edge-agent/ca.pem</code>.</p><a className="button infoButton" href="/api/v1/dev-ca" download><Download size={16} /> Download CA</a></div></li><li><span>2</span><div><strong>Set the gateway endpoint</strong><p>Use <code>{data.platform.device_gateway_endpoint}</code> in the agent configuration.</p></div></li><li><span>3</span><div><strong>Enroll once</strong><p>Start the agent with the token. Its private key never leaves the device.</p></div></li></ol></Panel><Panel title="Recent tokens" subtitle="Token values are not stored after creation"><SimpleTable headers={["ID", "Profile", "Usage", "Expires", "Created"]} rows={data.enrollmentTokens.map((token) => [shortId(token.id), token.profile_id || "Default", `${token.used_count} / ${token.max_devices}`, formatDate(token.expires_at), formatDate(token.created_at)])} empty="No enrollment tokens have been created." /></Panel></section>;
}

function Audit({ data, search }: { data: FleetData; search: string }) {
  const [category, setCategory] = React.useState<"all" | "security" | "device" | "deployment">("all");
  const categoryTerms = { security: ["auth", "enroll", "token", "user", "role", "certificate"], device: ["device", "job", "gateway", "telemetry", "inventory"], deployment: ["deployment", "ota", "artifact", "application"] };
  const rows = data.audit.filter((event) => {
    const haystack = `${event.action} ${event.resource_type}`.toLowerCase();
    return haystack.includes(search.toLowerCase()) && (category === "all" || categoryTerms[category].some((term) => haystack.includes(term)));
  });
  return <><Toolbar><div className="segmented" aria-label="Audit category filter"><button className={category === "all" ? "active" : ""} onClick={() => setCategory("all")}>All activity</button><button className={category === "security" ? "active" : ""} onClick={() => setCategory("security")}>Security</button><button className={category === "device" ? "active" : ""} onClick={() => setCategory("device")}>Device operations</button><button className={category === "deployment" ? "active" : ""} onClick={() => setCategory("deployment")}>Deployments</button></div></Toolbar><Panel title="Audit events" subtitle="Append-oriented operator and system activity"><SimpleTable headers={["Time", "Action", "Resource", "Actor", "Result", "Event ID"]} rows={rows.map((event) => [formatDate(event.created_at), event.action, `${event.resource_type}${event.resource_id ? ` - ${shortId(event.resource_id)}` : ""}`, shortId(event.actor_id || "system"), <Badge value={event.result} />, <code>{shortId(event.id)}</code>])} empty="No events match this category and search." /></Panel></>;
}

function Users({ data, onData }: { data: FleetData; onData: (data: FleetData) => void }) {
  const [tab, setTab] = React.useState<"users" | "roles">("users");
  const [showCreate, setShowCreate] = React.useState(false);
  const [email, setEmail] = React.useState("");
  const [role, setRole] = React.useState("fleet_operator");
  const [password, setPassword] = React.useState(() => generateTemporaryPassword());
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState("");
  const canManage = ["platform_administrator", "organization_owner", "organization_administrator"].includes(data.membership.role);
  const assignableRoles = Object.keys(data.roles).filter((name) => !["platform_administrator", "organization_owner"].includes(name));
  async function addMember() {
    setError("");
    if (!email.trim() || password.length < 12) { setError("Enter a valid email and a temporary password of at least 12 characters."); return; }
    setSaving(true);
    try {
      const user = await createTeamUser(data.membership, email.trim(), password, role);
      onData({ ...data, users: [...data.users, user].sort((left, right) => left.email.localeCompare(right.email)) });
      setShowCreate(false); setEmail(""); setPassword(generateTemporaryPassword());
    } catch (reason) { setError(reason instanceof Error ? reason.message : "The team member could not be added."); }
    finally { setSaving(false); }
  }
  return <><Toolbar><div className="segmented"><button className={tab === "users" ? "active" : ""} onClick={() => setTab("users")}>Team members</button><button className={tab === "roles" ? "active" : ""} onClick={() => setTab("roles")}>Roles & permissions</button></div>{canManage && <button className="button primary" onClick={() => { setError(""); setShowCreate(true); }}><Plus size={16} /> Add team member</button>}</Toolbar>{tab === "users" ? <Panel title="Organization access" subtitle="Accounts and roles are scoped to this organization"><SimpleTable headers={["User", "Role", "MFA", "Status", "Joined"]} rows={data.users.map((user) => [<div className="primaryCell"><div className="avatar smallAvatar">{user.email.slice(0, 2).toUpperCase()}</div><strong>{user.email}</strong></div>, (user.role || "unassigned").replaceAll("_", " "), <Badge value={user.mfa_enabled ? "enabled" : "not enabled"} />, <Badge value={user.activated_at ? "active" : "pending"} />, formatDate(user.created_at)])} /></Panel> : <Panel title="Built-in roles" subtitle="Permissions are explicit and enforced by the control API"><div className="roleList">{Object.entries(data.roles).map(([roleName, permissions]) => <div className="roleRow" key={roleName}><div><strong>{roleName.replaceAll("_", " ")}</strong><span>{permissions.length} permissions</span></div><div className="permissionList">{permissions.map((permission) => <code key={permission}>{permission}</code>)}</div></div>)}</div></Panel>}{showCreate && <Modal title="Add team member" onClose={() => setShowCreate(false)}><div className="modalForm"><label>Email address<input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="operator@example.com" autoComplete="off" /></label><label>Role<select value={role} onChange={(event) => setRole(event.target.value)}>{assignableRoles.map((name) => <option key={name} value={name}>{name.replaceAll("_", " ")}</option>)}</select></label><label>Temporary password<div className="passwordField"><input value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" /><button className="iconButton" onClick={() => navigator.clipboard.writeText(password)} title="Copy temporary password"><Copy size={16} /></button><button className="iconButton" onClick={() => setPassword(generateTemporaryPassword())} title="Generate another password"><RefreshCw size={16} /></button></div></label><div className="notice"><ShieldCheck size={18} /><div><strong>Share once through a secure channel</strong><span>The password is hashed before storage and cannot be retrieved later.</span></div></div>{error && <div className="formError"><AlertTriangle size={17} />{error}</div>}<div className="modalActions"><button className="button secondary" onClick={() => setShowCreate(false)}>Cancel</button><button className="button primary" disabled={saving} onClick={addMember}>{saving ? <LoaderCircle className="spin" size={16} /> : <UsersRound size={16} />} Add member</button></div></div></Modal>}</>;
}

function Health({ data }: { data: FleetData }) {
  const [deviceId, setDeviceId] = React.useState(data.devices[0]?.id || "");
  React.useEffect(() => {
    if (!deviceId && data.devices[0]) setDeviceId(data.devices[0].id);
  }, [data.devices, deviceId]);
  if (data.devices.length === 0) return <EmptySection icon={<Activity size={28} />} title="No device health data" text="Start an enrolled agent to populate the system health charts." />;
  const device = data.devices.find((item) => item.id === deviceId) || data.devices[0];
  const metrics = data.metrics.filter((metric) => metric.device_id === device.id);
  const deviceData = { ...data, metrics };
  const facts = data.inventory[device.id] || {};
  return <>
    <Toolbar><label className="selectLabel">Device<select value={device.id} onChange={(event) => setDeviceId(event.target.value)}>{data.devices.map((item) => <option value={item.id} key={item.id}>{item.display_name}</option>)}</select></label><div className="healthIdentity"><Badge value={device.presence} /><span>{device.hardware_model || device.architecture}</span></div></Toolbar>
    <section className="metricStrip telemetryMetrics"><Metric label="CPU" value={formatPercent(latestMatching(metrics, "cpu.utilization"))} note="Current utilization" tone="cyan" icon={<Cpu size={18} />} /><Metric label="Memory" value={formatPercent(latestMatching(metrics, "memory.utilization"))} note="Current utilization" tone="violet" icon={<HardDrive size={18} />} /><Metric label="Disk" value={formatPercent(latestMatching(metrics, "filesystem.utilization"))} note="Root filesystem" tone="amber" icon={<Database size={18} />} /><Metric label="Temperature" value={formatTemperature(latestMatching(metrics, "temperature"))} note={device.presence === "connected" ? "Thermal sensor" : `Last seen ${relativeTime(device.last_seen_at)}`} tone="red" icon={<Activity size={18} />} /></section>
    <section className="healthCharts">
      <Panel title="Resource utilization" subtitle="CPU, memory, disk, and temperature over time"><div className="healthChart">{metrics.length ? <TelemetryChart data={telemetryChart(deviceData)} /> : <ChartEmpty text="Waiting for resource telemetry" />}</div></Panel>
      <Panel title="Current resource load" subtitle="Latest utilization reported by the device"><div className="healthChart"><ResourceBars metrics={metrics} /></div></Panel>
      <Panel title="Network counters" subtitle="Total bytes reported across active interfaces"><div className="healthChart"><NetworkChart metrics={metrics} /></div></Panel>
      <Panel title="Installed capacity" subtitle="Physical memory and root filesystem capacity"><div className="healthChart"><CapacityChart memory={Number(facts.memory_total_bytes || latestMatching(metrics, "memory.total") || 0)} storage={Number(facts.storage_total_bytes || latestMatching(metrics, "filesystem.total") || 0)} /></div></Panel>
      <Panel title="Fleet availability" subtitle="Current heartbeat state for all enrolled devices"><div className="healthChart"><PresenceChart connected={data.devices.filter((item) => item.presence === "connected").length} total={data.devices.length} /></div></Panel>
    </section>
  </>;
}

function OnPrem({ data }: { data: FleetData }) {
  return <><div className="adminSummary"><div><span className="overline">Installation</span><strong>Soul Room Local</strong><p>Single-node Docker Compose - Version 0.1.0</p></div><Badge value="running" /></div><section className="adminGrid"><AdminAction icon={<Archive size={20} />} title="Backup & restore" detail="Use the Compose maintenance profile for portable backups." primary="Backup" meta="No synthetic backup history" /><AdminAction icon={<ShieldCheck size={20} />} title="Certificates" detail="Download the CA trusted by enrolled devices." primary="Download CA" href="/api/v1/dev-ca" meta={`${data.devices.length} issued device identities`} /><AdminAction icon={<HardDrive size={20} />} title="Fleet records" detail="Live records currently loaded from the platform store." primary="Review" meta={`${data.devices.length} devices - ${data.metrics.length} samples`} /><AdminAction icon={<PackageCheck size={20} />} title="Platform version" detail="Current local control-plane release." primary="Version" meta="0.1.0" /></section><Panel title="Environment endpoints" subtitle="Addresses exposed by the local Compose stack"><DetailList items={[["Web console", window.location.origin], ["Control API", `${window.location.protocol}//${window.location.hostname}:8080`], ["Device gateway", data.platform.device_gateway_endpoint], ["MinIO console", `http://${window.location.hostname}:9001`], ["Mailpit", `http://${window.location.hostname}:8025`]]} /></Panel></>;
}

function DeviceTable({ devices, compact = false, onSelect }: { devices: Device[]; compact?: boolean; onSelect?: (device: Device) => void }) {
  return <div className="tableWrap"><table><thead><tr><th>Device</th><th>Presence</th><th>Health</th><th>Profile</th><th>Platform</th><th>Last seen</th>{!compact && <th>Update</th>}</tr></thead><tbody>{devices.map((device) => <tr key={device.id} onClick={() => onSelect?.(device)} className={onSelect ? "clickable" : ""}><td><div className="primaryCell"><div className="deviceMini"><Cpu size={16} /></div><div><strong>{device.display_name}</strong><span>{device.serial}</span></div></div></td><td><Badge value={device.presence} /></td><td><Badge value={device.health_state || "unknown"} /></td><td>{device.profile_id || "Default"}</td><td><span>{device.os}</span><small className="subCell">{device.architecture}</small></td><td>{relativeTime(device.last_seen_at)}</td>{!compact && <td><Badge value={(device.update_state || "unknown").replaceAll("_", " ")} /></td>}</tr>)}</tbody></table>{devices.length === 0 && <EmptyState text="No devices match this view." />}</div>;
}

function Panel({ title, subtitle, action, children }: { title: string; subtitle?: string; action?: React.ReactNode; children: React.ReactNode }) {
  return <section className="panel"><header className="panelHeader"><div><h2>{title}</h2>{subtitle && <p>{subtitle}</p>}</div>{action}</header>{children}</section>;
}

function Toolbar({ children }: { children: React.ReactNode }) { return <div className="toolbar">{children}</div>; }

function Metric({ label, value, note, tone, icon }: { label: string; value: string; note: string; tone: string; icon: React.ReactNode }) { return <div className="metric"><div className={`metricIcon ${tone}`}>{icon}</div><span>{label}</span><strong>{value}</strong><small>{note}</small></div>; }

function Badge({ value }: { value: string }) { const tone = badgeTone(value); return <span className={`badge ${tone}`}><i />{value}</span>; }

function SimpleTable({ headers, rows, empty = "Nothing to show." }: { headers: string[]; rows: React.ReactNode[][]; empty?: string }) { return <div className="tableWrap"><table><thead><tr>{headers.map((header) => <th key={header}>{header}</th>)}</tr></thead><tbody>{rows.map((row, index) => <tr key={index}>{row.map((cell, cellIndex) => <td key={cellIndex}>{cell}</td>)}</tr>)}</tbody></table>{rows.length === 0 && <EmptyState text={empty} />}</div>; }

function EmptyState({ text }: { text: string }) { return <div className="emptyState"><FileClock size={22} /><span>{text}</span></div>; }

function EmptyFleet({ onEnroll }: { onEnroll: () => void }) {
  return <section className="emptyFleet"><div className="emptyFleetVisual"><RadioTower size={34} /><span className="signal one" /><span className="signal two" /></div><span className="overline">Ready for a real device</span><h2>Your fleet is empty</h2><p>Create a one-time token, enroll an embedded Linux device, and its actual inventory and telemetry will populate this workspace.</p><button className="button successButton" onClick={onEnroll}><Plus size={16} /> Enroll first device</button><div className="emptyFlow"><span><strong>1</strong> Create token</span><ChevronRight size={15} /><span><strong>2</strong> Start agent</span><ChevronRight size={15} /><span><strong>3</strong> Watch live data</span></div></section>;
}

function EmptySection({ icon, title, text }: { icon: React.ReactNode; title: string; text: string }) {
  return <section className="emptySection"><div className="emptySectionIcon">{icon}</div><h2>{title}</h2><p>{text}</p></section>;
}

function ChartEmpty({ text }: { text: string }) {
  return <div className="chartEmpty"><Activity size={22} /><span>{text}</span></div>;
}

function generateTemporaryPassword() {
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%";
  const bytes = crypto.getRandomValues(new Uint8Array(18));
  return Array.from(bytes, (value) => alphabet[value % alphabet.length]).join("");
}

function Attention({ severity, title, detail, action, onClick }: { severity: string; title: string; detail: string; action: string; onClick: () => void }) { return <div className="attention"><span className={`attentionMark ${severity}`} /><div><strong>{title}</strong><span>{detail}</span></div><button className="textButton" onClick={onClick}>{action}</button></div>; }

function Drawer({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) { return <div className="overlay" onMouseDown={onClose}><aside className="drawer" onMouseDown={(event) => event.stopPropagation()}><header><div><span className="overline">Device detail</span><h2>{title}</h2></div><button className="iconButton" onClick={onClose}><X size={18} /></button></header><div className="drawerBody">{children}</div></aside></div>; }

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) { return <div className="overlay centered" onMouseDown={onClose}><section className="modal" onMouseDown={(event) => event.stopPropagation()}><header><h2>{title}</h2><button className="iconButton" onClick={onClose}><X size={18} /></button></header>{children}</section></div>; }

function DetailList({ items }: { items: [string, React.ReactNode][] }) { return <dl className="detailList">{items.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value || "—"}</dd></div>)}</dl>; }

function StatRow({ label, value, detail }: { label: string; value: string; detail: string }) { return <div className="statRow"><div><strong>{label}</strong><span>{detail}</span></div><Badge value={value} /></div>; }

function Capacity({ label, value, detail }: { label: string; value: number; detail: string }) { return <div className="capacityRow"><div><strong>{label}</strong><span>{detail}</span></div><div className="progressTrack"><i style={{ width: `${value}%` }} /></div></div>; }

function AdminAction({ icon, title, detail, primary, meta, href }: { icon: React.ReactNode; title: string; detail: string; primary: string; meta: string; href?: string }) { const button = <>{icon}<div><strong>{title}</strong><span>{detail}</span></div><ChevronRight size={18} /></>; return <section className="adminAction">{href ? <a href={href} download>{button}</a> : <button disabled title={`${primary} action is not configured`}>{button}</button>}<small>{meta}</small></section>; }

function TelemetryChart({ data }: { data: { time: string; cpu?: number; memory?: number; disk?: number; temperature?: number }[] }) {
  const series = [{ label: "CPU", color: "#0891b2" }, { label: "Memory", color: "#5966be" }, { label: "Disk", color: "#c98a20" }, { label: "Temperature", color: "#cb5a52" }];
  return <div className="chartComposition"><div className="chartKey">{series.map((item) => <span key={item.label}><i style={{ background: item.color }} />{item.label}</span>)}</div><div className="chartPlot"><ResponsiveContainer width="100%" height="100%"><AreaChart data={data} margin={{ top: 10, right: 22, left: -10, bottom: 2 }}><CartesianGrid stroke="#e6e9f2" strokeDasharray="3 5" vertical={false} /><XAxis dataKey="time" minTickGap={38} tick={{ fill: "#747b91", fontSize: 11 }} axisLine={false} tickLine={false} /><YAxis domain={[0, 100]} width={42} tickFormatter={(value) => `${value}%`} tick={{ fill: "#747b91", fontSize: 11 }} axisLine={false} tickLine={false} /><Tooltip cursor={{ stroke: "#b1b5c8", strokeDasharray: "3 3" }} contentStyle={chartTooltipStyle} formatter={(value, seriesName) => [`${Number(value).toFixed(1)}${seriesName === "Temp °C" ? " °C" : "%"}`, seriesName]} /><Area type="monotone" dataKey="cpu" name="CPU" stroke="#0891b2" fill="#0891b2" fillOpacity={0.07} strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 2, fill: "#ffffff" }} /><Area type="monotone" dataKey="memory" name="Memory" stroke="#5966be" fill="#5966be" fillOpacity={0.045} strokeWidth={2.25} dot={false} activeDot={{ r: 4, strokeWidth: 2, fill: "#ffffff" }} /><Line type="monotone" dataKey="disk" name="Disk" stroke="#c98a20" strokeWidth={2} dot={false} activeDot={{ r: 4, strokeWidth: 2, fill: "#ffffff" }} /><Line type="monotone" dataKey="temperature" name="Temp °C" stroke="#cb5a52" strokeWidth={2} dot={false} activeDot={{ r: 4, strokeWidth: 2, fill: "#ffffff" }} /></AreaChart></ResponsiveContainer></div></div>;
}

function PresenceChart({ connected, total }: { connected: number; total: number }) {
  const rows = total ? [{ name: "Connected", value: connected, color: "#2a9d7f" }, { name: "Not connected", value: Math.max(total - connected, 0), color: "#d96861" }] : [{ name: "No devices", value: 1, color: "#dfe7e6" }];
  return <div className="chartComposition presenceComposition"><div className="chartPlot"><ResponsiveContainer width="100%" height="100%"><PieChart><Pie data={rows} dataKey="value" nameKey="name" innerRadius={66} outerRadius={88} startAngle={90} endAngle={-270} paddingAngle={total > 0 ? 2 : 0} cornerRadius={5}>{rows.map((row) => <Cell key={row.name} fill={row.color} stroke="none" />)}</Pie><Tooltip contentStyle={chartTooltipStyle} /><text x="50%" y="45%" textAnchor="middle" className="donutValue">{total ? `${Math.round(connected * 100 / total)}%` : "—"}</text><text x="50%" y="55%" textAnchor="middle" className="donutLabel">online</text></PieChart></ResponsiveContainer></div><div className="chartKey centered"><span><i style={{ background: "#2a9d7f" }} />Connected</span><span><i style={{ background: "#d96861" }} />Not connected</span></div></div>;
}

function ConnectorChart({ devices }: { devices: FleetData["downstream"] }) {
  const counts = Array.from(devices.reduce((map, device) => map.set(device.connector_type, (map.get(device.connector_type) || 0) + 1), new Map<string, number>())).map(([name, value]) => ({ name, value }));
  return <ResponsiveContainer width="100%" height="100%"><BarChart data={counts} layout="vertical" margin={{ left: 8, right: 40 }}><XAxis type="number" hide /><YAxis type="category" dataKey="name" width={95} axisLine={false} tickLine={false} tick={{ fill: "#536865", fontSize: 12 }} /><Tooltip cursor={{ fill: "#f3f7f6" }} contentStyle={chartTooltipStyle} /><Bar dataKey="value" name="Devices" fill="#3f7f9f" radius={[0, 5, 5, 0]} maxBarSize={18} background={{ fill: "#edf2f1", radius: 5 }}><LabelList dataKey="value" position="right" fill="#425653" fontSize={12} /></Bar></BarChart></ResponsiveContainer>;
}

function ResourceBars({ metrics }: { metrics: FleetData["metrics"] }) {
  const rows = [
    { name: "CPU", value: latestMatching(metrics, "cpu.utilization"), fill: "#0891b2" },
    { name: "Memory", value: latestMatching(metrics, "memory.utilization"), fill: "#6f78e8" },
    { name: "Disk", value: latestMatching(metrics, "filesystem.utilization"), fill: "#f2a51a" },
    { name: "Temperature", value: latestMatching(metrics, "temperature"), fill: "#ef6a6a" },
  ].filter((row) => row.value !== undefined);
  if (!rows.length) return <ChartEmpty text="Waiting for current resource values" />;
  return <ResponsiveContainer width="100%" height="100%"><BarChart data={rows} layout="vertical" margin={{ top: 8, left: 12, right: 54, bottom: 8 }}><XAxis type="number" domain={[0, 100]} hide /><YAxis type="category" dataKey="name" width={92} axisLine={false} tickLine={false} tick={{ fill: "#536865", fontSize: 12 }} /><Tooltip cursor={{ fill: "#f3f7f6" }} contentStyle={chartTooltipStyle} formatter={(value) => Number(value).toFixed(1)} /><Bar dataKey="value" radius={[0, 5, 5, 0]} maxBarSize={18} background={{ fill: "#edf2f1", radius: 5 }}><LabelList dataKey="value" position="right" formatter={(value) => Number(value).toFixed(1)} fill="#425653" fontSize={12} />{rows.map((row) => <Cell key={row.name} fill={row.fill} />)}</Bar></BarChart></ResponsiveContainer>;
}

function NetworkChart({ metrics }: { metrics: FleetData["metrics"] }) {
  const rows = [
    { name: "Received", value: latestMatching(metrics, "network.receive_bytes_total"), fill: "#2b8ee6" },
    { name: "Transmitted", value: latestMatching(metrics, "network.transmit_bytes_total"), fill: "#6d5ce7" },
  ].filter((row) => row.value !== undefined).map((row) => ({ ...row, value: Number(row.value) / 1024 / 1024 }));
  if (!rows.length) return <ChartEmpty text="Waiting for network counters" />;
  return <ResponsiveContainer width="100%" height="100%"><BarChart data={rows} margin={{ top: 28, right: 22, left: 2, bottom: 2 }}><CartesianGrid stroke="#e5eceb" strokeDasharray="3 5" vertical={false} /><XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fill: "#536865", fontSize: 12 }} /><YAxis unit=" MB" width={68} axisLine={false} tickLine={false} tick={{ fill: "#748583", fontSize: 11 }} /><Tooltip cursor={{ fill: "#f3f7f6" }} contentStyle={chartTooltipStyle} formatter={(value) => `${Number(value).toFixed(1)} MB`} /><Bar dataKey="value" radius={[6, 6, 0, 0]} maxBarSize={72}><LabelList dataKey="value" position="top" formatter={(value) => `${Number(value).toFixed(1)} MB`} fill="#425653" fontSize={11} />{rows.map((row) => <Cell key={row.name} fill={row.fill} />)}</Bar></BarChart></ResponsiveContainer>;
}

function CapacityChart({ memory, storage }: { memory: number; storage: number }) {
  const rows = [{ name: "Memory", value: memory / 1024 ** 3, fill: "#6f78e8" }, { name: "Root storage", value: storage / 1024 ** 3, fill: "#f2a51a" }].filter((row) => row.value > 0);
  if (!rows.length) return <ChartEmpty text="Waiting for inventory capacity" />;
  return <ResponsiveContainer width="100%" height="100%"><BarChart data={rows} margin={{ top: 28, right: 22, left: 2, bottom: 2 }}><CartesianGrid stroke="#e5eceb" strokeDasharray="3 5" vertical={false} /><XAxis dataKey="name" axisLine={false} tickLine={false} tick={{ fill: "#536865", fontSize: 12 }} /><YAxis unit=" GB" width={62} axisLine={false} tickLine={false} tick={{ fill: "#748583", fontSize: 11 }} /><Tooltip cursor={{ fill: "#f3f7f6" }} contentStyle={chartTooltipStyle} formatter={(value) => `${Number(value).toFixed(1)} GB`} /><Bar dataKey="value" radius={[6, 6, 0, 0]} maxBarSize={72}><LabelList dataKey="value" position="top" formatter={(value) => `${Number(value).toFixed(1)} GB`} fill="#425653" fontSize={11} />{rows.map((row) => <Cell key={row.name} fill={row.fill} />)}</Bar></BarChart></ResponsiveContainer>;
}

const chartTooltipStyle: React.CSSProperties = { border: "1px solid #d5e0de", borderRadius: 6, boxShadow: "0 10px 28px rgba(31, 54, 51, .12)", color: "#203431", fontSize: 12 };

const packageBrands: { test: RegExp; icon: SimpleIcon }[] = [
  { test: /^nginx/i, icon: siNginx }, { test: /^(apache2|httpd)/i, icon: siApache },
  { test: /^mariadb/i, icon: siMariadb }, { test: /^mysql/i, icon: siMysql }, { test: /^(postgres|libpq)/i, icon: siPostgresql },
  { test: /^mongo/i, icon: siMongodb }, { test: /^redis/i, icon: siRedis }, { test: /^rabbitmq/i, icon: siRabbitmq },
  { test: /^(docker|containerd)/i, icon: siDocker }, { test: /^python/i, icon: siPython }, { test: /^git($|-)/i, icon: siGit },
  { test: /^ubuntu/i, icon: siUbuntu }, { test: /^flatpak/i, icon: siFlatpak }, { test: /^(openssl|libssl)/i, icon: siOpenssl },
  { test: /^(curl|libcurl)/i, icon: siCurl }, { test: /^(nodejs|node-|npm$)/i, icon: siNodedotjs }, { test: /^anydesk/i, icon: siAnydesk },
  { test: /^(bash|dash$)/i, icon: siGnubash }, { test: /^(linux|kernel)/i, icon: siLinux }, { test: /^trivy/i, icon: siTrivy },
];

function PackageBrandIcon({ name }: { name: string }) {
  const brand = packageBrands.find((entry) => entry.test.test(name))?.icon;
  if (brand) return <span className="packageBrand" title={brand.title} style={{ color: `#${brand.hex}` }}><svg viewBox="0 0 24 24" role="img" aria-label={`${brand.title} logo`}><path fill="currentColor" d={brand.path} /></svg></span>;
  const initials = name.split(/[-_.]+/).filter(Boolean).slice(0, 2).map((part) => part[0]).join("").toUpperCase() || "PK";
  const tone = Array.from(name).reduce((sum, letter) => sum + letter.charCodeAt(0), 0) % 5;
  return <span className={`packageMonogram tone${tone}`} title="System package without an official project logo">{initials}</span>;
}

function pageExportData(page: PageId, data: FleetData) {
  const shared = { generated_at: new Date().toISOString(), page, organization: data.organization };
  switch (page) {
    case "devices": return { ...shared, devices: data.devices };
    case "gateways": return { ...shared, gateways: data.devices.filter((device) => device.capabilities?.includes("gateway")), downstream_devices: data.downstream };
    case "telemetry": case "health": return { ...shared, devices: data.devices, metrics: data.metrics, inventory: data.inventory };
    case "inventory": case "applications": return { ...shared, devices: data.devices, inventory: data.inventory, applications: data.applications };
    case "map": return { ...shared, locations: data.devices.map(({ id, display_name, presence, location }) => ({ id, display_name, presence, location })) };
    case "jobs": return { ...shared, jobs: data.jobs };
    case "deployments": return { ...shared, deployments: data.deployments };
    case "ota": return { ...shared, update_campaigns: data.otaCampaigns };
    case "audit": return { ...shared, audit_events: data.audit };
    case "users": return { ...shared, users: data.users, roles: data.roles };
    case "enrollment": return { ...shared, enrollment_tokens: data.enrollmentTokens.map(({ token: _token, ...record }) => record) };
    default: return { ...shared, devices: data.devices, jobs: data.jobs, alerts: deriveAlerts(data.devices) };
  }
}

function canAdminister(membership: Membership) { return ["platform_administrator", "organization_owner", "organization_administrator"].includes(membership.role); }
function normalizeArchitecture(value?: string) { const normalized = (value || "").toLowerCase(); if (["aarch64", "arm64"].includes(normalized)) return "arm64"; if (["arm", "armv7", "armv7l", "armhf"].includes(normalized)) return "armv7"; if (["amd64", "x86_64"].includes(normalized)) return "amd64"; return normalized; }
function formatCampaignState(value: string) { return value.replaceAll("canary", "pilot").replaceAll("_", " "); }

function telemetryChart(data: FleetData) { const grouped = new Map<string, { time: string; cpu?: number[]; memory?: number[]; disk?: number[]; temperature?: number[] }>(); data.metrics.forEach((metric) => { const time = new Date(metric.device_time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }); const point = grouped.get(time) || { time, cpu: [], memory: [], disk: [], temperature: [] }; if (metric.name.includes("cpu.utilization")) point.cpu?.push(metric.value); if (metric.name.includes("memory.utilization")) point.memory?.push(metric.value); if (metric.name.includes("filesystem.utilization")) point.disk?.push(metric.value); if (metric.name.includes("temperature")) point.temperature?.push(metric.value); grouped.set(time, point); }); const points = Array.from(grouped.values()).map((point) => ({ time: point.time, cpu: average(point.cpu), memory: average(point.memory), disk: average(point.disk), temperature: average(point.temperature) })); const step = Math.max(1, Math.ceil(points.length / 180)); return points.filter((_, index) => index % step === 0 || index === points.length - 1); }
function average(values?: number[]) { return values?.length ? Math.round(values.reduce((a, b) => a + b, 0) / values.length) : undefined; }
function latestMatching(metrics: FleetData["metrics"], name: string) { return metrics.filter((metric) => metric.name.includes(name)).at(-1)?.value; }
function formatPercent(value?: number) { return value === undefined ? "—" : `${value.toFixed(0)}%`; }
function formatTemperature(value?: number) { return value === undefined ? "—" : `${value.toFixed(1)} °C`; }
function nameFor(data: FleetData, id: string) { return data.devices.find((device) => device.id === id)?.display_name || shortId(id); }
function jobOutput(job: Job) { if (!job.result) return job.state === "queued" || job.state === "delivered" ? "Waiting for the device to return a result." : "No output was returned."; if (typeof job.result === "string") return job.result; const result = job.result as { stdout?: string; stderr?: string }; return result.stdout || result.stderr || JSON.stringify(job.result, null, 2); }
function vulnerabilityReport(job: Job): VulnerabilityReport | null { try { const parsed = JSON.parse(jobOutput(job)) as VulnerabilityReport; return parsed?.scanner === "trivy" && Array.isArray(parsed.packages) ? parsed : null; } catch { return null; } }
function shortId(value: string) { return value.length > 14 ? `${value.slice(0, 8)}…` : value; }
function formatBytes(bytes: number) { if (!bytes) return "0 B"; const units = ["B", "KB", "MB", "GB"]; const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1); return `${(bytes / 1024 ** index).toFixed(index > 1 ? 1 : 0)} ${units[index]}`; }
function formatDate(value?: string) { return value ? new Date(value).toLocaleString([], { dateStyle: "medium", timeStyle: "short" }) : "—"; }
function relativeTime(value?: string) { if (!value) return "Never"; const seconds = Math.floor((Date.now() - new Date(value).getTime()) / 1000); if (seconds < 60) return `${seconds}s ago`; if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`; if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`; return `${Math.floor(seconds / 86400)}d ago`; }
function recordText(record: Record<string, unknown>, key: string) { const value = record[key]; return value === undefined || value === null || value === "" ? "—" : String(value); }
function deriveAlerts(devices: Device[]) {
  const alerts: { id: string; severity: "critical" | "warning"; title: string; detail: string; device: string; page: PageId }[] = [];
  const now = Date.now();
  devices.forEach((device) => {
    const lastSeen = device.last_seen_at ? new Date(device.last_seen_at).getTime() : 0;
    if (device.presence === "offline" || (lastSeen > 0 && now - lastSeen > 5 * 60 * 1000)) alerts.push({ id: `${device.id}-offline`, severity: "critical", title: "Device is not reporting", detail: `Last seen ${relativeTime(device.last_seen_at)}`, device: device.display_name, page: "devices" });
    if (device.health_state && !["healthy", "unknown"].includes(device.health_state)) alerts.push({ id: `${device.id}-health`, severity: "warning", title: `Health state: ${device.health_state}`, detail: "Reported by the latest heartbeat", device: device.display_name, page: "telemetry" });
    if (device.certificate_expires_at) { const days = Math.ceil((new Date(device.certificate_expires_at).getTime() - now) / 86400000); if (days <= 30) alerts.push({ id: `${device.id}-certificate`, severity: days <= 7 ? "critical" : "warning", title: "Certificate renewal required", detail: `${Math.max(days, 0)} days remaining`, device: device.display_name, page: "enrollment" }); }
  });
  return alerts;
}
function badgeTone(value: string) { const normalized = value.toLowerCase(); if (["connected", "healthy", "operational", "completed", "verified", "active", "success", "ready", "up to date", "current", "enabled"].some((term) => normalized.includes(term))) return "positive"; if (["critical", "offline", "failed", "unsigned", "attention", "expired"].some((term) => normalized.includes(term))) return "negative"; if (["warning", "awaiting", "queued", "review", "monitoring", "available", "paused", "not enabled", "outdated"].some((term) => normalized.includes(term))) return "warning"; return "neutral"; }
