const NETWORK_MAP: Record<string, string> = {
  testnet: "testnet",
  pubnet: "public",
};

const network = process.env.NEXT_PUBLIC_STELLAR_NETWORK ?? "pubnet";
export const STELLAR_EXPLORER_URL = `https://stellar.expert/explorer/${NETWORK_MAP[network] ?? "public"}/tx`;
