// Kratos browser flows, read on the server (docs/04 §9). Pages render the flow's form and the
// browser posts it straight to Kratos at /.ory/* on the same origin, so the app holds no
// passwords and needs no client JavaScript. This is the only place that talks to Kratos.
import { headers } from "next/headers";
import { redirect } from "next/navigation";

/** Path where Caddy serves the Kratos public API to the browser (docs/06 §5). */
export const KRATOS_BROWSER_BASE = "/.ory";

export type FlowKind = "login" | "registration" | "verification" | "recovery" | "settings";

export type SearchParams = Record<string, string | string[] | undefined>;

/** Message of Kratos: `id` is stable, `text` is English (see lib/kratos-text.ts). */
export interface UiText {
  id: number;
  text: string;
  type: "info" | "error" | "success";
  context?: Record<string, unknown>;
}

export interface UiNodeAttributes {
  node_type: "input" | "img" | "a" | "text" | "script";
  name?: string;
  type?: string;
  value?: unknown;
  required?: boolean;
  disabled?: boolean;
  autocomplete?: string;
  pattern?: string;
  src?: string;
  width?: number;
  height?: number;
  href?: string;
  title?: UiText;
  id?: string;
  text?: UiText;
}

export interface UiNode {
  type: UiNodeAttributes["node_type"];
  group: string;
  attributes: UiNodeAttributes;
  messages: UiText[];
  meta: { label?: UiText };
}

export interface UiContainer {
  action: string;
  method: string;
  nodes: UiNode[];
  messages?: UiText[];
}

export interface Flow {
  id: string;
  ui: UiContainer;
  state?: string;
}

/** Query parameters a flow start passes on to Kratos. */
const START_PARAMS: Record<FlowKind, string[]> = {
  login: ["return_to", "aal", "refresh"],
  registration: ["return_to"],
  verification: ["return_to"],
  recovery: ["return_to"],
  settings: ["return_to"],
};

/** Browser URL that starts a new flow of kind (Kratos sets its CSRF cookie and comes back with ?flow=). */
export function startFlowUrl(kind: FlowKind, params: SearchParams = {}): string {
  const query = new URLSearchParams();
  for (const name of START_PARAMS[kind]) {
    const value = first(params[name]);
    if (value) query.set(name, value);
  }
  const qs = query.toString();
  return `${KRATOS_BROWSER_BASE}/self-service/${kind}/browser${qs ? `?${qs}` : ""}`;
}

/**
 * Returns the flow named by ?flow=, or starts a new one when there is none or it expired.
 * Returns null when Kratos is unreachable or rejects the flow; the page then offers a restart.
 */
export async function getFlow(kind: FlowKind, params: SearchParams): Promise<Flow | null> {
  const id = first(params.flow);
  if (!id) redirect(startFlowUrl(kind, params));

  const res = await kratos(`/self-service/${kind}/flows?id=${encodeURIComponent(id)}`);
  if (res === null) return null;
  if (res.ok) return (await res.json()) as Flow;
  if (res.status === 404 || res.status === 410) redirect(startFlowUrl(kind, params));
  const body = (await res.json().catch(() => ({}))) as { redirect_browser_to?: string };
  if (body.redirect_browser_to) redirect(body.redirect_browser_to);
  if (res.status === 401) redirect(startFlowUrl("login"));
  return null;
}

/** Logout URL for the current session, or null without a session. */
export async function getLogoutUrl(): Promise<string | null> {
  const res = await kratos("/self-service/logout/browser");
  if (!res?.ok) return null;
  const body = (await res.json()) as { logout_url?: string };
  return body.logout_url ?? null;
}

/** GET on the Kratos public API with the browser's cookies; null when Kratos is unreachable. */
async function kratos(path: string): Promise<Response | null> {
  const base = process.env.MSP_KRATOS_PUBLIC_URL;
  if (!base) throw new Error("MSP_KRATOS_PUBLIC_URL is not set");
  const cookie = (await headers()).get("cookie") ?? "";
  try {
    return await fetch(`${base.replace(/\/$/, "")}${path}`, {
      headers: { accept: "application/json", cookie },
      cache: "no-store",
      redirect: "manual",
    });
  } catch {
    return null;
  }
}

export function first(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}
