import type { Metadata } from "next";
import { FlowUnavailable, KratosFlow } from "@/components/kratos-flow";
import { getFlow, startFlowUrl, type SearchParams } from "@/lib/kratos";

export const metadata: Metadata = { title: "Masuk · MesinProspek" };

export default async function LoginPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const params = await searchParams;
  const flow = await getFlow("login", params);
  return (
    <>
      <h1 className="msp-text-h2 text-fg">Masuk</h1>
      {flow ? <KratosFlow ui={flow.ui} /> : <FlowUnavailable restartHref={startFlowUrl("login", params)} />}
      <div className="space-y-1 msp-text-body-sm text-fg-muted">
        <p>
          <a href={startFlowUrl("recovery")} className="text-fg-link underline">
            Lupa kata sandi?
          </a>
        </p>
        <p>
          Belum punya akun?{" "}
          <a href={startFlowUrl("registration")} className="text-fg-link underline">
            Daftar
          </a>
        </p>
      </div>
    </>
  );
}
