import {randomUUID} from 'node:crypto';
import {expect, test} from '@playwright/test';
import {account, loginForm, session} from './accounts';

test('an admin registers, edits and archives a student through the web app', async ({page, request}) => {
  const user = account('A');
  const family = 'Lifecycle ' + randomUUID().slice(0, 8);
  await page.goto('/#/modules/students');
  await loginForm(page, user);
  await page.getByRole('button', {name: 'Add student', exact: true}).click();
  await page.getByLabel('Given name', {exact: true}).fill('Jamie');
  await page.getByLabel('Family name', {exact: true}).fill(family);
  const registered = page.waitForResponse(r =>
    new URL(r.url()).pathname === '/api/v1/students' && r.request().method() === 'POST');
  await page.getByRole('button', {name: 'Save student', exact: true}).click();
  const response = await registered;
  expect(response.status()).toBe(201);
  const {id} = await response.json();
  const endpoint = '/api/v1/students/' + encodeURIComponent(id);
  const edit = (name: string) => page.getByRole('button', {name: 'Edit ' + name, exact: true});
  await edit('Jamie ' + family).click();
  await page.getByLabel('Given name', {exact: true}).fill('Jules');
  await page.getByRole('button', {name: 'Save changes', exact: true}).click();
  await expect(edit('Jules ' + family)).toBeVisible();

  // A new browser load must recover the saved record from the real backend.
  await page.reload();
  await loginForm(page, user);
  await expect(edit('Jules ' + family)).toBeVisible();

  await page.getByRole('button', {name: 'Archive Jules ' + family, exact: true}).click();
  await page.getByRole('button', {name: 'Archive student', exact: true}).click();
  await expect(edit('Jules ' + family)).toHaveCount(0);

  await page.reload();
  const rosterAfterLogin = page.waitForResponse(r =>
    new URL(r.url()).pathname === '/api/v1/students' && r.request().method() === 'GET');
  await loginForm(page, user);
  // Check the server's roster before checking the absence in the reloaded UI.
  const roster = await rosterAfterLogin;
  expect(roster.status()).toBe(200);
  expect((await roster.json()).items.map((s: {id: string}) => s.id)).not.toContain(id);
  await expect(edit('Jules ' + family)).toHaveCount(0);
  const headers = await session(request, user);
  const persisted = await request.get(endpoint, {headers});
  expect(persisted.status()).toBe(200);
  expect(await persisted.json()).toMatchObject({id, givenName: 'Jules', familyName: family, status: 'archived', version: 3});
});

test('appearance and density survive reload and sign-in for the same administrator', async ({page}) => {
  const user = account('A');
  await page.goto('/#/settings');
  await loginForm(page, user);
  const screen = page.locator('html');
  await page.getByLabel('Appearance', {exact: true}).selectOption('dark');
  await page.getByLabel('Density', {exact: true}).selectOption('compact');
  await page.getByLabel('Accent', {exact: true}).selectOption('indigo');
  await expect(screen).toHaveAttribute('data-theme', 'dark');
  await expect(screen).toHaveAttribute('data-density', 'compact');
  await expect(screen).toHaveAttribute('data-accent', 'indigo');
  await page.reload();
  await loginForm(page, user);
  await expect(page.getByLabel('Appearance', {exact: true})).toHaveValue('dark');
  await expect(page.getByLabel('Density', {exact: true})).toHaveValue('compact');
  await expect(page.getByLabel('Accent', {exact: true})).toHaveValue('indigo');

  // Same browser, second real school: settings are scoped to principal and school.
  await page.getByRole('button', {name: 'Sign out', exact: true}).click();
  await loginForm(page, account('B'));
  await expect(page.getByLabel('Appearance', {exact: true})).toHaveValue('system');
  await expect(page.getByLabel('Density', {exact: true})).toHaveValue('comfortable');
  await expect(page.getByLabel('Accent', {exact: true})).toHaveValue('pine');
});
