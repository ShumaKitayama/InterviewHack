// DesignDocument/7_BackendDesign.md 7.7.3
export function generateRequestId(): string {
  return crypto.randomUUID();
}
