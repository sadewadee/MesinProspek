import type { Metadata } from "next";
import { first, startFlowUrl, type SearchParams } from "@/lib/kratos";

export const metadata: Metadata = { title: "Terjadi kendala · MesinProspek" };

/** Kratos error page (selfservice.flows.error.ui_url); ?id= names the error, shown as a reference code. */
export default async function AuthErrorPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const id = first((await searchParams).id);
  const code = id?.replace(/-/g, "").slice(0, 8).toUpperCase();
  return (
    <>
      <h1 className="msp-text-h2 text-fg">Terjadi kendala</h1>
      <div role="alert" className="space-y-1 rounded-md border border-status-danger-border bg-status-danger-bg px-3 py-2">
        <p className="msp-text-body text-status-danger-fg">
          Proses masuk tidak dapat dilanjutkan. Mulai lagi dari halaman masuk.
        </p>
        {code && <p className="msp-text-caption text-status-danger-fg">Kode {code}</p>}
      </div>
      <p className="msp-text-body-sm">
        <a href={startFlowUrl("login")} className="text-fg-link underline">
          Ke halaman masuk
        </a>
      </p>
    </>
  );
}
