export class APIError extends Error {
  constructor(
    message: string,
    public readonly statusCode: number,
  ) {
    super(message);
    this.name = "APIError";
  }
}

export class ApiClient {
  private readonly baseUrl: string;
  private readonly apiKey: string;
  private readonly timeoutMs = 10_000;

  constructor(baseUrl: string, apiKey: string) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.apiKey = apiKey;
  }

  async get<T>(path: string): Promise<T> {
    return this.request<T>("GET", path);
  }

  async post<T>(path: string, body: Record<string, unknown>): Promise<T> {
    return this.request<T>("POST", path, body);
  }

  private async request<T>(
    method: string,
    path: string,
    body?: Record<string, unknown>,
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      Authorization: `Bearer ${this.apiKey}`,
      "Content-Type": "application/json",
    };

    const options: RequestInit = {
      method,
      headers,
      signal: AbortSignal.timeout(this.timeoutMs),
      ...(body && { body: JSON.stringify(body) }),
    };

    // First attempt
    let response = await this.fetchWithErrorHandling(url, options);

    // Retry once on 5xx
    if (response.status >= 500) {
      await this.sleep(500);
      response = await this.fetchWithErrorHandling(url, options);
      if (response.status >= 500) {
        throw new APIError("Server error. Please try again.", response.status);
      }
    }

    if (response.status === 401) {
      throw new APIError(
        "Invalid API key. Check your MANAGEMENT_BRAIN_API_KEY.",
        401,
      );
    }

    if (response.status === 429) {
      throw new APIError(
        "Board discussions are limited to once per 5 minutes.",
        429,
      );
    }

    if (response.status >= 400) {
      const errorBody = await response.json().catch(() => ({}));
      const message =
        (errorBody as Record<string, string>).error ||
        `API error (${response.status})`;
      throw new APIError(message, response.status);
    }

    const json = await response.json();

    // Normalize: extract .data if present, pass through otherwise
    if (json && typeof json === "object" && "data" in json) {
      return json.data as T;
    }
    return json as T;
  }

  private async fetchWithErrorHandling(
    url: string,
    options: RequestInit,
  ): Promise<Response> {
    try {
      return await fetch(url, options);
    } catch (error) {
      if (error instanceof DOMException && error.name === "TimeoutError") {
        throw new APIError(
          "Cloud API unreachable. Please check your network.",
          0,
        );
      }
      if (
        error instanceof TypeError &&
        error.message.includes("abort")
      ) {
        throw new APIError(
          "Cloud API unreachable. Please check your network.",
          0,
        );
      }
      throw new APIError(
        "Cloud API unreachable. Please check your network.",
        0,
      );
    }
  }

  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
