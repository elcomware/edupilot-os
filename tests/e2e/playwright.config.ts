import { defineConfig } from '@playwright/test';

const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:8080';
const host = new URL(baseURL).hostname;
if (!['localhost', '127.0.0.1', '[::1]'].includes(host) && process.env.E2E_ALLOW_REMOTE !== '1') {
  throw new Error('Use disposable schools on a local stack or set E2E_ALLOW_REMOTE=1 for a dedicated test deployment.');
}

export default defineConfig({
  testDir: './specs',
  retries: 0,
  workers: 1,
  timeout: 45_000,
  expect: {timeout: 10_000},
  forbidOnly: !!process.env.CI,
  reporter: 'list',
  use: {
    baseURL,
    browserName: 'chromium',
    launchOptions: process.env.E2E_CHROMIUM_EXECUTABLE
      ? { executablePath: process.env.E2E_CHROMIUM_EXECUTABLE }
      : undefined,
    // Request traces and screenshots can contain login credentials and bearer tokens.
    trace: 'off',
    video: 'off',
    screenshot: 'off',
  },
});
