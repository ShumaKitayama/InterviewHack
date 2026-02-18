// DesignDocument/7_BackendDesign.md 7.7.1
export interface ApiError {
  status: "error";
  code: string;
  message: string;
  detail?: Record<string, unknown>;
  requestId?: string;
}

export class ApiRequestError extends Error {
  code: string;
  requestId?: string;
  httpStatus: number;

  constructor(httpStatus: number, error: ApiError) {
    super(error.message);
    this.name = "ApiRequestError";
    this.code = error.code;
    this.requestId = error.requestId;
    this.httpStatus = httpStatus;
  }
}
