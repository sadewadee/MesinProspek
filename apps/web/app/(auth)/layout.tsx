import type { ReactNode } from "react";

/** Auth screens (docs/03 SCR-01): one card, at most 100 spacing units wide, centred on the canvas. */
export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <main className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-100 space-y-6 rounded-lg border border-border bg-surface p-6 shadow-sm">
        <p className="msp-text-label text-accent-fg">MesinProspek</p>
        {children}
      </div>
    </main>
  );
}
