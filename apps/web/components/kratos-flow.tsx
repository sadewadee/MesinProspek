// Renders the UI of a Kratos flow as plain HTML forms posted to Kratos (docs/04 §9).
// Each method group (Google, password, code, TOTP, …) is its own form; the "default" nodes
// (CSRF token, email) join each of them. Kratos validates and returns field messages.
import Image from "next/image";
import type { UiContainer, UiNode, UiText } from "@/lib/kratos";
import { textOf } from "@/lib/kratos-text";

const GROUP_ORDER = ["oidc", "profile", "password", "code", "totp", "lookup_secret"];

/** Section titles on the account page. */
const GROUP_TITLES: Record<string, string> = {
  profile: "Email",
  password: "Ganti kata sandi",
  totp: "Aplikasi autentikator (MFA)",
  oidc: "Akun Google",
};

const buttonBase =
  "inline-flex h-9 w-full items-center justify-center rounded-md px-4 msp-text-label focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-border-focus disabled:opacity-50";
const primaryButton = `${buttonBase} bg-accent text-fg-on-accent hover:bg-accent-hover`;
const secondaryButton = `${buttonBase} border border-border-strong bg-surface text-fg hover:bg-hover`;

export function KratosFlow({ ui, titled = false }: { ui: UiContainer; titled?: boolean }) {
  const defaults = ui.nodes.filter((n) => n.group === "default");
  const groups = [...new Set(ui.nodes.map((n) => n.group))]
    .filter((g) => g !== "default")
    .sort((a, b) => rank(a) - rank(b));
  if (groups.length === 0) groups.push("default");

  return (
    <div className="space-y-4">
      <Messages messages={ui.messages} />
      {groups.map((group, i) => (
        <div key={group} className="space-y-4">
          {i > 0 && groups[i - 1] === "oidc" && !titled && (
            <p className="text-center msp-text-caption text-fg-subtle">atau</p>
          )}
          <form action={ui.action} method={ui.method} className="space-y-3">
            {titled && GROUP_TITLES[group] && <h2 className="msp-text-h3 text-fg">{GROUP_TITLES[group]}</h2>}
            {/* Google needs no email field: only the hidden CSRF token joins its form. */}
            {(group === "oidc" ? defaults.filter(isHidden) : group === "default" ? [] : defaults).map((node) => (
              <Node key={key(node)} node={node} />
            ))}
            {ui.nodes
              .filter((n) => n.group === group)
              .map((node) => (
                <Node key={key(node)} node={node} />
              ))}
          </form>
        </div>
      ))}
    </div>
  );
}

/** Shown when Kratos is unreachable or rejects the flow (docs/03 §7 error pattern). */
export function FlowUnavailable({ restartHref }: { restartHref: string }) {
  return (
    <div role="alert" className="space-y-3 rounded-md border border-status-danger-border bg-status-danger-bg px-3 py-2">
      <p className="msp-text-body text-status-danger-fg">
        Formulir tidak dapat dimuat. Formulir ini sudah tidak berlaku atau layanan masuk sedang terganggu. Mulai
        lagi dalam beberapa saat.
      </p>
      <a href={restartHref} className="inline-block text-fg-link underline">
        Mulai lagi
      </a>
    </div>
  );
}

function Node({ node }: { node: UiNode }) {
  const a = node.attributes;
  const label = node.meta.label ? textOf(node.meta.label) : undefined;
  switch (a.node_type) {
    case "input":
      if (a.type === "hidden") return <input type="hidden" name={a.name} value={String(a.value ?? "")} />;
      if (a.type === "submit" || a.type === "button") {
        // Only the primary submit (name "method") needs the form's fields; the others (resend code,
        // back, unlink TOTP, link Google) skip browser validation. Kratos validates on the server.
        return (
          <button
            type="submit"
            name={a.name}
            value={String(a.value ?? "")}
            disabled={a.disabled}
            formNoValidate={a.name !== "method"}
            className={a.name === "method" ? primaryButton : secondaryButton}
          >
            {label}
          </button>
        );
      }
      return <Field node={node} label={label} />;
    case "img":
      return a.src ? (
        <Image src={a.src} alt={label ?? ""} width={a.width ?? 256} height={a.height ?? 256} unoptimized />
      ) : null;
    case "text":
      return (
        <div className="space-y-1">
          {label && <p className="msp-text-body-sm text-fg-muted">{label}</p>}
          {a.text && <code className="block break-all msp-text-mono text-fg">{textOf(a.text)}</code>}
        </div>
      );
    case "a":
      return (
        <a href={a.href} className="inline-block text-fg-link underline">
          {a.title ? textOf(a.title) : label}
        </a>
      );
    default:
      return null; // scripts (WebAuthn) are not enabled
  }
}

function Field({ node, label }: { node: UiNode; label?: string }) {
  const a = node.attributes;
  const errorId = node.messages.length > 0 ? `${a.name}-messages` : undefined;
  const checkbox = a.type === "checkbox";
  return (
    <label className={checkbox ? "flex items-center gap-2" : "block space-y-1"}>
      {!checkbox && <span className="block msp-text-label text-fg">{label}</span>}
      <input
        name={a.name}
        type={a.type}
        defaultValue={checkbox ? undefined : typeof a.value === "string" ? a.value : undefined}
        defaultChecked={checkbox ? a.value === true : undefined}
        required={a.required}
        disabled={a.disabled}
        autoComplete={a.autocomplete}
        pattern={a.pattern}
        aria-invalid={errorId ? true : undefined}
        aria-describedby={errorId}
        className={
          checkbox
            ? "size-4"
            : "h-9 w-full rounded-md border border-border-input bg-surface px-3 text-fg focus-visible:outline-2 focus-visible:outline-border-focus"
        }
      />
      {checkbox && <span className="msp-text-body text-fg">{label}</span>}
      {errorId && (
        <span id={errorId} className="block msp-text-body-sm text-status-danger-fg">
          {node.messages.map(textOf).join(" ")}
        </span>
      )}
    </label>
  );
}

const MESSAGE_STYLES: Record<UiText["type"], string> = {
  error: "border-status-danger-border bg-status-danger-bg text-status-danger-fg",
  info: "border-status-info-border bg-status-info-bg text-status-info-fg",
  success: "border-status-success-border bg-status-success-bg text-status-success-fg",
};

function Messages({ messages }: { messages?: UiText[] }) {
  if (!messages?.length) return null;
  return (
    <div className="space-y-2">
      {messages.map((m) => (
        <p
          key={m.id}
          role={m.type === "error" ? "alert" : "status"}
          className={`rounded-md border px-3 py-2 msp-text-body ${MESSAGE_STYLES[m.type] ?? MESSAGE_STYLES.info}`}
        >
          {textOf(m)}
        </p>
      ))}
    </div>
  );
}

function rank(group: string): number {
  const i = GROUP_ORDER.indexOf(group);
  return i === -1 ? GROUP_ORDER.length : i;
}

function isHidden(node: UiNode): boolean {
  return node.attributes.type === "hidden";
}

function key(node: UiNode): string {
  const a = node.attributes;
  return `${node.group}:${a.node_type}:${a.name ?? a.id ?? ""}:${String(a.value ?? "")}`;
}
