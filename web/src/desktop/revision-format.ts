// Formatter for Revision.timestamp values. Used by the Versions panel
// and the editor's revision banner; kept as a free function so neither
// component imports the other.
export function formatRevisionTimestamp(ts: string): string {
  const date = new Date(ts);
  if (Number.isNaN(date.getTime())) return ts;
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}
