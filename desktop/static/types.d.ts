// Type declarations for the local renderer pages (checked by tsconfig.renderer.json).

interface SetupInfo {
  serverUrl: string | null;
  firstRun: boolean;
  version: string;
}

interface SetupResult {
  ok: boolean;
  url?: string;
  error?: string;
  canForce?: boolean;
}

interface Window {
  ccSetup: {
    info(): Promise<SetupInfo>;
    check(url: string): Promise<SetupResult>;
    save(url: string, force: boolean): Promise<SetupResult>;
    cancel(): void;
  };
}
