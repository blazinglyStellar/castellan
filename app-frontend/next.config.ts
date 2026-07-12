import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  env: {
    NEXT_PUBLIC_STELLAR_NETWORK: process.env.STELLAR_NETWORK || "testnet",
  },
};

export default nextConfig;
