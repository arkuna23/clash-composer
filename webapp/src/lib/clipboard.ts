/**
 * Copy text to the clipboard with a fallback for non-secure contexts.
 *
 * navigator.clipboard is only available on HTTPS or localhost/127.0.0.1, so
 * accessing the webapp over plain HTTP from a LAN IP (e.g. 192.168.1.x) makes
 * it undefined. Fall back to a hidden textarea + document.execCommand("copy")
 * which still works in all current browsers when triggered from a user gesture.
 */
export async function copyToClipboard(text: string): Promise<void> {
  if (
    typeof navigator !== "undefined" &&
    navigator.clipboard &&
    typeof navigator.clipboard.writeText === "function" &&
    window.isSecureContext
  ) {
    await navigator.clipboard.writeText(text);
    return;
  }
  copyViaTextarea(text);
}

function copyViaTextarea(text: string): void {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  // Avoid scrolling to the bottom and visual artifacts.
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.top = "0";
  textarea.style.left = "0";
  textarea.style.width = "1px";
  textarea.style.height = "1px";
  textarea.style.padding = "0";
  textarea.style.border = "none";
  textarea.style.outline = "none";
  textarea.style.boxShadow = "none";
  textarea.style.background = "transparent";
  textarea.style.opacity = "0";

  document.body.appendChild(textarea);
  const selection = document.getSelection();
  const previousRange =
    selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null;

  textarea.select();
  textarea.setSelectionRange(0, text.length);

  let succeeded = false;
  try {
    succeeded = document.execCommand("copy");
  } catch {
    succeeded = false;
  } finally {
    document.body.removeChild(textarea);
    if (previousRange && selection) {
      selection.removeAllRanges();
      selection.addRange(previousRange);
    }
  }

  if (!succeeded) {
    throw new Error("clipboard copy is not supported in this context");
  }
}
