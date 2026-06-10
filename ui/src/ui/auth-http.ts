export type AuthFetchHost = {
  settings: { token: string };
};

export function authHeaders(host: AuthFetchHost): Record<string, string> {
  const headers: Record<string, string> = { Accept: "application/json" };
  const gatewayToken = host.settings.token?.trim();
  if (gatewayToken) {
    headers.Authorization = `Bearer ${gatewayToken}`;
  }
  return headers;
}

export function authFetch(
  host: AuthFetchHost,
  url: string,
  init: RequestInit = {},
): Promise<Response> {
  const extraHeaders = init.headers instanceof Headers
    ? Object.fromEntries(init.headers.entries())
    : (init.headers as Record<string, string> | undefined) ?? {};
  return fetch(url, {
    ...init,
    credentials: "include",
    headers: {
      ...authHeaders(host),
      ...extraHeaders,
    },
  });
}
