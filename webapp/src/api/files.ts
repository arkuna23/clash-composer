import { apiRequest } from "./client";
import type { UploadFileResponse } from "./types";

export function uploadFile(
  path: string,
  file: File,
  overwrite: boolean,
): Promise<UploadFileResponse> {
  const form = new FormData();
  form.append("file", file);

  const params = new URLSearchParams({
    path,
    overwrite: String(overwrite),
  });
  return apiRequest<UploadFileResponse>(`/files?${params.toString()}`, {
    method: "POST",
    body: form,
  });
}
