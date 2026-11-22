import { createClient } from '@mauzo/api-client';

/** Empty base URL = same origin (Vite proxy in dev, nginx in production). */
export const api = createClient({
  baseUrl: import.meta.env.VITE_API_URL ?? '',
});
