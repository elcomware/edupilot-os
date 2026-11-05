import test from 'node:test';
import assert from 'node:assert/strict';
import { apiRoot } from '../src/endpoint.ts';

test('uses a deployment origin and supplies the shared API version', () => {
  assert.equal(
    apiRoot('https://example.test/', false),
    'https://example.test/api/v1',
  );
});
test('never accepts insecure production transports or credentials in the endpoint', () => {
  for (const origin of [
    'http://127.0.0.1:3000',
    'http://school.example',
    'https://bearer@school.example',
    'https://school.example/private',
    'https://school.example/?debug',
    'https://school.example/#token',
    undefined,
  ]) {
    assert.equal(apiRoot(origin, false), null);
  }
  assert.equal(
    apiRoot('http://10.0.2.2:3000', true),
    'http://10.0.2.2:3000/api/v1',
  );
  assert.equal(apiRoot('http://school.example', true), null);
});
