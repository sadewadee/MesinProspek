import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Standalone output keeps the production image small (docs/06 build job).
  output: "standalone",
};

export default nextConfig;
