// Library files are .ds-only by design. The extension is implementation
// detail — writers think of "Act One", not "Act One.ds" — and showing it
// in the rename input invites a footgun (delete the extension, file
// disappears from the library since it no longer matches the .ds filter).
//
// These helpers strip .ds for display and re-append it on write. Folders
// are passed through untouched; callers must only invoke the file-name
// helpers when the entry is a file.

const DS_RE = /\.ds$/i;

export function displayFileName(name: string): string {
  return name.replace(DS_RE, '');
}

export function displayFilePath(path: string): string {
  if (!path.includes('/') && !path.includes('\\')) {
    return displayFileName(path);
  }
  const idx = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'));
  const dir = path.slice(0, idx + 1);
  const base = path.slice(idx + 1);
  return dir + displayFileName(base);
}

// Normalize whatever the user typed in a rename input back to a real .ds
// filename. We only strip a trailing `.ds` the user happened to type (so
// "Act One" and "Act One.ds" both land at "Act One.ds"); other dots are
// preserved verbatim. Stripping aggressively would munge legitimate
// names like "Act 2.1".
export function normalizeFileRename(input: string): string {
  const trimmed = input.trim();
  if (!trimmed) return '';
  return `${trimmed.replace(DS_RE, '')}.ds`;
}
