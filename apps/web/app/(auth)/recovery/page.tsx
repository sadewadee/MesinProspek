import type { Metadata } from "next";
import { FlowUnavailable, KratosFlow } from "@/components/kratos-flow";
import { getFlow, startFlowUrl, type SearchParams } from "@/lib/kratos";

export const metadata: Metadata = { title: "Pulihkan akun · MesinProspek" };

export default async function RecoveryPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const params = await searchParams;
  const flow = await getFlow("recovery", params);
  return (
    <>
      <div className="space-y-1">
        <h1 className="msp-text-h2 text-fg">Pulihkan akun</h1>
        <p className="msp-text-body text-fg-muted">
          Masukkan email akun Anda. Kami kirim kode untuk masuk dan mengganti kata sandi.
        </p>
      </div>
      {flow ? <KratosFlow ui={flow.ui} /> : <FlowUnavailable restartHref={startFlowUrl("recovery", params)} />}
      <p className="msp-text-body-sm text-fg-muted">
        <a href={startFlowUrl("login")} className="text-fg-link underline">
          Kembali ke halaman masuk
        </a>
      </p>
    </>
  );
}
