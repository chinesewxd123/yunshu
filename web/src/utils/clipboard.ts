/** 复制到剪贴板：优先 Clipboard API，HTTP 非安全上下文时回退到 execCommand（与复制命令按钮兼容）。 */
export async function copyToClipboard(text: string): Promise<void> {
  const t = String(text ?? "");
  if (!t) return;

  if (typeof navigator !== "undefined" && navigator.clipboard?.writeText && window.isSecureContext) {
    await navigator.clipboard.writeText(t);
    return;
  }

  const ta = document.createElement("textarea");
  ta.value = t;
  ta.setAttribute("readonly", "");
  ta.style.position = "fixed";
  ta.style.left = "-9999px";
  ta.style.top = "0";
  document.body.appendChild(ta);
  ta.select();
  ta.setSelectionRange(0, t.length);
  try {
    const ok = document.execCommand("copy");
    if (!ok) {
      throw new Error("execCommand copy returned false");
    }
  } finally {
    document.body.removeChild(ta);
  }
}
