import createClient, { type ClientOptions } from "openapi-fetch";
import type { components, paths } from "./schema";

export type { components, paths };

/** Typed fetch client for the MesinProspek API (contracts/openapi.yaml). */
export function createApiClient(options?: ClientOptions) {
  return createClient<paths>(options);
}
