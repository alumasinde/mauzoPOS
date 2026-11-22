const TZ = 'Africa/Nairobi';

/** e.g. "Tue, 27 May 2025" */
export function formatDate(d: Date | string, timeZone = TZ): string {
  return new Intl.DateTimeFormat('en-KE', {
    weekday: 'short',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    timeZone,
  }).format(new Date(d));
}

/** e.g. "10:24:37 AM" */
export function formatTime(d: Date | string, timeZone = TZ): string {
  return new Intl.DateTimeFormat('en-KE', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: true,
    timeZone,
  }).format(new Date(d));
}
