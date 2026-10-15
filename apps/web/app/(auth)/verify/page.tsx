import type { Metadata } from "next";
import { FlowUnavailable, KratosFlow } from "@/components/kratos-flow";
import { getFlow, startFlowUrl, type SearchParams } from "@/lib/kratos";

export const metadata: Metadata = { title: "Verifikasi email · MesinProspek" };

export default async function VerifyPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const params = await searchParams;
  const flow = await getFlow("verification", params);
  return (
    <>
      <div className="space-y-1">
        <h1 className="msp-text-h2 text-fg">Verifikasi email</h1>
        <p className="msp-text-body text-fg-muted">
          Data hanya bisa dibuka setelah email Anda terverifikasi. Masukkan kode yang kami kirim ke email Anda.
        </p>
      </div>
      {flow ? <KratosFlow ui={flow.ui} /> : <FlowUnavailable restartHref={startFlowUrl("verification", params)} />}
    </>
  );
}
