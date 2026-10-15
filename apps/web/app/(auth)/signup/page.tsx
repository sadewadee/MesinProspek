import type { Metadata } from "next";
import { FlowUnavailable, KratosFlow } from "@/components/kratos-flow";
import { getFlow, startFlowUrl, type SearchParams } from "@/lib/kratos";

export const metadata: Metadata = { title: "Daftar · MesinProspek" };

export default async function SignupPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const params = await searchParams;
  const flow = await getFlow("registration", params);
  return (
    <>
      <div className="space-y-1">
        <h1 className="msp-text-h2 text-fg">Daftar</h1>
        <p className="msp-text-body text-fg-muted">Setelah mendaftar, verifikasi email Anda untuk mulai memakai data.</p>
      </div>
      {flow ? <KratosFlow ui={flow.ui} /> : <FlowUnavailable restartHref={startFlowUrl("registration", params)} />}
      <p className="msp-text-body-sm text-fg-muted">
        Sudah punya akun?{" "}
        <a href={startFlowUrl("login")} className="text-fg-link underline">
          Masuk
        </a>
      </p>
    </>
  );
}
