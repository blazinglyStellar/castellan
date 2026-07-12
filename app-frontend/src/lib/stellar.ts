const STELLAR_NETWORK = process.env.NEXT_PUBLIC_STELLAR_NETWORK ?? "public"

const EXPLORER_URLS: Record<string, string> = {
  public: "https://stellar.expert/explorer/public/tx",
  testnet: "https://stellar.expert/explorer/testnet/tx",
  futurenet: "https://stellar.expert/explorer/futurenet/tx",
}

export function getStellarExplorerUrl(): string {
  return EXPLORER_URLS[STELLAR_NETWORK] ?? EXPLORER_URLS.public
}
