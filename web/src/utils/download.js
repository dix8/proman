export function parseContentDispositionFilename(contentDisposition) {
  if (!contentDisposition) {
    return "";
  }

  const filenameStarMatch = contentDisposition.match(
    /filename\*=UTF-8''([^;]+)/i,
  );
  if (filenameStarMatch?.[1]) {
    try {
      return decodeURIComponent(filenameStarMatch[1]);
    } catch {
      return filenameStarMatch[1];
    }
  }

  const filenameMatch =
    contentDisposition.match(/filename="([^"]+)"/i) ||
    contentDisposition.match(/filename=([^;]+)/i);
  if (filenameMatch?.[1]) {
    return filenameMatch[1].trim();
  }

  return "";
}

export function triggerBlobDownload(blob, filename) {
  if (!Array.isArray(window.__promanDownloads)) {
    window.__promanDownloads = [];
  }
  window.__promanDownloads.push({ filename });

  const objectUrl = window.URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = objectUrl;
  if (filename) {
    anchor.download = filename;
  }
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  window.URL.revokeObjectURL(objectUrl);
}
