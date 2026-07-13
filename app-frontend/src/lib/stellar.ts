const STELLAR_NETWORK = process.env.NEXT_PUBLIC_STELLAR_NETWORK ?? "public"

const EXPLORER_URLS: Record<string, string> = {
  public: "https://stellar.expert/explorer/public/tx",
  testnet: "https://stellar.expert/explorer/testnet/tx",
  futurenet: "https://stellar.expert/explorer/futurenet/tx",
}

const NETWORK_LABELS: Record<string, string> = {
  public: "Mainnet",
  testnet: "Testnet",
  futurenet: "Futurenet",
}

export function getStellarExplorerUrl(): string {
  return EXPLORER_URLS[STELLAR_NETWORK] ?? EXPLORER_URLS.public
}

export function getStellarNetworkLabel(): string {
  return NETWORK_LABELS[STELLAR_NETWORK] ?? STELLAR_NETWORK
}
