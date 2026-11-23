import { createClient } from '@mauzo/api-client';
import { auth } from './auth.svelte';
import { branch } from './branch.svelte';

/** Empty base URL = same origin (Vite proxy in dev, nginx in production). */
export const api = createClient({
  baseUrl: import.meta.env.VITE_API_URL ?? '',
  getToken: () => auth.token,
  onUnauthorized: () => auth.unauthorized(),
  // The branch picked in the top bar scopes every request (BranchScope).
  getHeaders: () => branch.header(),
});
