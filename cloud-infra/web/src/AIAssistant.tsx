import React from "react";
import {
  Activity,
  AlertTriangle,
  Bot,
  Check,
  ChevronRight,
  Cpu,
  KeyRound,
  LoaderCircle,
  MessageSquareText,
  RadioTower,
  Send,
  Settings2,
  ShieldCheck,
  Sparkles,
  Trash2,
  X,
  Zap,
} from "lucide-react";
import {
  AIAction,
  AIChatMessage,
  AIConfiguration,
  AIProvider,
  askAssistant,
  deleteAIConfiguration,
  executeAIAction,
  FleetData,
  getAIConfiguration,
  saveAIConfiguration,
  testAIConnection,
} from "./api";

export type AssistantNavigationPage =
  | "overview" | "devices" | "gateways" | "telemetry" | "inventory" | "map"
  | "jobs" | "remote" | "deployments" | "ota" | "alerts" | "profiles"
  | "applications" | "artifacts" | "enrollment" | "audit" | "users" | "health" | "onprem";

type ChatEntry = {
  id: string;
  role: "user" | "assistant";
  content: string;
  actions?: AIAction[];
  suggestions?: string[];
  model?: string;
  tokens?: number;
};

const providerDefaults: Record<AIProvider, { baseURL: string; model: string; label: string; detail: string }> = {
  openai: { baseURL: "https://api.openai.com/v1", model: "gpt-5-mini", label: "OpenAI", detail: "Fast hosted reasoning" },
  anthropic: { baseURL: "https://api.anthropic.com", model: "claude-haiku-5-5", label: "Anthropic", detail: "Low-latency Claude" },
  compatible: { baseURL: "http://host.docker.internal:11434/v1", model: "qwen3:4b", label: "Local / compatible", detail: "Ollama, LocalAI, or private gateway" },
};

const quickPrompts = [
  "Which devices need attention right now?",
  "Explain why any devices are offline.",
  "Review recent failed or stuck jobs.",
  "What should I check before the next update campaign?",
];

function restoredConversation(key: string): ChatEntry[] {
  try {
    const value = JSON.parse(sessionStorage.getItem(key) || "[]") as ChatEntry[];
    return Array.isArray(value) ? value.filter((entry) => entry?.role && typeof entry.content === "string").slice(-20) : [];
  } catch {
    return [];
  }
}

export default function AIAssistant({ data, onNavigate, onRefresh }: {
  data: FleetData;
  onNavigate: (page: AssistantNavigationPage) => void;
  onRefresh: () => void;
}) {
  const storageKey = `soul_room_ai_conversation_${data.membership.tenant_id}`;
  const [configuration, setConfiguration] = React.useState<AIConfiguration | null>(null);
  const [configError, setConfigError] = React.useState("");
  const [manageOpen, setManageOpen] = React.useState(false);
  const [provider, setProvider] = React.useState<AIProvider>("openai");
  const [baseURL, setBaseURL] = React.useState(providerDefaults.openai.baseURL);
  const [model, setModel] = React.useState(providerDefaults.openai.model);
  const [apiKey, setAPIKey] = React.useState("");
  const [enabled, setEnabled] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [testing, setTesting] = React.useState(false);
  const [connectionMessage, setConnectionMessage] = React.useState("");
  const [entries, setEntries] = React.useState<ChatEntry[]>(() => restoredConversation(storageKey));
  const [message, setMessage] = React.useState("");
  const [sending, setSending] = React.useState(false);
  const [chatError, setChatError] = React.useState("");
  const [approval, setApproval] = React.useState<AIAction | null>(null);
  const [runningAction, setRunningAction] = React.useState("");
  const [actionResults, setActionResults] = React.useState<Record<string, string>>({});
  const conversationEnd = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    getAIConfiguration(data.membership).then((value) => {
      setConfiguration(value);
      if (value.provider) setProvider(value.provider);
      if (value.base_url) setBaseURL(value.base_url);
      if (value.model) setModel(value.model);
      setEnabled(value.provider ? value.enabled : true);
      setManageOpen(!value.configured && value.can_manage);
    }).catch((reason) => setConfigError(reason instanceof Error ? reason.message : "AI configuration could not be loaded."));
  }, [data.membership]);

  React.useEffect(() => {
    sessionStorage.setItem(storageKey, JSON.stringify(entries.slice(-20).map(({ actions: _actions, ...entry }) => entry)));
    conversationEnd.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, [entries, storageKey]);

  function changeProvider(next: AIProvider) {
    setProvider(next);
    setBaseURL(providerDefaults[next].baseURL);
    setModel(providerDefaults[next].model);
    setConnectionMessage("");
  }

  async function saveConnection(event: React.FormEvent) {
    event.preventDefault();
    setSaving(true);
    setConfigError("");
    setConnectionMessage("");
    try {
      const value = await saveAIConfiguration(data.membership, {
        provider,
        base_url: baseURL.trim(),
        model: model.trim(),
        api_key: apiKey.trim() || undefined,
        enabled,
      });
      setConfiguration(value);
      setAPIKey("");
      setConnectionMessage("Connection settings saved. Test once before using the assistant.");
    } catch (reason) {
      setConfigError(reason instanceof Error ? reason.message : "Connection settings could not be saved.");
    } finally {
      setSaving(false);
    }
  }

  async function testConnection() {
    setTesting(true);
    setConfigError("");
    setConnectionMessage("");
    try {
      const result = await testAIConnection(data.membership);
      setConnectionMessage(`${result.provider} connection verified with ${result.model}.`);
    } catch (reason) {
      setConfigError(reason instanceof Error ? reason.message : "Connection test failed.");
    } finally {
      setTesting(false);
    }
  }

  async function removeConnection() {
    setSaving(true);
    setConfigError("");
    try {
      const value = await deleteAIConfiguration(data.membership);
      setConfiguration(value);
      setManageOpen(true);
      setConnectionMessage("Provider credentials removed.");
      setAPIKey("");
    } catch (reason) {
      setConfigError(reason instanceof Error ? reason.message : "Connection could not be removed.");
    } finally {
      setSaving(false);
    }
  }

  async function submit(prompt = message) {
    const clean = prompt.trim();
    if (!clean || sending || !configuration?.configured) return;
    const history: AIChatMessage[] = entries.slice(-6).map(({ role, content }) => ({ role, content }));
    const userEntry: ChatEntry = { id: crypto.randomUUID(), role: "user", content: clean };
    setEntries((current) => [...current, userEntry]);
    setMessage("");
    setSending(true);
    setChatError("");
    try {
      const response = await askAssistant(data.membership, clean, history);
      setEntries((current) => [...current, {
        id: crypto.randomUUID(),
        role: "assistant",
        content: response.answer,
        actions: response.actions || [],
        suggestions: response.follow_up_prompts || [],
        model: response.model,
        tokens: (response.usage?.input_tokens || 0) + (response.usage?.output_tokens || 0),
      }]);
    } catch (reason) {
      setChatError(reason instanceof Error ? reason.message : "The assistant could not complete that request.");
    } finally {
      setSending(false);
    }
  }

  async function runAction(action: AIAction) {
    if (!action.approval_token) return;
    setRunningAction(action.id);
    setChatError("");
    try {
      const result = await executeAIAction(data.membership, action.approval_token);
      setActionResults((current) => ({ ...current, [action.id]: `Queued as ${result.job.id.slice(-8)}` }));
      setApproval(null);
      onRefresh();
    } catch (reason) {
      setChatError(reason instanceof Error ? reason.message : "The approved action could not be queued.");
      setApproval(null);
    } finally {
      setRunningAction("");
    }
  }

  const connected = data.devices.filter((device) => device.presence === "connected").length;
  const unhealthy = data.devices.filter((device) => device.health_state && device.health_state !== "healthy").length;
  const isConfigured = Boolean(configuration?.configured);

  return <div className="assistantPage">
    <section className={isConfigured ? "assistantStatus ready" : "assistantStatus"}>
      <div className="assistantStatusIcon"><Sparkles size={21} /></div>
      <div><span>{isConfigured ? "OPERATIONS COPILOT READY" : "CONNECT A MODEL PROVIDER"}</span><strong>{isConfigured ? `${configuration?.model} can analyze this tenant's minimized fleet snapshot` : "Bring your own hosted or local model"}</strong><small>{isConfigured ? "No background requests. Actions always require your approval." : "Credentials are encrypted in the control-plane volume and never sent to the browser again."}</small></div>
      {configuration?.can_manage && <button className="button secondary" onClick={() => setManageOpen(!manageOpen)}><Settings2 size={16} />{manageOpen ? "Close setup" : "AI connection"}</button>}
    </section>

    {configError && <div className="inlineError assistantError"><AlertTriangle size={17} />{configError}<button onClick={() => setConfigError("")}><X size={15} /></button></div>}

    {manageOpen && configuration?.can_manage && <form className="assistantConfiguration" onSubmit={saveConnection}>
      <header><div><h2>Model connection</h2><p>Choose a provider for this organization. Leaving the key blank preserves the saved credential.</p></div><ShieldCheck size={20} /></header>
      <div className="providerOptions" role="radiogroup" aria-label="AI provider">
        {(Object.keys(providerDefaults) as AIProvider[]).map((item) => <button type="button" role="radio" aria-checked={provider === item} className={provider === item ? "providerOption active" : "providerOption"} key={item} onClick={() => changeProvider(item)}><span>{item === "compatible" ? <Cpu size={17} /> : item === "anthropic" ? <MessageSquareText size={17} /> : <Sparkles size={17} />}</span><strong>{providerDefaults[item].label}</strong><small>{providerDefaults[item].detail}</small>{provider === item && <Check size={15} />}</button>)}
      </div>
      <div className="assistantConfigFields">
        <label>Model<input value={model} onChange={(event) => setModel(event.target.value)} required maxLength={128} /></label>
        <label>Base URL<input value={baseURL} onChange={(event) => setBaseURL(event.target.value)} disabled={provider !== "compatible"} required /></label>
        <label>API key<div className="secretInput"><KeyRound size={16} /><input type="password" value={apiKey} onChange={(event) => setAPIKey(event.target.value)} placeholder={configuration?.provider === provider && configuration.key_hint ? `Saved key ${configuration.key_hint}` : provider === "compatible" ? "Optional for local providers" : "Enter provider API key"} autoComplete="off" /></div></label>
        <label className="assistantEnable"><input type="checkbox" checked={enabled} onChange={(event) => setEnabled(event.target.checked)} /><span><strong>Enable assistant</strong><small>Users with Assistant access can submit analysis requests.</small></span></label>
      </div>
      <footer>
        <div>{connectionMessage && <span className="connectionSuccess"><Check size={14} />{connectionMessage}</span>}</div>
        {configuration?.configured && <button type="button" className="textButton dangerText" onClick={removeConnection} disabled={saving}><Trash2 size={15} /> Remove</button>}
        {configuration?.configured && <button type="button" className="button secondary" onClick={testConnection} disabled={testing || saving}>{testing ? <LoaderCircle className="spin" size={16} /> : <Zap size={16} />} Test</button>}
        <button className="button primary" disabled={saving}>{saving ? <LoaderCircle className="spin" size={16} /> : <Check size={16} />} Save connection</button>
      </footer>
    </form>}

    {!isConfigured ? <section className="assistantEmpty">
      <div><Bot size={30} /></div><h2>Connect the assistant to begin</h2><p>{configuration?.can_manage ? "Choose a hosted provider or an OpenAI-compatible local endpoint. Soul Room sends only a compact tenant-scoped fleet snapshot when you ask a question." : "An organization owner needs to configure the AI connection before this workspace can be used."}</p>
      {configuration?.can_manage && !manageOpen && <button className="button primary" onClick={() => setManageOpen(true)}><Settings2 size={16} /> Configure connection</button>}
    </section> : <div className="assistantWorkspace">
      <section className="assistantConversation" aria-label="AI conversation">
        <header><div><Bot size={18} /><span><strong>Fleet conversation</strong><small>Session-only history</small></span></div>{entries.length > 0 && <button className="iconButton small" title="Clear conversation" aria-label="Clear conversation" onClick={() => setEntries([])}><Trash2 size={15} /></button>}</header>
        <div className="assistantMessages">
          {entries.length === 0 && <div className="assistantWelcome"><Sparkles size={24} /><h2>What would you like to understand?</h2><p>Ask about health, connectivity, jobs, software inventory, or update readiness.</p><div className="assistantPromptGrid">{quickPrompts.map((prompt) => <button key={prompt} onClick={() => submit(prompt)}><span>{prompt}</span><ChevronRight size={15} /></button>)}</div></div>}
          {entries.map((entry) => <article className={`assistantMessage ${entry.role}`} key={entry.id}>
            <div className="messageIdentity">{entry.role === "assistant" ? <Bot size={16} /> : <span>{(data.users.find((user) => user.id === data.membership.user_id)?.email || "You").slice(0, 2).toUpperCase()}</span>}</div>
            <div className="messageBody"><strong>{entry.role === "assistant" ? "Soul Room Assistant" : "You"}</strong><p>{entry.content}</p>
              {entry.actions && entry.actions.length > 0 && <div className="assistantActions">{entry.actions.map((action) => <div className="assistantAction" key={action.id}><span className={`actionRisk ${action.risk}`}>{action.risk}</span><div><strong>{action.title}</strong><small>{action.rationale}</small>{action.device_name && <em>{action.device_name} · {action.job_type?.replaceAll("_", " ")}</em>}</div>{actionResults[action.id] ? <span className="actionQueued"><Check size={14} />{actionResults[action.id]}</span> : action.kind === "navigate" && action.page ? <button className="button secondary" onClick={() => onNavigate(action.page as AssistantNavigationPage)}>Open <ChevronRight size={14} /></button> : <button className={action.risk === "high" ? "button warningButton" : "button primary"} onClick={() => setApproval(action)}>Review <ChevronRight size={14} /></button>}</div>)}</div>}
              {entry.model && <small className="assistantMeta">{entry.model}{entry.tokens ? ` · ${entry.tokens.toLocaleString()} tokens` : ""}</small>}
              {entry.suggestions && entry.suggestions.length > 0 && <div className="assistantSuggestions">{entry.suggestions.map((suggestion) => <button key={suggestion} onClick={() => submit(suggestion)}>{suggestion}</button>)}</div>}
            </div>
          </article>)}
          {sending && <article className="assistantMessage assistant"><div className="messageIdentity"><Bot size={16} /></div><div className="messageBody"><strong>Soul Room Assistant</strong><div className="assistantThinking"><i /><i /><i /><span>Analyzing the latest fleet snapshot</span></div></div></article>}
          <div ref={conversationEnd} />
        </div>
        {chatError && <div className="assistantChatError"><AlertTriangle size={15} />{chatError}<button onClick={() => setChatError("")}><X size={14} /></button></div>}
        <form className="assistantComposer" onSubmit={(event) => { event.preventDefault(); submit(); }}>
          <textarea value={message} onChange={(event) => setMessage(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter" && !event.shiftKey) { event.preventDefault(); submit(); } }} maxLength={2000} rows={2} placeholder="Ask about your fleet..." aria-label="Message the fleet assistant" />
          <div><span>{message.length > 1700 ? `${2000 - message.length} characters left` : "Enter to send · Shift+Enter for a new line"}</span><button className="button primary" disabled={!message.trim() || sending} aria-label="Send message">{sending ? <LoaderCircle className="spin" size={17} /> : <Send size={17} />} Send</button></div>
        </form>
      </section>

      <aside className="assistantContext">
        <header><span>LIVE CONTEXT</span><strong>What the assistant can see</strong></header>
        <div className="assistantContextMetric"><span className="green"><RadioTower size={17} /></span><div><strong>{connected} / {data.devices.length}</strong><small>devices connected</small></div></div>
        <div className="assistantContextMetric"><span className="amber"><Activity size={17} /></span><div><strong>{unhealthy}</strong><small>devices reporting health concerns</small></div></div>
        <div className="assistantContextMetric"><span className="blue"><Cpu size={17} /></span><div><strong>{data.metrics.length ? "Current" : "No data"}</strong><small>latest compact telemetry</small></div></div>
        <div className="assistantPrivacy"><ShieldCheck size={18} /><div><strong>Privacy boundary</strong><p>No credentials, certificates, precise location, remote-access identity, raw logs, or user records are included.</p></div></div>
        <div className="assistantEfficiency"><Zap size={17} /><div><strong>Token-efficient by design</strong><p>Relevant devices and recent operational state only. No background model calls.</p></div></div>
      </aside>
    </div>}

    {approval && <div className="modalBackdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setApproval(null); }}><section className="modal assistantApproval" role="dialog" aria-modal="true" aria-labelledby="ai-approval-title"><header><div><span>OPERATOR APPROVAL</span><h2 id="ai-approval-title">{approval.title}</h2></div><button className="iconButton" onClick={() => setApproval(null)} aria-label="Close approval"><X size={17} /></button></header><div className="approvalDetails"><div><span>Device</span><strong>{approval.device_name}</strong></div><div><span>Command</span><strong>{approval.job_type?.replaceAll("_", " ")}</strong></div><div><span>Risk</span><strong className={`actionRisk ${approval.risk}`}>{approval.risk}</strong></div></div><p>{approval.rationale}</p><div className="approvalNotice"><ShieldCheck size={17} /><span>The model cannot run this itself. Your approval queues a typed, audited device job; no shell command is generated.</span></div><footer><button className="button secondary" onClick={() => setApproval(null)}>Cancel</button><button className={approval.risk === "high" ? "button warningButton" : "button primary"} onClick={() => runAction(approval)} disabled={runningAction === approval.id}>{runningAction === approval.id ? <LoaderCircle className="spin" size={16} /> : <Check size={16} />} Approve and queue</button></footer></section></div>}
  </div>;
}
