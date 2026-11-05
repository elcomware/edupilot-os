import {randomUUID} from 'node:crypto';
import {expect, test} from '@playwright/test';
import {account, session} from './accounts';

test('another school cannot enumerate, read, change or archive a student', async ({request}) => {
  const a = account('A'), b = account('B');
  expect(a.school).not.toBe(b.school);
  const [owner, stranger] = await Promise.all([session(request, a), session(request, b)]);
  const givenName = 'River', familyName = 'Isolation ' + randomUUID().slice(0, 8);
  const created = await request.post('/api/v1/students', {headers: owner, data: {givenName, familyName}});
  expect(created.status()).toBe(201);
  const student = await created.json();
  const url = '/api/v1/students/' + encodeURIComponent(student.id);
  try {
    const before = await request.get(url, {headers: owner});
    expect(before.status()).toBe(200);
    expect((await before.json()).id).toBe(student.id);
    expect((await request.get(url)).status()).toBe(401);
    // Iterate every page; checking just page 1 can hide a scope leak.
    for (let offset = 0; ; offset += 100) {
      const roster = await request.get('/api/v1/students', {headers: stranger, params: {limit: 100, offset}});
      expect(roster.status()).toBe(200);
      const {items, total} = await roster.json();
      expect(items.map((s: {id: string}) => s.id)).not.toContain(student.id);
      if (offset + 100 >= total) break;
    }
    expect((await request.get(url, {headers: stranger})).status()).toBe(404);
    expect((await request.patch(url, {headers: stranger, data: {givenName: 'Changed', familyName, version: student.version}})).status()).toBe(404);
    expect((await request.delete(url, {headers: stranger, params: {version: student.version}})).status()).toBe(404);
    expect((await request.get(url, {headers: {...stranger, 'X-School-ID': a.school}})).status()).toBe(403);
    const after = await request.get(url, {headers: owner});
    expect(after.status()).toBe(200);
    expect(await after.json()).toMatchObject({givenName, familyName, status: 'active', version: 1});
  } finally {
    // Archive only the row this test created and only if its version is unchanged.
    const cleanup = await request.delete(url, {headers: owner, params: {version: student.version}});
    expect(cleanup.status()).toBe(204);
  }
});
