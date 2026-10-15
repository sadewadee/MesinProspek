import type { Metadata } from "next";
import { FlowUnavailable, KratosFlow } from "@/components/kratos-flow";
import { getFlow, getLogoutUrl, startFlowUrl, type SearchParams } from "@/lib/kratos";

export const metadata: Metadata = { title: "Akun · MesinProspek" };

/** Account settings (docs/03 SCR-01): password, Google link, TOTP. Without a session Kratos sends the user to login. */
export default async function AccountPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const params = await searchParams;
  const flow = await getFlow("settings", params);
  const logoutUrl = await getLogoutUrl();
  return (
    <>
      <h1 className="msp-text-h2 text-fg">Akun</h1>
      {flow ? <KratosFlow ui={flow.ui} titled /> : <FlowUnavailable restartHref={startFlowUrl("settings", params)} />}
      {logoutUrl && (
        <p className="msp-text-body-sm">
          <a href={logoutUrl} className="text-fg-link underline">
            Keluar
          </a>
        </p>
      )}
    </>
  );
}
