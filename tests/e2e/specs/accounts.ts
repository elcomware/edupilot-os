import {expect, type APIRequestContext, type Page} from '@playwright/test';

type Slot = 'A' | 'B';
type Account = {school: string; email: string; password: string};

export function account(slot: Slot): Account {
  const required = (name: string) => {
    const key = 'E2E_' + name + '_' + slot;
    if (!process.env[key]) throw new Error('Missing ' + key + '; see tests/e2e/README.md');
    return process.env[key]!;
  };
  return {school: required('SCHOOL'), email: required('EMAIL'), password: required('PASSWORD')};
}

export async function loginForm(page: Page, user: Account) {
  await page.getByLabel('School ID', {exact: true}).fill(user.school);
  await page.getByLabel('Email', {exact: true}).fill(user.email);
  await page.getByLabel('Password', {exact: true}).fill(user.password);
  const submit = page.getByRole('button', {name: 'Sign in', exact: true});
  await submit.click();
  await expect(page.getByLabel('School ID', {exact: true})).toHaveCount(0);
}

export async function session(request: APIRequestContext, user: Account) {
  const response = await request.post('/api/v1/session', {data: user});
  expect(response.ok()).toBe(true);
  const body = await response.json();
  expect(body.principal.school).toBe(user.school);
  expect(typeof body.token).toBe('string');
  expect(body.token.length).toBeGreaterThan(0);
  return {Authorization: 'Bearer ' + body.token};
}
